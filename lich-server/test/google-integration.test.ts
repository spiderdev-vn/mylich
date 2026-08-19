import { test, describe } from 'node:test';
import assert from 'node:assert/strict';
import { createTestApp, registerAndLoginUser } from './setup.ts';
import { GoogleEventMapper } from '../src/integrations/google/google.mapper.ts';

describe('Google Calendar Integration Tests', () => {
  describe('GoogleEventMapper', () => {
    test('should correctly map Lich event to Google event format', () => {
      const lichEvent: any = {
        id: 'evt-1',
        title: 'Standup',
        description: 'Daily standup',
        start_at: '2026-08-20T10:00:00.000Z',
        end_at: '2026-08-20T10:30:00.000Z',
        timezone: 'Asia/Ho_Chi_Minh',
        location: 'Room A',
      };

      const gEvent = GoogleEventMapper.toGoogle(lichEvent);
      assert.equal(gEvent.summary, 'Standup');
      assert.equal(gEvent.description, 'Daily standup');
      assert.equal(gEvent.location, 'Room A');
      assert.equal(gEvent.start?.dateTime, '2026-08-20T10:00:00.000Z');
      assert.equal(gEvent.start?.timeZone, 'Asia/Ho_Chi_Minh');
    });

    test('should correctly map Google event to Lich event format', () => {
      const gEvent: any = {
        id: 'g-123',
        summary: 'Planning',
        description: 'Sprint planning',
        location: 'Zoom',
        start: { dateTime: '2026-08-21T14:00:00+07:00', timeZone: 'Asia/Ho_Chi_Minh' },
        end: { dateTime: '2026-08-21T15:00:00+07:00', timeZone: 'Asia/Ho_Chi_Minh' },
      };

      const lich = GoogleEventMapper.toLich(gEvent, 'cal-1');
      assert.equal(lich.title, 'Planning');
      assert.equal(lich.description, 'Sprint planning');
      assert.equal(lich.location, 'Zoom');
      assert.equal(lich.timezone, 'Asia/Ho_Chi_Minh');
      assert.equal(new Date(lich.start_at).toISOString(), new Date('2026-08-21T14:00:00+07:00').toISOString());
    });

    test('should correctly map all-day Google event (date string)', () => {
      const gEvent: any = {
        id: 'g-allday',
        summary: 'Holiday',
        start: { date: '2026-09-02' },
        end: { date: '2026-09-02' },
      };

      const lich = GoogleEventMapper.toLich(gEvent, 'cal-1');
      assert.equal(lich.title, 'Holiday');
      assert.equal(lich.start_at, '2026-09-02T00:00:00.000Z');
    });
  });

  describe('Integration REST Routes & Sync Lifecycle', () => {
    test('should get Google OAuth auth URL', async () => {
      const app = await createTestApp();
      const auth = await registerAndLoginUser(app, 'guser1', 'password123');

      const res = await app.inject({
        method: 'GET',
        url: '/api/v1/integrations/google/auth-url',
        headers: { Authorization: `Bearer ${auth.token}` },
      });

      assert.equal(res.statusCode, 200);
      const body = JSON.parse(res.payload);
      assert.ok(body.auth_url);
      assert.ok(body.auth_url.includes('callback'));
      await app.close();
    });

    test('should handle OAuth callback and report status connected', async () => {
      const app = await createTestApp();
      const auth = await registerAndLoginUser(app, 'guser2', 'password123');

      // 1. Initial status: disconnected
      const initialStatusRes = await app.inject({
        method: 'GET',
        url: '/api/v1/integrations/google/status',
        headers: { Authorization: `Bearer ${auth.token}` },
      });
      assert.equal(initialStatusRes.statusCode, 200);
      assert.equal(JSON.parse(initialStatusRes.payload).connected, false);

      // 2. Fetch auth-url to get state
      const authUrlRes = await app.inject({
        method: 'GET',
        url: '/api/v1/integrations/google/auth-url',
        headers: { Authorization: `Bearer ${auth.token}` },
      });
      const authUrl = JSON.parse(authUrlRes.payload).auth_url;
      const urlObj = new URL(authUrl);
      const state = urlObj.searchParams.get('state');

      // 3. Callback redirect simulation
      const callbackRes = await app.inject({
        method: 'GET',
        url: `/api/v1/auth/google/callback?code=mock-code-123&state=${state}`,
      });
      assert.equal(callbackRes.statusCode, 200);
      assert.ok(callbackRes.headers['content-type']?.includes('text/html'));

      // 4. Status is now connected with auto-mapped default calendar
      const statusRes = await app.inject({
        method: 'GET',
        url: '/api/v1/integrations/google/status',
        headers: { Authorization: `Bearer ${auth.token}` },
      });
      assert.equal(statusRes.statusCode, 200);
      const status = JSON.parse(statusRes.payload);
      assert.equal(status.connected, true);
      assert.ok(status.mappedCalendars.length > 0);
      assert.equal(status.mappedCalendars[0].externalCalendarId, 'primary');
      await app.close();
    });

    test('should list external calendars and update mapping', async () => {
      const app = await createTestApp();
      const auth = await registerAndLoginUser(app, 'guser3', 'password123');

      // Connect first
      const authUrlRes = await app.inject({
        method: 'GET',
        url: '/api/v1/integrations/google/auth-url',
        headers: { Authorization: `Bearer ${auth.token}` },
      });
      const state = new URL(JSON.parse(authUrlRes.payload).auth_url).searchParams.get('state');
      await app.inject({ method: 'GET', url: `/api/v1/auth/google/callback?code=mock-code&state=${state}` });

      // List external calendars
      const calListRes = await app.inject({
        method: 'GET',
        url: '/api/v1/integrations/google/calendars',
        headers: { Authorization: `Bearer ${auth.token}` },
      });
      assert.equal(calListRes.statusCode, 200);
      const cals = JSON.parse(calListRes.payload).calendars;
      assert.ok(cals.length >= 1);

      // Get user's default calendar
      const userCalsRes = await app.inject({
        method: 'GET',
        url: '/api/v1/calendars',
        headers: { Authorization: `Bearer ${auth.token}` },
      });
      const userCalId = JSON.parse(userCalsRes.payload)[0].id;

      // Update mapping
      const mapRes = await app.inject({
        method: 'POST',
        url: '/api/v1/integrations/google/map',
        headers: { Authorization: `Bearer ${auth.token}` },
        payload: {
          calendar_id: userCalId,
          external_calendar_id: 'work@example.com',
          sync_direction: 'bidirectional',
        },
      });
      assert.equal(mapRes.statusCode, 200);
      assert.equal(JSON.parse(mapRes.payload).success, true);
      await app.close();
    });

    test('should sync events bidirectional with Google', async () => {
      const app = await createTestApp();
      const auth = await registerAndLoginUser(app, 'guser4', 'password123');

      // 1. Connect
      const authUrlRes = await app.inject({
        method: 'GET',
        url: '/api/v1/integrations/google/auth-url',
        headers: { Authorization: `Bearer ${auth.token}` },
      });
      const state = new URL(JSON.parse(authUrlRes.payload).auth_url).searchParams.get('state');
      await app.inject({ method: 'GET', url: `/api/v1/auth/google/callback?code=mock-code&state=${state}` });

      // 2. Create a local event in Lich
      const userCalsRes = await app.inject({
        method: 'GET',
        url: '/api/v1/calendars',
        headers: { Authorization: `Bearer ${auth.token}` },
      });
      const userCalId = JSON.parse(userCalsRes.payload)[0].id;

      await app.inject({
        method: 'POST',
        url: '/api/v1/events',
        headers: { Authorization: `Bearer ${auth.token}` },
        payload: {
          calendar_id: userCalId,
          title: 'Họp Google Sync',
          start_at: '2026-08-20T10:00:00.000Z',
          end_at: '2026-08-20T11:00:00.000Z',
          timezone: 'Asia/Ho_Chi_Minh',
        },
      });

      // 3. Trigger Sync
      const syncRes = await app.inject({
        method: 'POST',
        url: '/api/v1/integrations/google/sync',
        headers: { Authorization: `Bearer ${auth.token}` },
        payload: { direction: 'both' },
      });

      assert.equal(syncRes.statusCode, 200);
      const syncResult = JSON.parse(syncRes.payload);
      assert.equal(syncResult.success, true);
      assert.equal(syncResult.pushed, 1);

      // 4. Disconnect
      const disconnectRes = await app.inject({
        method: 'DELETE',
        url: '/api/v1/integrations/google',
        headers: { Authorization: `Bearer ${auth.token}` },
      });
      assert.equal(disconnectRes.statusCode, 200);

      const statusAfter = await app.inject({
        method: 'GET',
        url: '/api/v1/integrations/google/status',
        headers: { Authorization: `Bearer ${auth.token}` },
      });
      assert.equal(JSON.parse(statusAfter.payload).connected, false);
      await app.close();
    });

    test('should create a new calendar on Google and map it', async () => {
      const app = await createTestApp();
      const auth = await registerAndLoginUser(app, 'guser5', 'password123');

      // 1. Connect
      const authUrlRes = await app.inject({
        method: 'GET',
        url: '/api/v1/integrations/google/auth-url',
        headers: { Authorization: `Bearer ${auth.token}` },
      });
      const state = new URL(JSON.parse(authUrlRes.payload).auth_url).searchParams.get('state');
      await app.inject({ method: 'GET', url: `/api/v1/auth/google/callback?code=mock-code&state=${state}` });

      // 2. Create a secondary Lich calendar
      const calRes = await app.inject({
        method: 'POST',
        url: '/api/v1/calendars',
        headers: { Authorization: `Bearer ${auth.token}` },
        payload: { name: 'Projects & Work', timezone: 'Asia/Ho_Chi_Minh' },
      });
      const newCal = JSON.parse(calRes.payload);

      // 3. Create on Google and map
      const createGoogleRes = await app.inject({
        method: 'POST',
        url: '/api/v1/integrations/google/create-calendar',
        headers: { Authorization: `Bearer ${auth.token}` },
        payload: { calendar_id: newCal.id, name: 'Projects & Work (Google)' },
      });
      assert.equal(createGoogleRes.statusCode, 201);
      const resPayload = JSON.parse(createGoogleRes.payload);
      assert.equal(resPayload.success, true);
      assert.ok(resPayload.external_calendar.id);
      assert.equal(resPayload.external_calendar.name, 'Projects & Work (Google)');

      // 4. Verify in status mapped calendars
      const statusRes = await app.inject({
        method: 'GET',
        url: '/api/v1/integrations/google/status',
        headers: { Authorization: `Bearer ${auth.token}` },
      });
      const status = JSON.parse(statusRes.payload);
      const mapped = status.mappedCalendars.find((m: any) => m.calendarId === newCal.id);
      assert.ok(mapped);
      assert.equal(mapped.externalCalendarId, resPayload.external_calendar.id);

      await app.close();
    });
  });
});


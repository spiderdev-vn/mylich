import { test, describe } from 'node:test';
import assert from 'node:assert/strict';
import { createTestApp } from './setup.ts';

describe('Event Endpoints', () => {
  test('creates, reads, updates, and deletes an event', async () => {
    const app = await createTestApp();
    const regRes = await app.inject({
      method: 'POST',
      url: '/auth/register',
      payload: { username: 'eventuser', password: 'password123' },
    });
    const { token } = JSON.parse(regRes.body);
    const authHeaders = { authorization: `Bearer ${token}` };

    // 1. Create event
    const createRes = await app.inject({
      method: 'POST',
      url: '/events',
      headers: authHeaders,
      payload: {
        title: 'Team Sync',
        description: 'Weekly team catch-up',
        start_at: '2026-08-18T10:00:00+07:00',
        end_at: '2026-08-18T11:00:00+07:00',
        location: 'Room 101',
        timezone: 'Asia/Ho_Chi_Minh',
      },
    });

    assert.equal(createRes.statusCode, 201);
    const event = JSON.parse(createRes.body);
    assert.equal(event.title, 'Team Sync');
    assert.equal(event.location, 'Room 101');
    assert.ok(event.calendar_id);

    // 2. Get event by id
    const getRes = await app.inject({
      method: 'GET',
      url: `/events/${event.id}`,
      headers: authHeaders,
    });
    assert.equal(getRes.statusCode, 200);
    const fetched = JSON.parse(getRes.body);
    assert.equal(fetched.id, event.id);

    // 3. Update event
    const updateRes = await app.inject({
      method: 'PATCH',
      url: `/events/${event.id}`,
      headers: authHeaders,
      payload: {
        title: 'Updated Team Sync',
        location: 'Zoom Room',
      },
    });
    assert.equal(updateRes.statusCode, 200);
    const updated = JSON.parse(updateRes.body);
    assert.equal(updated.title, 'Updated Team Sync');
    assert.equal(updated.location, 'Zoom Room');

    // 4. Delete event
    const deleteRes = await app.inject({
      method: 'DELETE',
      url: `/events/${event.id}`,
      headers: authHeaders,
    });
    assert.equal(deleteRes.statusCode, 204);

    // 5. Verify deleted
    const verifyGet = await app.inject({
      method: 'GET',
      url: `/events/${event.id}`,
      headers: authHeaders,
    });
    assert.equal(verifyGet.statusCode, 404);
  });

  test('validates event dates: rejects end_at <= start_at', async () => {
    const app = await createTestApp();
    const regRes = await app.inject({
      method: 'POST',
      url: '/auth/register',
      payload: { username: 'datevaluser', password: 'password123' },
    });
    const { token } = JSON.parse(regRes.body);

    const res = await app.inject({
      method: 'POST',
      url: '/events',
      headers: { authorization: `Bearer ${token}` },
      payload: {
        title: 'Invalid Time Event',
        start_at: '2026-08-18T12:00:00Z',
        end_at: '2026-08-18T11:00:00Z',
      },
    });

    assert.equal(res.statusCode, 400);
    const body = JSON.parse(res.body);
    assert.equal(body.error, 'BAD_REQUEST');
  });

  test('filters events by date range and orders by start_at ASC', async () => {
    const app = await createTestApp();
    const regRes = await app.inject({
      method: 'POST',
      url: '/auth/register',
      payload: { username: 'rangeuser', password: 'password123' },
    });
    const { token } = JSON.parse(regRes.body);
    const authHeaders = { authorization: `Bearer ${token}` };

    // Create 3 events on different days
    await app.inject({
      method: 'POST',
      url: '/events',
      headers: authHeaders,
      payload: {
        title: 'Event Aug 18 Evening',
        start_at: '2026-08-18T19:00:00Z',
        end_at: '2026-08-18T20:00:00Z',
      },
    });
    await app.inject({
      method: 'POST',
      url: '/events',
      headers: authHeaders,
      payload: {
        title: 'Event Aug 18 Morning',
        start_at: '2026-08-18T09:00:00Z',
        end_at: '2026-08-18T10:00:00Z',
      },
    });
    await app.inject({
      method: 'POST',
      url: '/events',
      headers: authHeaders,
      payload: {
        title: 'Event Aug 25',
        start_at: '2026-08-25T10:00:00Z',
        end_at: '2026-08-25T11:00:00Z',
      },
    });

    // Query only Aug 18
    const queryRes = await app.inject({
      method: 'GET',
      url: '/events?from=2026-08-18T00:00:00Z&to=2026-08-18T23:59:59Z',
      headers: authHeaders,
    });

    assert.equal(queryRes.statusCode, 200);
    const events = JSON.parse(queryRes.body);
    assert.equal(events.length, 2);
    // Verified sorted ascending
    assert.equal(events[0].title, 'Event Aug 18 Morning');
    assert.equal(events[1].title, 'Event Aug 18 Evening');
  });
});

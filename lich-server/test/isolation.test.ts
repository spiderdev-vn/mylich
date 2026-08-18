import { test, describe } from 'node:test';
import assert from 'node:assert/strict';
import { createTestApp } from './setup.ts';

describe('Multi-Tenant Isolation', () => {
  test('User A cannot access or see User B calendars and events', async () => {
    const app = await createTestApp();

    // 1. Create User A
    const resA = await app.inject({
      method: 'POST',
      url: '/auth/register',
      payload: { username: 'user_a', password: 'password123' },
    });
    const tokenA = JSON.parse(resA.body).token;
    const headersA = { authorization: `Bearer ${tokenA}` };

    // 2. Create User B
    const resB = await app.inject({
      method: 'POST',
      url: '/auth/register',
      payload: { username: 'user_b', password: 'password123' },
    });
    const tokenB = JSON.parse(resB.body).token;
    const headersB = { authorization: `Bearer ${tokenB}` };

    // 3. User A creates custom calendar & event
    const calResA = await app.inject({
      method: 'POST',
      url: '/calendars',
      headers: headersA,
      payload: { name: 'Secret A Calendar' },
    });
    const calA = JSON.parse(calResA.body);

    const eventResA = await app.inject({
      method: 'POST',
      url: '/events',
      headers: headersA,
      payload: {
        calendar_id: calA.id,
        title: 'Secret A Event',
        start_at: '2026-08-18T10:00:00Z',
        end_at: '2026-08-18T11:00:00Z',
      },
    });
    const eventA = JSON.parse(eventResA.body);

    // 4. User B tries to GET User A's calendar by ID -> 404
    const getCalB = await app.inject({
      method: 'GET',
      url: `/calendars/${calA.id}`,
      headers: headersB,
    });
    assert.equal(getCalB.statusCode, 404);

    // 5. User B tries to GET User A's event by ID -> 404
    const getEventB = await app.inject({
      method: 'GET',
      url: `/events/${eventA.id}`,
      headers: headersB,
    });
    assert.equal(getEventB.statusCode, 404);

    // 6. User B lists events -> should not see User A's event
    const listEventsB = await app.inject({
      method: 'GET',
      url: '/events',
      headers: headersB,
    });
    assert.equal(listEventsB.statusCode, 200);
    const eventsB = JSON.parse(listEventsB.body);
    assert.equal(eventsB.length, 0);

    // 7. User B tries to update User A's event -> 404
    const updateEventB = await app.inject({
      method: 'PATCH',
      url: `/events/${eventA.id}`,
      headers: headersB,
      payload: { title: 'Hacked Title' },
    });
    assert.equal(updateEventB.statusCode, 404);

    // 8. User B tries to delete User A's event -> 404
    const deleteEventB = await app.inject({
      method: 'DELETE',
      url: `/events/${eventA.id}`,
      headers: headersB,
    });
    assert.equal(deleteEventB.statusCode, 404);
  });
});

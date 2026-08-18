import { test, describe } from 'node:test';
import assert from 'node:assert/strict';
import { createTestApp } from './setup.ts';

describe('Calendar Endpoints', () => {
  test('CRUD operations for calendars', async () => {
    const app = await createTestApp();
    const regRes = await app.inject({
      method: 'POST',
      url: '/auth/register',
      payload: { username: 'caluser', password: 'password123' },
    });
    const { token } = JSON.parse(regRes.body);
    const authHeaders = { authorization: `Bearer ${token}` };

    // 1. Create a work calendar
    const createRes = await app.inject({
      method: 'POST',
      url: '/calendars',
      headers: authHeaders,
      payload: {
        name: 'Work',
        description: 'Work related schedule',
        timezone: 'America/New_York',
      },
    });
    assert.equal(createRes.statusCode, 201);
    const createdCal = JSON.parse(createRes.body);
    assert.equal(createdCal.name, 'Work');
    assert.equal(createdCal.timezone, 'America/New_York');

    // 2. List calendars (should have 2: Personal + Work)
    const listRes = await app.inject({
      method: 'GET',
      url: '/calendars',
      headers: authHeaders,
    });
    assert.equal(listRes.statusCode, 200);
    const list = JSON.parse(listRes.body);
    assert.equal(list.length, 2);

    // 3. Get single calendar
    const getRes = await app.inject({
      method: 'GET',
      url: `/calendars/${createdCal.id}`,
      headers: authHeaders,
    });
    assert.equal(getRes.statusCode, 200);
    const fetched = JSON.parse(getRes.body);
    assert.equal(fetched.id, createdCal.id);

    // 4. Update calendar
    const updateRes = await app.inject({
      method: 'PATCH',
      url: `/calendars/${createdCal.id}`,
      headers: authHeaders,
      payload: {
        name: 'Work Updated',
        description: 'Updated description',
      },
    });
    assert.equal(updateRes.statusCode, 200);
    const updated = JSON.parse(updateRes.body);
    assert.equal(updated.name, 'Work Updated');
    assert.equal(updated.description, 'Updated description');

    // 5. Delete calendar
    const delRes = await app.inject({
      method: 'DELETE',
      url: `/calendars/${createdCal.id}`,
      headers: authHeaders,
    });
    assert.equal(delRes.statusCode, 204);

    // 6. Verify deleted
    const verifyGet = await app.inject({
      method: 'GET',
      url: `/calendars/${createdCal.id}`,
      headers: authHeaders,
    });
    assert.equal(verifyGet.statusCode, 404);
  });

  test('rejects invalid IANA timezone', async () => {
    const app = await createTestApp();
    const regRes = await app.inject({
      method: 'POST',
      url: '/auth/register',
      payload: { username: 'tzuser', password: 'password123' },
    });
    const { token } = JSON.parse(regRes.body);

    const res = await app.inject({
      method: 'POST',
      url: '/calendars',
      headers: { authorization: `Bearer ${token}` },
      payload: {
        name: 'Bad TZ',
        timezone: 'Mars/Phobos',
      },
    });

    assert.equal(res.statusCode, 400);
    const body = JSON.parse(res.body);
    assert.equal(body.error, 'BAD_REQUEST');
  });
});

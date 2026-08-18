import { test, describe } from 'node:test';
import assert from 'node:assert/strict';
import { createTestApp } from './setup.ts';

describe('Auth Endpoints', () => {
  test('registers a new user and auto-creates Personal calendar', async () => {
    const app = await createTestApp();
    const res = await app.inject({
      method: 'POST',
      url: '/auth/register',
      payload: {
        username: 'alice',
        password: 'password123',
        timezone: 'Asia/Ho_Chi_Minh',
      },
    });

    assert.equal(res.statusCode, 201);
    const body = JSON.parse(res.body);
    assert.ok(body.token);
    assert.equal(body.user.username, 'alice');

    // Verify calendar was auto-created
    const calRes = await app.inject({
      method: 'GET',
      url: '/calendars',
      headers: {
        authorization: `Bearer ${body.token}`,
      },
    });
    assert.equal(calRes.statusCode, 200);
    const calendars = JSON.parse(calRes.body);
    assert.equal(calendars.length, 1);
    assert.equal(calendars[0].name, 'Personal');
    assert.equal(calendars[0].timezone, 'Asia/Ho_Chi_Minh');
    assert.equal(calendars[0].is_default, true);
  });

  test('rejects duplicate username registration', async () => {
    const app = await createTestApp();
    await app.inject({
      method: 'POST',
      url: '/auth/register',
      payload: { username: 'bob', password: 'password123' },
    });

    const res = await app.inject({
      method: 'POST',
      url: '/auth/register',
      payload: { username: 'bob', password: 'differentpassword' },
    });

    assert.equal(res.statusCode, 409);
    const body = JSON.parse(res.body);
    assert.equal(body.error, 'CONFLICT');
  });

  test('login with valid credentials returns token', async () => {
    const app = await createTestApp();
    await app.inject({
      method: 'POST',
      url: '/auth/register',
      payload: { username: 'charlie', password: 'securepassword' },
    });

    const res = await app.inject({
      method: 'POST',
      url: '/auth/login',
      payload: { username: 'charlie', password: 'securepassword' },
    });

    assert.equal(res.statusCode, 200);
    const body = JSON.parse(res.body);
    assert.ok(body.token);
    assert.equal(body.user.username, 'charlie');
  });

  test('login with invalid password returns 401', async () => {
    const app = await createTestApp();
    await app.inject({
      method: 'POST',
      url: '/auth/register',
      payload: { username: 'david', password: 'mypassword' },
    });

    const res = await app.inject({
      method: 'POST',
      url: '/auth/login',
      payload: { username: 'david', password: 'wrongpassword' },
    });

    assert.equal(res.statusCode, 401);
    const body = JSON.parse(res.body);
    assert.equal(body.error, 'UNAUTHORIZED');
  });

  test('GET /auth/me returns current user with valid token', async () => {
    const app = await createTestApp();
    const regRes = await app.inject({
      method: 'POST',
      url: '/auth/register',
      payload: { username: 'eve', password: 'password123' },
    });
    const { token } = JSON.parse(regRes.body);

    const res = await app.inject({
      method: 'GET',
      url: '/auth/me',
      headers: { authorization: `Bearer ${token}` },
    });

    assert.equal(res.statusCode, 200);
    const body = JSON.parse(res.body);
    assert.equal(body.user.username, 'eve');
  });

  test('GET /auth/me without token returns 401', async () => {
    const app = await createTestApp();
    const res = await app.inject({
      method: 'GET',
      url: '/auth/me',
    });

    assert.equal(res.statusCode, 401);
  });
});

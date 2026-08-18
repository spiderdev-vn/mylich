import { test, describe } from 'node:test';
import assert from 'node:assert/strict';
import { createTestApp } from './setup.ts';

describe('GET /health', () => {
  test('returns status ok', async () => {
    const app = await createTestApp();
    const res = await app.inject({
      method: 'GET',
      url: '/health',
    });

    assert.equal(res.statusCode, 200);
    const body = JSON.parse(res.body);
    assert.deepEqual(body, { status: 'ok' });
  });
});

import { test, describe } from 'node:test';
import assert from 'node:assert/strict';
import { createTestApp } from './setup.ts';

describe('GET /health & /api/v1/health', () => {
  test('returns status ok at root and api v1', async () => {
    const app = await createTestApp();
    const res = await app.inject({
      method: 'GET',
      url: '/health',
    });

    assert.equal(res.statusCode, 200);
    const body = JSON.parse(res.body);
    assert.deepEqual(body, { status: 'ok' });

    const v1Res = await app.inject({
      method: 'GET',
      url: '/api/v1/health',
    });
    assert.equal(v1Res.statusCode, 200);
    assert.deepEqual(JSON.parse(v1Res.body), { status: 'ok' });
  });
});

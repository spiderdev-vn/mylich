import { test, describe, beforeEach, afterEach } from 'node:test';
import assert from 'node:assert/strict';
import type { FastifyInstance } from 'fastify';
import { createTestApp, registerAndLoginUser } from './setup.ts';

describe('Sync Endpoints', () => {
  let app: FastifyInstance;

  beforeEach(async () => {
    app = await createTestApp();
  });

  afterEach(async () => {
    await app.close();
  });

  test('records changes and supports incremental sync with cursor', async () => {
    const { token } = await registerAndLoginUser(app, 'sync_user');

    // 1. Initial sync - should have 0 changes
    const initialSyncRes = await app.inject({
      method: 'GET',
      url: '/sync',
      headers: { authorization: `Bearer ${token}` },
    });
    assert.equal(initialSyncRes.statusCode, 200);
    const initialSync = JSON.parse(initialSyncRes.payload);
    assert.equal(initialSync.changes.length, 0);
    const initialCursor = initialSync.cursor;

    // 2. Create an event
    const createRes = await app.inject({
      method: 'POST',
      url: '/events',
      headers: { authorization: `Bearer ${token}` },
      payload: {
        title: 'Meeting 1',
        start_at: '2026-08-18T10:00:00Z',
        end_at: '2026-08-18T11:00:00Z',
        timezone: 'UTC',
      },
    });
    assert.equal(createRes.statusCode, 201);
    const createdEvent = JSON.parse(createRes.payload);

    // 3. Sync since initial cursor - should contain create action
    const sync1Res = await app.inject({
      method: 'GET',
      url: `/sync?since=${encodeURIComponent(initialCursor)}`,
      headers: { authorization: `Bearer ${token}` },
    });
    assert.equal(sync1Res.statusCode, 200);
    const sync1 = JSON.parse(sync1Res.payload);
    assert.equal(sync1.changes.length, 1);
    assert.equal(sync1.changes[0].action, 'create');
    assert.equal(sync1.changes[0].entity_id, createdEvent.id);
    assert.equal(sync1.changes[0].data.title, 'Meeting 1');

    const cursor1 = sync1.cursor;

    // 4. Update the event
    const updateRes = await app.inject({
      method: 'PATCH',
      url: `/events/${createdEvent.id}`,
      headers: { authorization: `Bearer ${token}` },
      payload: {
        title: 'Meeting 1 Updated',
      },
    });
    assert.equal(updateRes.statusCode, 200);

    // 5. Delete the event (soft delete)
    const deleteRes = await app.inject({
      method: 'DELETE',
      url: `/events/${createdEvent.id}`,
      headers: { authorization: `Bearer ${token}` },
    });
    assert.equal(deleteRes.statusCode, 204);

    // 6. Verify deleted event is not returned in standard GET /events
    const listRes = await app.inject({
      method: 'GET',
      url: '/events',
      headers: { authorization: `Bearer ${token}` },
    });
    assert.equal(listRes.statusCode, 200);
    const listEvents = JSON.parse(listRes.payload);
    assert.equal(listEvents.length, 0);

    // 7. Sync since cursor1 - should contain update and delete changes
    const sync2Res = await app.inject({
      method: 'GET',
      url: `/sync?since=${encodeURIComponent(cursor1)}`,
      headers: { authorization: `Bearer ${token}` },
    });
    assert.equal(sync2Res.statusCode, 200);
    const sync2 = JSON.parse(sync2Res.payload);
    assert.equal(sync2.changes.length, 2);
    assert.equal(sync2.changes[0].action, 'update');
    assert.equal(sync2.changes[0].data.title, 'Meeting 1 Updated');
    assert.equal(sync2.changes[1].action, 'delete');
    assert.equal(sync2.changes[1].entity_id, createdEvent.id);
  });

  test('multi-tenant isolation: User A cannot see User B sync changes', async () => {
    const { token: tokenA } = await registerAndLoginUser(app, 'sync_user_a');
    const { token: tokenB } = await registerAndLoginUser(app, 'sync_user_b');

    // User A creates event
    await app.inject({
      method: 'POST',
      url: '/events',
      headers: { authorization: `Bearer ${tokenA}` },
      payload: {
        title: 'Secret Event A',
        start_at: '2026-08-18T10:00:00Z',
        end_at: '2026-08-18T11:00:00Z',
        timezone: 'UTC',
      },
    });

    // User B syncs - should see 0 changes
    const syncBRes = await app.inject({
      method: 'GET',
      url: '/sync',
      headers: { authorization: `Bearer ${tokenB}` },
    });
    assert.equal(syncBRes.statusCode, 200);
    const syncB = JSON.parse(syncBRes.payload);
    assert.equal(syncB.changes.length, 0);
  });
});

import type { Database } from '../connection.ts';

export interface EventIntegration {
  id: string;
  event_id: string;
  integration_id: string;
  external_id: string;
  external_updated_at?: string | null;
  etag?: string | null;
  metadata?: string | null;
  created_at: string;
  updated_at: string;
}

export interface ConflictRecord {
  id: string;
  user_id: string;
  event_id?: string | null;
  integration_id: string;
  local_snapshot: string;
  remote_snapshot: string;
  status: 'unresolved' | 'resolved_local' | 'resolved_remote';
  detected_at: string;
  resolved_at?: string | null;
}

export class EventIntegrationRepository {
  private db: Database;

  constructor(db: Database) {
    this.db = db;
  }

  public findByEventAndIntegration(eventId: string, integrationId: string): EventIntegration | null {
    const stmt = this.db.prepare(
      'SELECT * FROM event_integrations WHERE event_id = ? AND integration_id = ?',
    );
    const row = stmt.get(eventId, integrationId) as EventIntegration | undefined;
    return row || null;
  }

  public findByExternalId(integrationId: string, externalId: string): EventIntegration | null {
    const stmt = this.db.prepare(
      'SELECT * FROM event_integrations WHERE integration_id = ? AND external_id = ?',
    );
    const row = stmt.get(integrationId, externalId) as EventIntegration | undefined;
    return row || null;
  }

  public upsert(mapping: {
    id: string;
    event_id: string;
    integration_id: string;
    external_id: string;
    external_updated_at?: string | null;
    etag?: string | null;
    metadata?: string | null;
  }): EventIntegration {
    const now = new Date().toISOString();
    const stmt = this.db.prepare(`
      INSERT INTO event_integrations (id, event_id, integration_id, external_id, external_updated_at, etag, metadata, created_at, updated_at)
      VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
      ON CONFLICT(event_id, integration_id) DO UPDATE SET
        external_id = excluded.external_id,
        external_updated_at = excluded.external_updated_at,
        etag = excluded.etag,
        metadata = excluded.metadata,
        updated_at = excluded.updated_at
    `);
    stmt.run(
      mapping.id,
      mapping.event_id,
      mapping.integration_id,
      mapping.external_id,
      mapping.external_updated_at ?? null,
      mapping.etag ?? null,
      mapping.metadata ?? null,
      now,
      now,
    );

    return this.findByEventAndIntegration(mapping.event_id, mapping.integration_id)!;
  }

  public deleteByEventAndIntegration(eventId: string, integrationId: string): void {
    const stmt = this.db.prepare(
      'DELETE FROM event_integrations WHERE event_id = ? AND integration_id = ?',
    );
    stmt.run(eventId, integrationId);
  }

  // --- Conflicts ---
  public recordConflict(conflict: {
    id: string;
    user_id: string;
    event_id?: string | null;
    integration_id: string;
    local_snapshot: string;
    remote_snapshot: string;
  }): ConflictRecord {
    const now = new Date().toISOString();
    const stmt = this.db.prepare(`
      INSERT INTO conflicts (id, user_id, event_id, integration_id, local_snapshot, remote_snapshot, status, detected_at)
      VALUES (?, ?, ?, ?, ?, ?, 'unresolved', ?)
    `);
    stmt.run(
      conflict.id,
      conflict.user_id,
      conflict.event_id ?? null,
      conflict.integration_id,
      conflict.local_snapshot,
      conflict.remote_snapshot,
      now,
    );

    return this.getConflict(conflict.id)!;
  }

  public getConflict(id: string): ConflictRecord | null {
    const stmt = this.db.prepare('SELECT * FROM conflicts WHERE id = ?');
    const row = stmt.get(id) as ConflictRecord | undefined;
    return row || null;
  }

  public listUnresolvedConflicts(userId: string): ConflictRecord[] {
    const stmt = this.db.prepare(
      "SELECT * FROM conflicts WHERE user_id = ? AND status = 'unresolved' ORDER BY detected_at DESC",
    );
    return stmt.all(userId) as ConflictRecord[];
  }

  public resolveConflict(id: string, status: 'resolved_local' | 'resolved_remote'): void {
    const now = new Date().toISOString();
    const stmt = this.db.prepare(
      'UPDATE conflicts SET status = ?, resolved_at = ? WHERE id = ?',
    );
    stmt.run(status, now, id);
  }
}

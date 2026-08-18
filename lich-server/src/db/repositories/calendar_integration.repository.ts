import type { Database } from '../connection.ts';

export interface CalendarIntegration {
  id: string;
  calendar_id: string;
  integration_id: string;
  external_calendar_id: string;
  sync_direction: 'push' | 'pull' | 'bidirectional';
  enabled: number;
  created_at: string;
  updated_at: string;
}

export interface IntegrationSyncState {
  id: string;
  integration_id: string;
  resource: string;
  cursor: string;
  last_synced_at: string;
}

export class CalendarIntegrationRepository {
  private db: Database;

  constructor(db: Database) {
    this.db = db;
  }

  public listByIntegrationId(integrationId: string): CalendarIntegration[] {
    const stmt = this.db.prepare('SELECT * FROM calendar_integrations WHERE integration_id = ?');
    return stmt.all(integrationId) as unknown as CalendarIntegration[];
  }

  public findByCalendarAndIntegration(calendarId: string, integrationId: string): CalendarIntegration | null {
    const stmt = this.db.prepare(
      'SELECT * FROM calendar_integrations WHERE calendar_id = ? AND integration_id = ?',
    );
    const row = stmt.get(calendarId, integrationId) as unknown as CalendarIntegration | undefined;
    return row || null;
  }

  public findByExternalCalendarId(integrationId: string, externalCalId: string): CalendarIntegration | null {
    const stmt = this.db.prepare(
      'SELECT * FROM calendar_integrations WHERE integration_id = ? AND external_calendar_id = ?',
    );
    const row = stmt.get(integrationId, externalCalId) as unknown as CalendarIntegration | undefined;
    return row || null;
  }

  public upsert(mapping: {
    id: string;
    calendar_id: string;
    integration_id: string;
    external_calendar_id: string;
    sync_direction?: 'push' | 'pull' | 'bidirectional';
    enabled?: boolean;
  }): CalendarIntegration {
    const now = new Date().toISOString();
    const enabledVal = mapping.enabled === false ? 0 : 1;
    const directionVal = mapping.sync_direction || 'bidirectional';

    const stmt = this.db.prepare(`
      INSERT INTO calendar_integrations (id, calendar_id, integration_id, external_calendar_id, sync_direction, enabled, created_at, updated_at)
      VALUES (?, ?, ?, ?, ?, ?, ?, ?)
      ON CONFLICT(calendar_id, integration_id) DO UPDATE SET
        external_calendar_id = excluded.external_calendar_id,
        sync_direction = excluded.sync_direction,
        enabled = excluded.enabled,
        updated_at = excluded.updated_at
    `);
    stmt.run(
      mapping.id,
      mapping.calendar_id,
      mapping.integration_id,
      mapping.external_calendar_id,
      directionVal,
      enabledVal,
      now,
      now,
    );

    return this.findByCalendarAndIntegration(mapping.calendar_id, mapping.integration_id)!;
  }

  public delete(id: string): void {
    const stmt = this.db.prepare('DELETE FROM calendar_integrations WHERE id = ?');
    stmt.run(id);
  }

  // --- Sync State (Cursor / syncToken) ---
  public getSyncState(integrationId: string, resource: string): IntegrationSyncState | null {
    const stmt = this.db.prepare(
      'SELECT * FROM integration_sync_state WHERE integration_id = ? AND resource = ?',
    );
    const row = stmt.get(integrationId, resource) as unknown as IntegrationSyncState | undefined;
    return row || null;
  }

  public setSyncState(id: string, integrationId: string, resource: string, cursor: string): void {
    const now = new Date().toISOString();
    const stmt = this.db.prepare(`
      INSERT INTO integration_sync_state (id, integration_id, resource, cursor, last_synced_at)
      VALUES (?, ?, ?, ?, ?)
      ON CONFLICT(integration_id, resource) DO UPDATE SET
        cursor = excluded.cursor,
        last_synced_at = excluded.last_synced_at
    `);
    stmt.run(id, integrationId, resource, cursor, now);
  }
}

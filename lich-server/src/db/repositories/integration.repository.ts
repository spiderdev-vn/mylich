import type { Database } from '../connection.ts';

export interface Integration {
  id: string;
  user_id: string;
  provider: string;
  status: 'connected' | 'disconnected' | 'error' | 'needs_reauth';
  created_at: string;
  updated_at: string;
}

export interface IntegrationCredentials {
  integration_id: string;
  access_token: string;
  refresh_token?: string | null;
  token_type: string;
  expires_at?: string | null;
  scope?: string | null;
  created_at: string;
  updated_at: string;
}

export class IntegrationRepository {
  private db: Database;

  constructor(db: Database) {
    this.db = db;
  }

  public findByUserAndProvider(userId: string, provider: string): Integration | null {
    const stmt = this.db.prepare('SELECT * FROM integrations WHERE user_id = ? AND provider = ?');
    const row = stmt.get(userId, provider) as Integration | undefined;
    return row || null;
  }

  public findById(id: string): Integration | null {
    const stmt = this.db.prepare('SELECT * FROM integrations WHERE id = ?');
    const row = stmt.get(id) as Integration | undefined;
    return row || null;
  }

  public upsertIntegration(integration: {
    id: string;
    user_id: string;
    provider: string;
    status: 'connected' | 'disconnected' | 'error' | 'needs_reauth';
  }): Integration {
    const now = new Date().toISOString();
    const stmt = this.db.prepare(`
      INSERT INTO integrations (id, user_id, provider, status, created_at, updated_at)
      VALUES (?, ?, ?, ?, ?, ?)
      ON CONFLICT(user_id, provider) DO UPDATE SET
        status = excluded.status,
        updated_at = excluded.updated_at
    `);
    stmt.run(integration.id, integration.user_id, integration.provider, integration.status, now, now);

    return this.findByUserAndProvider(integration.user_id, integration.provider)!;
  }

  public saveCredentials(creds: {
    integration_id: string;
    access_token: string;
    refresh_token?: string | null;
    token_type?: string;
    expires_at?: string | null;
    scope?: string | null;
  }): IntegrationCredentials {
    const now = new Date().toISOString();
    const stmt = this.db.prepare(`
      INSERT INTO integration_credentials (integration_id, access_token, refresh_token, token_type, expires_at, scope, created_at, updated_at)
      VALUES (?, ?, ?, ?, ?, ?, ?, ?)
      ON CONFLICT(integration_id) DO UPDATE SET
        access_token = excluded.access_token,
        refresh_token = COALESCE(excluded.refresh_token, integration_credentials.refresh_token),
        token_type = excluded.token_type,
        expires_at = excluded.expires_at,
        scope = excluded.scope,
        updated_at = excluded.updated_at
    `);
    stmt.run(
      creds.integration_id,
      creds.access_token,
      creds.refresh_token ?? null,
      creds.token_type ?? 'Bearer',
      creds.expires_at ?? null,
      creds.scope ?? null,
      now,
      now,
    );

    return this.getCredentials(creds.integration_id)!;
  }

  public getCredentials(integrationId: string): IntegrationCredentials | null {
    const stmt = this.db.prepare('SELECT * FROM integration_credentials WHERE integration_id = ?');
    const row = stmt.get(integrationId) as IntegrationCredentials | undefined;
    return row || null;
  }

  public updateStatus(id: string, status: 'connected' | 'disconnected' | 'error' | 'needs_reauth'): void {
    const now = new Date().toISOString();
    const stmt = this.db.prepare('UPDATE integrations SET status = ?, updated_at = ? WHERE id = ?');
    stmt.run(status, now, id);
  }

  public delete(id: string): void {
    const stmt = this.db.prepare('DELETE FROM integrations WHERE id = ?');
    stmt.run(id);
  }
}

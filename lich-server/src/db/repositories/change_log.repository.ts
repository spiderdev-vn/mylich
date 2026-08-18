import crypto from 'node:crypto';
import type { Database } from '../connection.ts';

export interface ChangeLogRecord {
  id: string;
  user_id: string;
  entity_type: 'event' | 'calendar';
  entity_id: string;
  action: 'create' | 'update' | 'delete';
  data: string | null;
  created_at: string;
}

export class ChangeLogRepository {
  private db: Database;

  constructor(db: Database) {
    this.db = db;
  }

  recordChange(params: {
    userId: string;
    entityType: 'event' | 'calendar';
    entityId: string;
    action: 'create' | 'update' | 'delete';
    data?: any;
    createdAt?: string;
  }): ChangeLogRecord {
    const id = crypto.randomUUID();
    const createdAt = params.createdAt || new Date().toISOString();
    const dataStr = params.data ? JSON.stringify(params.data) : null;

    const stmt = this.db.prepare(`
      INSERT INTO change_logs (id, user_id, entity_type, entity_id, action, data, created_at)
      VALUES (?, ?, ?, ?, ?, ?, ?)
    `);

    stmt.run(
      id,
      params.userId,
      params.entityType,
      params.entityId,
      params.action,
      dataStr,
      createdAt,
    );

    return {
      id,
      user_id: params.userId,
      entity_type: params.entityType,
      entity_id: params.entityId,
      action: params.action,
      data: dataStr,
      created_at: createdAt,
    };
  }

  getChangesSince(userId: string, since?: string, limit: number = 100): ChangeLogRecord[] {
    if (since) {
      const stmt = this.db.prepare(`
        SELECT id, user_id, entity_type, entity_id, action, data, created_at
        FROM change_logs
        WHERE user_id = ? AND created_at > ?
        ORDER BY created_at ASC
        LIMIT ?
      `);
      return stmt.all(userId, since, limit) as unknown as ChangeLogRecord[];
    } else {
      const stmt = this.db.prepare(`
        SELECT id, user_id, entity_type, entity_id, action, data, created_at
        FROM change_logs
        WHERE user_id = ?
        ORDER BY created_at ASC
        LIMIT ?
      `);
      return stmt.all(userId, limit) as unknown as ChangeLogRecord[];
    }
  }
}

import type { Database } from '../connection.ts';

export interface Calendar {
  id: string;
  user_id: string;
  name: string;
  description: string | null;
  timezone: string;
  is_default: number;
  created_at: string;
  updated_at: string;
}

export class CalendarRepository {
  private db: Database;

  constructor(db: Database) {
    this.db = db;
  }

  public create(calendar: {
    id: string;
    user_id: string;
    name: string;
    description?: string | null;
    timezone: string;
    is_default?: boolean;
  }): Calendar {
    const now = new Date().toISOString();
    const isDefaultInt = calendar.is_default ? 1 : 0;
    const stmt = this.db.prepare(`
      INSERT INTO calendars (id, user_id, name, description, timezone, is_default, created_at, updated_at)
      VALUES (?, ?, ?, ?, ?, ?, ?, ?)
    `);
    stmt.run(
      calendar.id,
      calendar.user_id,
      calendar.name,
      calendar.description || null,
      calendar.timezone,
      isDefaultInt,
      now,
      now
    );

    return {
      id: calendar.id,
      user_id: calendar.user_id,
      name: calendar.name,
      description: calendar.description || null,
      timezone: calendar.timezone,
      is_default: isDefaultInt,
      created_at: now,
      updated_at: now,
    };
  }

  public findById(id: string): Calendar | null {
    const stmt = this.db.prepare(`SELECT * FROM calendars WHERE id = ?`);
    const row = stmt.get(id) as Calendar | undefined;
    return row || null;
  }

  public findByUserId(userId: string): Calendar[] {
    const stmt = this.db.prepare(`SELECT * FROM calendars WHERE user_id = ? ORDER BY is_default DESC, name ASC`);
    return stmt.all(userId) as Calendar[];
  }

  public findDefaultByUserId(userId: string): Calendar | null {
    const stmt = this.db.prepare(`SELECT * FROM calendars WHERE user_id = ? AND is_default = 1 LIMIT 1`);
    const row = stmt.get(userId) as Calendar | undefined;
    return row || null;
  }

  public update(
    id: string,
    fields: Partial<Pick<Calendar, 'name' | 'description' | 'timezone' | 'is_default'>>
  ): Calendar | null {
    const existing = this.findById(id);
    if (!existing) {
      return null;
    }

    const name = fields.name !== undefined ? fields.name : existing.name;
    const description = fields.description !== undefined ? fields.description : existing.description;
    const timezone = fields.timezone !== undefined ? fields.timezone : existing.timezone;
    const isDefault = fields.is_default !== undefined ? fields.is_default : existing.is_default;
    const now = new Date().toISOString();

    const stmt = this.db.prepare(`
      UPDATE calendars
      SET name = ?, description = ?, timezone = ?, is_default = ?, updated_at = ?
      WHERE id = ?
    `);
    stmt.run(name, description, timezone, isDefault, now, id);

    return this.findById(id);
  }

  public delete(id: string): boolean {
    const stmt = this.db.prepare(`DELETE FROM calendars WHERE id = ?`);
    const result = stmt.run(id);
    return (result.changes ?? 0) > 0;
  }
}

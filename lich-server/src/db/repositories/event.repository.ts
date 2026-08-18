import type { Database } from '../connection.ts';

export interface Event {
  id: string;
  calendar_id: string;
  title: string;
  description: string | null;
  start_at: string;
  end_at: string;
  timezone: string;
  location: string | null;
  created_at: string;
  updated_at: string;
  deleted_at: string | null;
}

export class EventRepository {
  private db: Database;

  constructor(db: Database) {
    this.db = db;
  }

  public create(event: {
    id: string;
    calendar_id: string;
    title: string;
    description?: string | null;
    start_at: string;
    end_at: string;
    timezone: string;
    location?: string | null;
  }): Event {
    const now = new Date().toISOString();
    const stmt = this.db.prepare(`
      INSERT INTO events (id, calendar_id, title, description, start_at, end_at, timezone, location, created_at, updated_at, deleted_at)
      VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NULL)
    `);
    stmt.run(
      event.id,
      event.calendar_id,
      event.title,
      event.description || null,
      event.start_at,
      event.end_at,
      event.timezone,
      event.location || null,
      now,
      now
    );

    return {
      id: event.id,
      calendar_id: event.calendar_id,
      title: event.title,
      description: event.description || null,
      start_at: event.start_at,
      end_at: event.end_at,
      timezone: event.timezone,
      location: event.location || null,
      created_at: now,
      updated_at: now,
      deleted_at: null,
    };
  }

  public findById(id: string): Event | null {
    const stmt = this.db.prepare(`
      SELECT * FROM events WHERE id = ? AND deleted_at IS NULL
    `);
    const row = stmt.get(id) as Event | undefined;
    return row || null;
  }

  public findByIdWithCalendar(id: string): (Event & { user_id: string }) | null {
    const stmt = this.db.prepare(`
      SELECT e.*, c.user_id
      FROM events e
      INNER JOIN calendars c ON e.calendar_id = c.id
      WHERE e.id = ? AND e.deleted_at IS NULL
    `);
    const row = stmt.get(id) as (Event & { user_id: string }) | undefined;
    return row || null;
  }

  public findEvents(params: {
    userId: string;
    calendarId?: string;
    from?: string;
    to?: string;
  }): Event[] {
    const conditions: string[] = [
      'c.user_id = ?',
      'e.deleted_at IS NULL',
    ];
    const args: any[] = [params.userId];

    if (params.calendarId) {
      conditions.push('e.calendar_id = ?');
      args.push(params.calendarId);
    }

    if (params.from) {
      conditions.push('e.end_at > ?');
      args.push(params.from);
    }

    if (params.to) {
      conditions.push('e.start_at < ?');
      args.push(params.to);
    }

    const whereClause = conditions.join(' AND ');
    const sql = `
      SELECT e.*
      FROM events e
      INNER JOIN calendars c ON e.calendar_id = c.id
      WHERE ${whereClause}
      ORDER BY e.start_at ASC
    `;

    const stmt = this.db.prepare(sql);
    return stmt.all(...args) as Event[];
  }

  public update(
    id: string,
    fields: Partial<Pick<Event, 'title' | 'description' | 'start_at' | 'end_at' | 'timezone' | 'location' | 'calendar_id'>>
  ): Event | null {
    const existing = this.findById(id);
    if (!existing) {
      return null;
    }

    const title = fields.title !== undefined ? fields.title : existing.title;
    const description = fields.description !== undefined ? fields.description : existing.description;
    const start_at = fields.start_at !== undefined ? fields.start_at : existing.start_at;
    const end_at = fields.end_at !== undefined ? fields.end_at : existing.end_at;
    const timezone = fields.timezone !== undefined ? fields.timezone : existing.timezone;
    const location = fields.location !== undefined ? fields.location : existing.location;
    const calendar_id = fields.calendar_id !== undefined ? fields.calendar_id : existing.calendar_id;
    const now = new Date().toISOString();

    const stmt = this.db.prepare(`
      UPDATE events
      SET title = ?, description = ?, start_at = ?, end_at = ?, timezone = ?, location = ?, calendar_id = ?, updated_at = ?
      WHERE id = ? AND deleted_at IS NULL
    `);
    stmt.run(title, description, start_at, end_at, timezone, location, calendar_id, now, id);

    return this.findById(id);
  }

  public delete(id: string): boolean {
    const stmt = this.db.prepare('DELETE FROM events WHERE id = ?');
    const result = stmt.run(id);
    return (result.changes ?? 0) > 0;
  }

  public softDelete(id: string): boolean {
    const now = new Date().toISOString();
    const stmt = this.db.prepare(`
      UPDATE events SET deleted_at = ? WHERE id = ? AND deleted_at IS NULL
    `);
    const result = stmt.run(now, id);
    return (result.changes ?? 0) > 0;
  }
}

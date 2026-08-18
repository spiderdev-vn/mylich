import type { Database } from '../connection.ts';

export interface User {
  id: string;
  username: string;
  password_hash: string;
  created_at: string;
  updated_at: string;
}

export class UserRepository {
  private db: Database;

  constructor(db: Database) {
    this.db = db;
  }

  public create(user: { id: string; username: string; password_hash: string }): User {
    const now = new Date().toISOString();
    const stmt = this.db.prepare(`
      INSERT INTO users (id, username, password_hash, created_at, updated_at)
      VALUES (?, ?, ?, ?, ?)
    `);
    stmt.run(user.id, user.username, user.password_hash, now, now);

    return {
      id: user.id,
      username: user.username,
      password_hash: user.password_hash,
      created_at: now,
      updated_at: now,
    };
  }

  public findById(id: string): User | null {
    const stmt = this.db.prepare(`SELECT * FROM users WHERE id = ?`);
    const row = stmt.get(id) as User | undefined;
    return row || null;
  }

  public findByUsername(username: string): User | null {
    const stmt = this.db.prepare(`SELECT * FROM users WHERE username = ? COLLATE NOCASE`);
    const row = stmt.get(username) as User | undefined;
    return row || null;
  }
}

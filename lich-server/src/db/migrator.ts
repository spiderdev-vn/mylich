import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import type { Database } from './connection.ts';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

export class Migrator {
  private db: Database;
  private migrationsDir: string;

  constructor(db: Database, migrationsDir?: string) {
    this.db = db;
    this.migrationsDir = migrationsDir || path.join(__dirname, 'migrations');
  }

  public run(): void {
    // 1. Ensure migrations tracking table exists
    this.db.exec(`
      CREATE TABLE IF NOT EXISTS _migrations (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        name TEXT UNIQUE NOT NULL,
        applied_at TEXT NOT NULL
      );
    `);

    // 2. Read migration files
    if (!fs.existsSync(this.migrationsDir)) {
      return;
    }

    const files = fs
      .readdirSync(this.migrationsDir)
      .filter((f) => f.endsWith('.sql'))
      .sort();

    // 3. Get applied migrations
    const stmt = this.db.prepare('SELECT name FROM _migrations');
    const appliedRows = stmt.all() as { name: string }[];
    const appliedSet = new Set(appliedRows.map((r) => r.name));

    // 4. Apply pending migrations
    for (const file of files) {
      if (!appliedSet.has(file)) {
        const filePath = path.join(this.migrationsDir, file);
        const sql = fs.readFileSync(filePath, 'utf-8');

        this.db.exec('BEGIN TRANSACTION;');
        try {
          this.db.exec(sql);
          const insertStmt = this.db.prepare(
            'INSERT INTO _migrations (name, applied_at) VALUES (?, ?);',
          );
          insertStmt.run(file, new Date().toISOString());
          this.db.exec('COMMIT;');
        } catch (err) {
          this.db.exec('ROLLBACK;');
          throw new Error(`Failed to apply migration '${file}': ${(err as Error).message}`);
        }
      }
    }
  }
}

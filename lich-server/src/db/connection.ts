import { DatabaseSync } from 'node:sqlite';
import fs from 'node:fs';
import path from 'node:path';

export class Database {
  private db: DatabaseSync;

  constructor(dbPath: string) {
    if (dbPath !== ':memory:') {
      const dir = path.dirname(dbPath);
      if (!fs.existsSync(dir)) {
        fs.mkdirSync(dir, { recursive: true });
      }
      try {
        fs.accessSync(dir, fs.constants.W_OK);
      } catch {
        throw new Error(
          `Permission denied: Thư mục "${dir}" không có quyền ghi. ` +
          `Nếu dùng Docker, hãy chạy trên host: sudo chown -R 1000:1000 ./data`
        );
      }
    }

    this.db = new DatabaseSync(dbPath);
    this.db.exec('PRAGMA foreign_keys = ON;');
    if (dbPath !== ':memory:') {
      this.db.exec('PRAGMA journal_mode = WAL;');
    }
  }

  get instance(): DatabaseSync {
    return this.db;
  }

  exec(sql: string): void {
    this.db.exec(sql);
  }

  prepare(sql: string) {
    return this.db.prepare(sql);
  }

  close(): void {
    this.db.close();
  }
}

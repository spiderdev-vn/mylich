import path from 'node:path';

export interface AppConfig {
  host: string;
  port: number;
  databasePath: string;
  jwtSecret: string;
  logLevel: string;
}

export function loadConfig(): AppConfig {
  const host = process.env.HOST || '127.0.0.1';
  const port = parseInt(process.env.PORT || '3000', 10);
  const databasePath = process.env.DATABASE_PATH || './data/lich.db';
  const jwtSecret = process.env.JWT_SECRET || 'lich-default-dev-secret-key-32chars-min';
  const logLevel = process.env.LOG_LEVEL || 'info';

  return {
    host,
    port: isNaN(port) ? 3000 : port,
    databasePath: path.resolve(process.cwd(), databasePath),
    jwtSecret,
    logLevel,
  };
}

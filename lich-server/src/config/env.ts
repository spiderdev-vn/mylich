import path from 'node:path';

export interface AppConfig {
  host: string;
  port: number;
  databasePath: string;
  jwtSecret: string;
  logLevel: string;
  googleClientId?: string;
  googleClientSecret?: string;
  googleRedirectUri?: string;
  useFakeGoogleProvider?: boolean;
}

export function loadConfig(): AppConfig {
  const host = process.env.HOST || '127.0.0.1';
  const port = parseInt(process.env.PORT || '3000', 10);
  const databasePath = process.env.DATABASE_PATH || './data/lich.db';
  const jwtSecret = process.env.JWT_SECRET || 'lich-default-dev-secret-key-32chars-min';
  const logLevel = process.env.LOG_LEVEL || 'info';

  const googleClientId = process.env.GOOGLE_CLIENT_ID || '';
  const googleClientSecret = process.env.GOOGLE_CLIENT_SECRET || '';
  const googleRedirectUri =
    process.env.GOOGLE_REDIRECT_URI || `http://${host}:${port}/auth/google/callback`;
  const fakeEnv = (process.env.USE_FAKE_GOOGLE || '').toLowerCase();
  const useFakeGoogleProvider =
    fakeEnv === 'true' || fakeEnv === 't' || fakeEnv === '1' || fakeEnv === 'yes' || !googleClientId || !googleClientSecret;

  return {
    host,
    port: isNaN(port) ? 3000 : port,
    databasePath: path.resolve(process.cwd(), databasePath),
    jwtSecret,
    logLevel,
    googleClientId,
    googleClientSecret,
    googleRedirectUri,
    useFakeGoogleProvider,
  };
}

import fs from 'node:fs';
import path from 'node:path';

export interface AppConfig {
  host: string;
  port: number;
  databasePath: string;
  jwtSecret: string;
  logLevel: string;
  trustProxy: boolean | string | number;
  googleClientId?: string;
  googleClientSecret?: string;
  googleRedirectUri?: string;
  useFakeGoogleProvider?: boolean;
}

/**
 * Tự động tìm và nạp file .env nếu có sẵn mà không cần cờ --env-file
 */
function tryLoadEnvFiles(): void {
  const candidatePaths = [
    path.resolve(process.cwd(), '.env'),
    path.resolve(process.cwd(), 'lich-server/.env'),
    path.resolve(process.cwd(), '../.env'),
  ];

  for (const envPath of candidatePaths) {
    if (fs.existsSync(envPath)) {
      try {
        if (typeof (process as any).loadEnvFile === 'function') {
          (process as any).loadEnvFile(envPath);
        }
      } catch {
        // Bỏ qua lỗi nếu file không thể đọc
      }
    }
  }
}

export function loadConfig(): AppConfig {
  tryLoadEnvFiles();

  const host = process.env.HOST || '127.0.0.1';
  const port = parseInt(process.env.PORT || '3000', 10);
  const databasePath = process.env.DATABASE_PATH || './data/lich.db';
  const jwtSecret = process.env.JWT_SECRET || 'lich-default-dev-secret-key-32chars-min';
  const logLevel = process.env.LOG_LEVEL || 'info';

  const trustProxyEnv = process.env.TRUST_PROXY ?? process.env.BEHIND_PROXY;
  let trustProxy: boolean | string | number = false;
  if (trustProxyEnv !== undefined) {
    const trimmed = trustProxyEnv.trim().toLowerCase();
    if (trimmed === 'true' || trimmed === '1' || trimmed === 'yes') {
      trustProxy = true;
    } else if (trimmed === 'false' || trimmed === '0' || trimmed === 'no') {
      trustProxy = false;
    } else if (!isNaN(Number(trimmed))) {
      trustProxy = Number(trimmed);
    } else {
      trustProxy = trustProxyEnv.trim();
    }
  }

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
    trustProxy,
    googleClientId,
    googleClientSecret,
    googleRedirectUri,
    useFakeGoogleProvider,
  };
}

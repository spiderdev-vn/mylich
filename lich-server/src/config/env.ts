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

function validateConfig(config: AppConfig): void {
  const errors: string[] = [];

  // 1. Kiểm tra Port
  if (isNaN(config.port) || config.port < 1 || config.port > 65535) {
    errors.push(`• PORT không hợp lệ (${config.port}). Phải là một số nguyên từ 1 đến 65535.`);
  }

  // 2. Kiểm tra JWT_SECRET
  const devPlaceholders = [
    'lich-default-dev-secret-key-32chars-min',
    'change-this-to-a-secure-random-secret-in-production',
    'supersecret-dev-jwt-key-replace-in-production-32chars',
  ];

  if (!config.jwtSecret || config.jwtSecret.trim() === '') {
    errors.push('• JWT_SECRET không được để trống.');
  } else if (config.jwtSecret.length < 32) {
    errors.push(`• JWT_SECRET quá ngắn (${config.jwtSecret.length} ký tự). Độ dài tối thiểu phải là 32 ký tự để đảm bảo bảo mật.`);
  } else if (
    process.env.NODE_ENV === 'production' &&
    devPlaceholders.includes(config.jwtSecret)
  ) {
    errors.push('• Ở môi trường production (NODE_ENV=production), JWT_SECRET bắt buộc phải là một chuỗi ngẫu nhiên bảo mật, không được dùng giá trị mặc định.');
  }

  // 3. Kiểm tra cặp biến Google OAuth (nếu có cấu hình)
  if (
    (config.googleClientId && !config.googleClientSecret) ||
    (!config.googleClientId && config.googleClientSecret)
  ) {
    errors.push('• Cấu hình Google OAuth chưa đầy đủ: Cần cung cấp đồng thời cả GOOGLE_CLIENT_ID và GOOGLE_CLIENT_SECRET (hoặc để trống cả 2 để dùng FakeGoogleProvider).');
  }

  if (errors.length > 0) {
    const errorMsg = [
      '',
      '===========================================================',
      ' ❌ LỖI CẤU HÌNH MÁY CHỦ (Lich Server Configuration Guard) ',
      '===========================================================',
      ...errors,
      '',
      '👉 Vui lòng kiểm tra lại file .env hoặc các biến môi trường.',
      '===========================================================',
      '',
    ].join('\n');

    throw new Error(errorMsg);
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

  const config: AppConfig = {
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

  // Guard check
  validateConfig(config);

  return config;
}

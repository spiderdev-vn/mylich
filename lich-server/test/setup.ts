import type { FastifyInstance } from 'fastify';
import { buildApp } from '../src/app.ts';
import { Database } from '../src/db/connection.ts';
import type { AppConfig } from '../src/config/env.ts';

export async function createTestApp(): Promise<FastifyInstance> {
  const db = new Database(':memory:');
  const testConfig: AppConfig = {
    host: '127.0.0.1',
    port: 0,
    databasePath: ':memory:',
    jwtSecret: 'test-jwt-secret-key-32-characters-minimum-for-hs256',
    logLevel: 'silent',
  };

  return buildApp({ config: testConfig, database: db });
}

export async function registerAndLoginUser(
  app: FastifyInstance,
  username: string = 'testuser',
  password: string = 'password123',
): Promise<{ user: any; token: string }> {
  const regRes = await app.inject({
    method: 'POST',
    url: '/auth/register',
    payload: { username, password },
  });

  const body = JSON.parse(regRes.payload);
  return {
    user: body.user,
    token: body.token,
  };
}

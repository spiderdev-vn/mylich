import type { FastifyPluginAsync } from 'fastify';
import type { AuthService } from './auth.service.ts';

export function createAuthRoutes(authService: AuthService): FastifyPluginAsync {
  return async (fastify) => {
    fastify.post('/register', async (request, reply) => {
      const body = request.body as { username?: string; password?: string; timezone?: string } | undefined;
      const result = await authService.register(
        body?.username || '',
        body?.password || '',
        body?.timezone || 'UTC'
      );
      reply.status(201);
      return result;
    });

    fastify.post('/login', async (request) => {
      const body = request.body as { username?: string; password?: string } | undefined;
      const result = await authService.login(
        body?.username || '',
        body?.password || ''
      );
      return result;
    });

    fastify.get('/me', { preHandler: [fastify.authenticate] }, async (request) => {
      return {
        user: request.user,
      };
    });
  };
}

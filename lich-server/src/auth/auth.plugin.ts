import type { FastifyPluginAsync, FastifyRequest, FastifyReply } from 'fastify';
import fp from 'fastify-plugin';
import type { AuthService } from './auth.service.ts';
import { UnauthorizedError } from '../common/errors.ts';

declare module 'fastify' {
  interface FastifyRequest {
    user: {
      id: string;
      username: string;
    };
  }
  interface FastifyInstance {
    authenticate: (request: FastifyRequest, reply: FastifyReply) => Promise<void>;
  }
}

export function createAuthPlugin(authService: AuthService): FastifyPluginAsync {
  const plugin: FastifyPluginAsync = async (fastify) => {
    fastify.decorate(
      'authenticate',
      async (request: FastifyRequest, _reply: FastifyReply) => {
        const authHeader = request.headers.authorization;
        if (!authHeader || !authHeader.startsWith('Bearer ')) {
          throw new UnauthorizedError('Missing or invalid Authorization header');
        }

        const token = authHeader.substring('Bearer '.length).trim();
        if (!token) {
          throw new UnauthorizedError('Token is missing');
        }

        const payload = await authService.verifyToken(token);
        request.user = {
          id: payload.userId,
          username: payload.username,
        };
      }
    );
  };

  return fp(plugin, {
    name: 'auth-plugin',
  });
}

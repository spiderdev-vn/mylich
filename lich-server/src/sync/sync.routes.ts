import type { FastifyPluginAsync } from 'fastify';
import type { SyncService } from './sync.service.ts';

interface SyncRouteOptions {
  syncService: SyncService;
}

export const syncRoutes: FastifyPluginAsync<SyncRouteOptions> = async (fastify, opts) => {
  fastify.addHook('onRequest', fastify.authenticate);

  fastify.get<{
    Querystring: {
      since?: string;
      limit?: string;
    };
  }>('/sync', async (request, reply) => {
    const user = request.user!;
    const since = request.query.since;
    const limit = request.query.limit ? parseInt(request.query.limit, 10) : 100;

    const data = opts.syncService.getSyncData(user.id, since, isNaN(limit) ? 100 : limit);
    return reply.status(200).send(data);
  });
};

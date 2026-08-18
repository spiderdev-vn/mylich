import type { FastifyPluginAsync } from 'fastify';
import type { EventService } from './event.service.ts';
import type { CreateEventDto, UpdateEventDto, EventQueryDto } from './event.types.ts';

export function createEventRoutes(eventService: EventService): FastifyPluginAsync {
  return async (fastify) => {
    fastify.addHook('preHandler', fastify.authenticate);

    fastify.post('/', async (request, reply) => {
      const body = request.body as CreateEventDto;
      const event = eventService.create(request.user.id, body || {});
      reply.status(201);
      return event;
    });

    fastify.get<{ Querystring: EventQueryDto }>('/', async (request) => {
      return eventService.list(request.user.id, request.query || {});
    });

    fastify.get<{ Params: { id: string } }>('/:id', async (request) => {
      return eventService.getById(request.params.id, request.user.id);
    });

    fastify.patch<{ Params: { id: string } }>('/:id', async (request) => {
      const body = request.body as UpdateEventDto;
      return eventService.update(request.params.id, request.user.id, body || {});
    });

    fastify.delete<{ Params: { id: string } }>('/:id', async (request, reply) => {
      eventService.delete(request.params.id, request.user.id);
      reply.status(204);
      return;
    });
  };
}

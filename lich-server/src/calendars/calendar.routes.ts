import type { FastifyPluginAsync } from 'fastify';
import type { CalendarService } from './calendar.service.ts';
import type { CreateCalendarDto, UpdateCalendarDto } from './calendar.types.ts';

export function createCalendarRoutes(calendarService: CalendarService): FastifyPluginAsync {
  return async (fastify) => {
    fastify.addHook('preHandler', fastify.authenticate);

    fastify.post('/', async (request, reply) => {
      const body = request.body as CreateCalendarDto;
      const calendar = calendarService.create(request.user.id, body || {});
      reply.status(201);
      return calendar;
    });

    fastify.get('/', async (request) => {
      return calendarService.listByUser(request.user.id);
    });

    fastify.get<{ Params: { id: string } }>('/:id', async (request) => {
      return calendarService.getById(request.params.id, request.user.id);
    });

    fastify.patch<{ Params: { id: string } }>('/:id', async (request) => {
      const body = request.body as UpdateCalendarDto;
      return calendarService.update(request.params.id, request.user.id, body || {});
    });

    fastify.delete<{ Params: { id: string } }>('/:id', async (request, reply) => {
      calendarService.delete(request.params.id, request.user.id);
      reply.status(204);
      return;
    });
  };
}

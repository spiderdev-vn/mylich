import type { FastifyInstance, FastifyPluginAsync } from 'fastify';
import type { IntegrationService } from './integration.service.ts';

export interface IntegrationRoutesOptions {
  integrationService: IntegrationService;
}

export const integrationRoutes: FastifyPluginAsync<IntegrationRoutesOptions> = async (
  fastify: FastifyInstance,
  opts: IntegrationRoutesOptions,
) => {
  const { integrationService } = opts;

  // 1. GET /integrations/google/auth-url (Protected)
  fastify.get(
    '/integrations/google/auth-url',
    { preHandler: [fastify.authenticate] },
    async (request) => {
      const user = request.user as { id: string };
      const { authUrl } = integrationService.getAuthUrl(user.id);
      return { auth_url: authUrl };
    },
  );

  // 2. GET /auth/google/callback (Public OAuth redirect handler)
  fastify.get('/auth/google/callback', async (request, reply) => {
    const query = request.query as { code?: string; state?: string; error?: string };

    if (query.error) {
      return reply.type('text/html').send(`
        <html>
          <body style="font-family: sans-serif; text-align: center; padding: 40px;">
            <h2 style="color: #e53e3e;">Xác thực Google Thất Bại</h2>
            <p>${query.error}</p>
            <p>Bạn có thể đóng cửa sổ này và thử lại trong terminal.</p>
          </body>
        </html>
      `);
    }

    if (!query.code || !query.state) {
      return reply.status(400).send({ error: 'Missing code or state in OAuth callback' });
    }

    try {
      const stateObj = JSON.parse(Buffer.from(query.state, 'base64url').toString('utf8'));
      const userId = stateObj.userId;
      const res = await integrationService.handleCallback(userId, query.code);

      return reply.type('text/html').send(`
        <html>
          <body style="font-family: sans-serif; text-align: center; padding: 40px; background: #0f172a; color: #f8fafc;">
            <h1 style="color: #4ade80;">✓ Kết Nối Google Calendar Thành Công!</h1>
            <p>Tài khoản: <strong>${res.email || 'Google User'}</strong> đã được liên kết với Lich.</p>
            <p style="color: #94a3b8;">Bạn có thể đóng trình duyệt này và quay lại giao diện terminal (lich).</p>
          </body>
        </html>
      `);
    } catch (err: any) {
      return reply.status(500).type('text/html').send(`
        <html>
          <body style="font-family: sans-serif; text-align: center; padding: 40px;">
            <h2 style="color: #e53e3e;">Lỗi Lưu Xác Thực Google</h2>
            <p>${err.message}</p>
          </body>
        </html>
      `);
    }
  });

  // 3. GET /integrations/google/status (Protected)
  fastify.get(
    '/integrations/google/status',
    { preHandler: [fastify.authenticate] },
    async (request) => {
      const user = request.user as { id: string };
      return integrationService.getStatus(user.id);
    },
  );

  // 4. GET /integrations/google/calendars (Protected)
  fastify.get(
    '/integrations/google/calendars',
    { preHandler: [fastify.authenticate] },
    async (request) => {
      const user = request.user as { id: string };
      const calendars = await integrationService.listExternalCalendars(user.id);
      return { calendars };
    },
  );

  // 5. POST /integrations/google/map (Protected)
  fastify.post(
    '/integrations/google/map',
    { preHandler: [fastify.authenticate] },
    async (request, reply) => {
      const user = request.user as { id: string };
      const body = request.body as {
        calendar_id: string;
        external_calendar_id: string;
        sync_direction?: 'push' | 'pull' | 'bidirectional';
      };

      if (!body.calendar_id || !body.external_calendar_id) {
        return reply.status(400).send({ error: 'calendar_id and external_calendar_id are required' });
      }

      integrationService.mapCalendar(
        user.id,
        body.calendar_id,
        body.external_calendar_id,
        body.sync_direction,
      );

      return { success: true };
    },
  );

  // 6. POST /integrations/google/sync (Protected)
  fastify.post(
    '/integrations/google/sync',
    { preHandler: [fastify.authenticate] },
    async (request) => {
      const user = request.user as { id: string };
      const body = (request.body || {}) as {
        calendar_id?: string;
        direction?: 'push' | 'pull' | 'both';
      };

      const result = await integrationService.sync(user.id, body.calendar_id, body.direction);
      return {
        success: true,
        pushed: result.pushed,
        pulled: result.pulled,
      };
    },
  );

  // 7. DELETE /integrations/google (Protected)
  fastify.delete(
    '/integrations/google',
    { preHandler: [fastify.authenticate] },
    async (request) => {
      const user = request.user as { id: string };
      integrationService.disconnect(user.id);
      return { success: true, message: 'Google integration disconnected' };
    },
  );
};

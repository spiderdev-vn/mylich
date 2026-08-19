import type { FastifyInstance, FastifyPluginAsync } from 'fastify';
import type { IntegrationService } from './integration.service.ts';

export interface IntegrationRoutesOptions {
  integrationService: IntegrationService;
}

function escapeHtml(str: string): string {
  return str
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#039;');
}

function renderAuthCallbackHTML(opts: {
  success: boolean;
  title: string;
  email?: string;
  message?: string;
}): string {
  const { success, title, email, message } = opts;
  const safeTitle = escapeHtml(title);
  const safeEmail = email ? escapeHtml(email) : '';
  const safeMessage = message ? escapeHtml(message) : '';

  return `<!DOCTYPE html>
<html lang="vi">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Mỹ Lích — ${safeTitle}</title>
  <link rel="icon" href="data:image/svg+xml,<svg xmlns=%22http://www.w3.org/2000/svg%22 viewBox=%220 0 100 100%22><text y=%22.9em%22 font-size=%2290%22>🗓️</text></svg>">
  <link rel="stylesheet" href="/style.css">
  <style>
    .auth-wrapper {
      min-height: calc(100vh - 220px);
      display: flex;
      flex-direction: column;
      justify-content: center;
      align-items: center;
      text-align: center;
      padding: 20px 0 40px;
    }
    .auth-card {
      background: var(--surface);
      border: 2px solid ${success ? 'var(--pop-green)' : 'var(--pop-pink)'};
      border-radius: 24px;
      padding: 40px 32px;
      max-width: 480px;
      width: 100%;
      box-shadow: 0 8px 0 ${success ? '#15803d' : '#be123c'}, 0 20px 40px rgba(0,0,0,0.5);
      backdrop-filter: blur(12px);
    }
    .status-icon {
      width: 64px;
      height: 64px;
      margin: 0 auto 20px;
      border-radius: 20px;
      display: flex;
      align-items: center;
      justify-content: center;
      font-size: 28px;
      font-weight: 800;
      background: ${success ? 'rgba(34, 197, 94, 0.15)' : 'rgba(255, 42, 133, 0.15)'};
      border: 2px solid ${success ? 'var(--pop-green)' : 'var(--pop-pink)'};
      color: ${success ? 'var(--pop-green)' : 'var(--pop-pink)'};
    }
    .auth-title {
      font-size: 1.45rem;
      font-weight: 800;
      color: var(--text-heading);
      margin-bottom: 12px;
      letter-spacing: -0.02em;
    }
    .email-pill {
      display: inline-flex;
      align-items: center;
      gap: 8px;
      background: #110d24;
      border: 1px solid var(--border);
      border-radius: 12px;
      padding: 8px 16px;
      font-family: 'JetBrains Mono', monospace;
      font-size: 0.95rem;
      font-weight: 700;
      color: var(--pop-cyan);
      margin: 12px 0 16px;
    }
    .auth-desc {
      color: var(--muted);
      font-size: 0.95rem;
      line-height: 1.5;
      margin-bottom: 20px;
    }
    .cli-hint {
      background: rgba(168, 85, 247, 0.1);
      border: 1px dashed var(--primary);
      border-radius: 12px;
      padding: 10px 14px;
      font-size: 0.88rem;
      color: #d8b4fe;
      margin-bottom: 24px;
      display: flex;
      align-items: center;
      justify-content: center;
      gap: 8px;
    }
    .btn-group-auth {
      display: flex;
      gap: 12px;
      justify-content: center;
    }
  </style>
</head>
<body>
  <div class="container">
    <header>
      <a href="/" class="logo">
        <span class="logo-badge">MỸ LÍCH</span>
        <span class="logo-text">Calendar CLI</span>
      </a>
      <nav>
        <a href="/">Trang chủ</a>
        <a href="/docs">Tài liệu</a>
        <a href="https://github.com/spiderdev-vn/mylich" target="_blank" rel="noopener">★ GitHub</a>
      </nav>
    </header>

    <main class="auth-wrapper">
      <div class="auth-card">
        <div class="status-icon">${success ? '✓' : '✕'}</div>
        <h1 class="auth-title">${safeTitle}</h1>
        ${safeEmail ? `<div class="email-pill"><span>📧</span><span>${safeEmail}</span></div>` : ''}
        ${safeMessage ? `<p class="auth-desc">${safeMessage}</p>` : ''}
        <div class="cli-hint">
          <span>💻</span>
          <span>Bạn có thể đóng tab này và tiếp tục thao tác trên Terminal.</span>
        </div>
        <div class="btn-group-auth">
          <a href="/" class="clay-btn clay-btn-secondary" style="font-size: 0.88rem; padding: 8px 18px;">
            <span>Trang chủ</span>
          </a>
          <a href="/docs" class="clay-btn" style="font-size: 0.88rem; padding: 8px 18px;">
            <span>Tài liệu CLI</span>
          </a>
        </div>
      </div>
    </main>
  </div>
</body>
</html>`;
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
      return reply.type('text/html; charset=utf-8').send(
        renderAuthCallbackHTML({
          success: false,
          title: 'Xác Thực Google Thất Bại',
          message: query.error,
        }),
      );
    }

    if (!query.code || !query.state) {
      return reply.status(400).type('text/html; charset=utf-8').send(
        renderAuthCallbackHTML({
          success: false,
          title: 'Thiếu Tham Số Xác Thực',
          message: 'Không tìm thấy mã xác thực (code) hoặc trạng thái (state) trong yêu cầu OAuth callback.',
        }),
      );
    }

    try {
      const stateObj = JSON.parse(Buffer.from(query.state, 'base64url').toString('utf8'));
      const userId = stateObj.userId;
      const res = await integrationService.handleCallback(userId, query.code);

      return reply.type('text/html; charset=utf-8').send(
        renderAuthCallbackHTML({
          success: true,
          title: 'Liên Kết Google Calendar Thành Công!',
          email: res.email || 'Google User',
          message: 'Tài khoản Google Calendar đã được ánh xạ và sẵn sàng đồng bộ 2 chiều.',
        }),
      );
    } catch (err: any) {
      return reply.status(500).type('text/html; charset=utf-8').send(
        renderAuthCallbackHTML({
          success: false,
          title: 'Lỗi Lưu Xác Thực Google',
          message: err.message,
        }),
      );
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

  // 6. POST /integrations/google/create-calendar (Protected - create a new calendar on Google and map it)
  fastify.post(
    '/integrations/google/create-calendar',
    { preHandler: [fastify.authenticate] },
    async (request, reply) => {
      const user = request.user as { id: string };
      const body = request.body as {
        calendar_id: string;
        name?: string;
        sync_direction?: 'push' | 'pull' | 'bidirectional';
      };

      if (!body.calendar_id) {
        return reply.status(400).send({ error: 'calendar_id is required' });
      }

      const extCal = await integrationService.createAndMapCalendar(
        user.id,
        body.calendar_id,
        body.name,
        body.sync_direction,
      );

      reply.status(201);
      return { success: true, external_calendar: extCal };
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
        event_id?: string;
        from?: string;
        to?: string;
      };

      const result = await integrationService.sync(
        user.id,
        body.calendar_id,
        body.direction,
        body.event_id,
        body.from,
        body.to,
      );
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

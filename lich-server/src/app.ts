import Fastify, { type FastifyInstance } from 'fastify';
import cors from '@fastify/cors';
import { type AppConfig, loadConfig } from './config/env.ts';
import { Database } from './db/connection.ts';
import { Migrator } from './db/migrator.ts';
import { UserRepository } from './db/repositories/user.repository.ts';
import { CalendarRepository } from './db/repositories/calendar.repository.ts';
import { EventRepository } from './db/repositories/event.repository.ts';
import { ChangeLogRepository } from './db/repositories/change_log.repository.ts';
import { IntegrationRepository } from './db/repositories/integration.repository.ts';
import { CalendarIntegrationRepository } from './db/repositories/calendar_integration.repository.ts';
import { EventIntegrationRepository } from './db/repositories/event_integration.repository.ts';
import { AuthService } from './auth/auth.service.ts';
import { CalendarService } from './calendars/calendar.service.ts';
import { EventService } from './events/event.service.ts';
import { SyncService } from './sync/sync.service.ts';
import { IntegrationService } from './integrations/integration.service.ts';
import { GoogleProvider } from './integrations/google/google.provider.ts';
import { FakeGoogleProvider } from './integrations/fake/fake.provider.ts';
import { createAuthPlugin } from './auth/auth.plugin.ts';
import { createAuthRoutes } from './auth/auth.routes.ts';
import { createCalendarRoutes } from './calendars/calendar.routes.ts';
import { createEventRoutes } from './events/event.routes.ts';
import { syncRoutes } from './sync/sync.routes.ts';
import { integrationRoutes } from './integrations/integration.routes.ts';
import { AppError } from './common/errors.ts';

export interface BuildAppOptions {
  config?: AppConfig;
  database?: Database;
}

export async function buildApp(options: BuildAppOptions = {}): Promise<FastifyInstance> {
  const config = options.config || loadConfig();
  const db = options.database || new Database(config.databasePath);

  // Run migrations
  const migrator = new Migrator(db);
  migrator.run();

  // Instantiate repositories
  const userRepo = new UserRepository(db);
  const calendarRepo = new CalendarRepository(db);
  const eventRepo = new EventRepository(db);
  const changeLogRepo = new ChangeLogRepository(db);
  const integrationRepo = new IntegrationRepository(db);
  const calendarIntegrationRepo = new CalendarIntegrationRepository(db);
  const eventIntegrationRepo = new EventIntegrationRepository(db);

  // Instantiate providers & services
  const authService = new AuthService(userRepo, calendarRepo, config.jwtSecret);
  const calendarService = new CalendarService(calendarRepo);
  const eventService = new EventService(eventRepo, calendarRepo, changeLogRepo);
  const syncService = new SyncService(changeLogRepo);

  const googleProvider = config.useFakeGoogleProvider
    ? new FakeGoogleProvider()
    : new GoogleProvider({
        clientId: config.googleClientId || '',
        clientSecret: config.googleClientSecret || '',
        redirectUri: config.googleRedirectUri || '',
      });

  const integrationService = new IntegrationService(
    integrationRepo,
    calendarIntegrationRepo,
    eventIntegrationRepo,
    eventRepo,
    calendarRepo,
    googleProvider,
  );

  const app = Fastify({
    logger: {
      level: config.logLevel,
    },
  });

  // CORS
  await app.register(cors, {
    origin: true,
  });

  // Error handler
  app.setErrorHandler((error: any, _request, reply) => {
    if (error instanceof AppError) {
      reply.status(error.statusCode).send({
        error: error.code,
        message: error.message,
        statusCode: error.statusCode,
      });
      return;
    }

    if (error?.validation) {
      reply.status(400).send({
        error: 'VALIDATION_ERROR',
        message: error.message,
        statusCode: 400,
      });
      return;
    }

    app.log.error(error);
    const statusCode = typeof error?.statusCode === 'number' ? error.statusCode : 500;
    const message = typeof error?.message === 'string' ? error.message : 'An unexpected error occurred';
    reply.status(statusCode).send({
      error: 'INTERNAL_SERVER_ERROR',
      message,
      statusCode,
    });
  });

  // Health check
  app.get('/health', async () => {
    return { status: 'ok' };
  });

  // Auth decorator plugin
  await app.register(createAuthPlugin(authService));

  // Routes
  await app.register(createAuthRoutes(authService), { prefix: '/auth' });
  await app.register(createCalendarRoutes(calendarService), { prefix: '/calendars' });
  await app.register(createEventRoutes(eventService), { prefix: '/events' });
  await app.register(syncRoutes, { syncService });
  await app.register(integrationRoutes, { integrationService });

  // Remote nuke endpoint for resetting user data
  app.post('/nuke', { preHandler: [app.authenticate] }, async (request) => {
    const user = request.user as { id: string };
    const userId = user.id;

    db.exec('BEGIN TRANSACTION;');
    try {
      db.prepare('DELETE FROM conflicts WHERE user_id = ?').run(userId);
      db.prepare('DELETE FROM integrations WHERE user_id = ?').run(userId);
      db.prepare('DELETE FROM change_logs WHERE user_id = ?').run(userId);
      db.prepare('DELETE FROM calendars WHERE user_id = ?').run(userId);

      // Recreate default Personal calendar
      calendarRepo.create({
        id: crypto.randomUUID(),
        user_id: userId,
        name: 'Personal',
        timezone: 'UTC',
        is_default: true,
      });

      db.exec('COMMIT;');
      return { success: true, message: 'Remote user data nuked successfully' };
    } catch (err) {
      db.exec('ROLLBACK;');
      throw err;
    }
  });

  // Store db instance on fastify for test cleanup / closing
  app.decorate('db', db);

  return app;
}

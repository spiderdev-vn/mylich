import { buildApp } from './app.ts';
import { loadConfig } from './config/env.ts';

async function start() {
  const config = loadConfig();
  const app = await buildApp({ config });

  try {
    await app.listen({ host: config.host, port: config.port });
    app.log.info(`Lich Server listening on http://${config.host}:${config.port}`);
  } catch (err) {
    app.log.error(err);
    process.exit(1);
  }

  // Graceful shutdown
  const signals = ['SIGINT', 'SIGTERM'];
  for (const signal of signals) {
    process.on(signal, async () => {
      app.log.info(`Received ${signal}, closing server...`);
      try {
        await app.close();
        process.exit(0);
      } catch (err) {
        app.log.error(`Error during graceful shutdown: ${err}`);
        process.exit(1);
      }
    });
  }
}

start();

import type { FastifyPluginAsync } from 'fastify';
import fs from 'node:fs';
import path from 'node:path';

function findPublicFile(filename: string): string | null {
  const candidates = [
    path.resolve(process.cwd(), 'public', filename),
    path.resolve(process.cwd(), 'lich-server', 'public', filename),
    path.resolve(process.cwd(), '..', 'public', filename),
  ];

  for (const p of candidates) {
    if (fs.existsSync(p)) {
      return p;
    }
  }
  return null;
}

function serveFile(reply: any, filename: string, contentType: string, cacheControl = 'public, max-age=3600') {
  const filePath = findPublicFile(filename);
  if (!filePath) {
    return reply.status(404).send('File not found');
  }

  try {
    const data = fs.readFileSync(filePath);
    reply.header('Cache-Control', cacheControl);
    reply.type(contentType);
    return reply.send(data);
  } catch (err) {
    return reply.status(500).send(`Error reading file: ${err}`);
  }
}

export const publicRoutes: FastifyPluginAsync = async (app) => {
  // 1. Home / Landing Page
  app.get('/', async (_request, reply) => {
    return serveFile(reply, 'index.html', 'text/html; charset=utf-8', 'no-cache');
  });

  // 2. Documentation Page
  app.get('/docs', async (_request, reply) => {
    return serveFile(reply, 'docs.html', 'text/html; charset=utf-8', 'no-cache');
  });

  // 3. Privacy Policy
  app.get('/privacy', async (_request, reply) => {
    return serveFile(reply, 'privacy.html', 'text/html; charset=utf-8', 'no-cache');
  });

  // 4. Terms of Service
  app.get('/terms', async (_request, reply) => {
    return serveFile(reply, 'terms.html', 'text/html; charset=utf-8', 'no-cache');
  });

  // 4. Stylesheet
  app.get('/style.css', async (_request, reply) => {
    return serveFile(reply, 'style.css', 'text/css; charset=utf-8', 'public, max-age=86400');
  });

  // 5. Static Images
  app.get('/mascot.jpg', async (_request, reply) => {
    return serveFile(reply, 'mascot.jpg', 'image/jpeg', 'public, max-age=86400');
  });

  app.get('/hero.jpg', async (_request, reply) => {
    return serveFile(reply, 'hero.jpg', 'image/jpeg', 'public, max-age=86400');
  });

  // 6. Google Site Verification Files
  app.get('/google:hash.html', async (request, reply) => {
    const { hash } = request.params as { hash: string };
    return serveFile(reply, `google${hash}.html`, 'text/html; charset=utf-8', 'no-cache');
  });
};

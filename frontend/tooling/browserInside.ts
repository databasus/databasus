import { execFileSync } from 'node:child_process';
import { once } from 'node:events';
import { readFile } from 'node:fs/promises';
import { preview } from 'vite';

import { runChild } from './runChild.ts';
import { verifyBrowserProbes } from './verifyBrowserProbes.ts';

const environment = JSON.parse(await readFile('browser/environment.json', 'utf8'));
if (
  process.versions.node !== environment.nodeVersion ||
  execFileSync('pnpm', ['--version'], { encoding: 'utf8' }).trim() !== environment.pnpmVersion
)
  throw new Error('Browser runtime versions disagree with the pinned environment');

const built = await runChild('pnpm', ['build']);
if (built) process.exit(built);
const server = await preview({ preview: { host: '127.0.0.1', port: 0, strictPort: true } });
try {
  if (!server.httpServer.listening) await once(server.httpServer, 'listening');
  const address = server.httpServer.address();
  if (!address || typeof address === 'string')
    throw new Error('Browser preview has no bound TCP port');
  const baseURL = `http://127.0.0.1:${address.port}`;
  if (!(await fetch(baseURL, { signal: AbortSignal.timeout(10000) })).ok)
    throw new Error('Production preview readiness failed');

  process.env.BROWSER_BASE_URL = baseURL;
  await verifyBrowserProbes();
  process.exitCode = await runChild('pnpm', ['exec', 'playwright', 'test', '--project=ui']);
} finally {
  await new Promise<void>((resolve, reject) =>
    server.httpServer.close((error) => (error ? reject(error) : resolve())),
  );
}

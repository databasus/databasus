import { createHash } from 'node:crypto';
import { cp, mkdir, readFile } from 'node:fs/promises';
import { basename, dirname, join, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

import { removeBrowserContainer } from './removeBrowserContainer.ts';
import { runChild } from './runChild.ts';

const frontendRoot = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const runRoot = process.env.DATABASUS_BROWSER_RUN_ROOT;
if (!runRoot) throw new Error('Missing browser owner directory');
const environment = JSON.parse(
  await readFile(join(frontendRoot, 'browser/environment.json'), 'utf8'),
);
const inputs = ['browser/Dockerfile', 'browser/environment.json', 'package.json', 'pnpm-lock.yaml'];
const identity = createHash('sha256');
for (const input of inputs) identity.update(await readFile(join(frontendRoot, input)));
const image = `databasus-browser-ui:${identity.digest('hex').slice(0, 24)}`;
const container = `databasus-browser-e2e-${process.env.DATABASUS_BROWSER_RUN_ID ?? basename(runRoot)}`;
const workspace = join(runRoot, 'frontend');
const controller = new AbortController();
const interrupt = () => controller.abort();
process.on('SIGINT', interrupt);
process.on('SIGTERM', interrupt);
try {
  if (process.argv[2] === 'build') {
    const context = join(runRoot, 'image');
    await mkdir(context, { recursive: true });
    for (const input of ['package.json', 'pnpm-lock.yaml'])
      await cp(join(frontendRoot, input), join(context, input));
    await cp(join(frontendRoot, 'browser/Dockerfile'), join(context, 'Dockerfile'));
    const exitCode = await runChild(
      'docker',
      [
        'build',
        '--platform',
        environment.platform,
        '--build-arg',
        `NODE_VERSION=${environment.nodeVersion}`,
        '--build-arg',
        `NODE_ARCHIVE_SHA256=${environment.nodeArchiveSha256}`,
        '--build-arg',
        `PNPM_VERSION=${environment.pnpmVersion}`,
        '--tag',
        image,
        context,
      ],
      { signal: controller.signal },
    );
    if (exitCode) throw new Error(`Browser dependency build failed: ${exitCode}`);
    const excluded = new Set([
      'node_modules',
      'dist',
      'browser-results',
      'playwright-report',
      'test-results',
      '.git',
    ]);
    await cp(frontendRoot, workspace, {
      recursive: true,
      filter: (path) => !excluded.has(basename(path)) && !path.endsWith('.tsbuildinfo'),
    });
    await mkdir(join(runRoot, 'home'), { recursive: true });
  }
  const phase = process.argv[2];
  if (!['build', 'run'].includes(phase)) throw new Error('Unknown internal browser phase');
  if (phase === 'run') {
    const path =
      process.env.DATABASUS_BROWSER_PROBE === 'readiness'
        ? '/api/v1/undeclared-readiness-probe'
        : '/api/v1/system/version';
    const readiness = await fetch(`${process.env.BROWSER_BASE_URL}${path}`, {
      signal: AbortSignal.timeout(10_000),
    });
    if (!readiness.ok) throw new Error(`E2E readiness failure: HTTP ${readiness.status}`);
  }
  process.exitCode = await runChild(
    'docker',
    [
      'run',
      '--rm',
      '--init',
      '--network=host',
      '--name',
      container,
      '--platform',
      environment.platform,
      '--shm-size=1g',
      '--user',
      `${process.getuid?.() ?? 1000}:${process.getgid?.() ?? 1000}`,
      '--volume',
      `${runRoot}:/work:Z`,
      '--workdir',
      '/work/frontend',
      '--env',
      'HOME=/work/home',
      '--env',
      'CI=1',
      ...['BROWSER_BASE_URL', 'E2E_EMAIL', 'E2E_PASSWORD', 'DATABASUS_BROWSER_PROBE'].flatMap(
        (name) => ['--env', name],
      ),
      image,
      'sh',
      '-c',
      phase === 'build'
        ? 'cp -a /opt/dependencies/node_modules ./node_modules && pnpm build'
        : 'pnpm exec playwright test --project=e2e',
    ],
    { signal: controller.signal },
  );
} finally {
  await removeBrowserContainer(container);
  process.off('SIGINT', interrupt);
  process.off('SIGTERM', interrupt);
}

import { createHash, randomUUID } from 'node:crypto';
import { cp, mkdir, mkdtemp, readFile, rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { basename, dirname, join, resolve } from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';

import { runChild } from './runChild.ts';

export const runBrowser = async ({
  frontendRoot,
  args,
}: {
  frontendRoot: string;
  args: string[];
}): Promise<number> => {
  if (!args.includes('--project=ui')) throw new Error('Select exactly --project=ui.');
  const isUpdatingReferences = args.includes('--update-references');
  if (args.some((argument) => !['--project=ui', '--update-references'].includes(argument)))
    throw new Error('Unsupported browser argument; use the documented UI entry points.');

  const environment = JSON.parse(
    await readFile(join(frontendRoot, 'browser/environment.json'), 'utf8'),
  );
  const mise = await readFile(join(frontendRoot, '../mise.toml'), 'utf8');
  if (
    !mise.includes(`node = "${environment.nodeVersion}"`) ||
    !mise.includes(`pnpm = "${environment.pnpmVersion}"`)
  )
    throw new Error('Browser tool versions disagree with mise.toml');

  const inputs = [
    'browser/Dockerfile',
    'browser/environment.json',
    'package.json',
    'pnpm-lock.yaml',
  ];
  const identity = createHash('sha256');
  for (const input of inputs) identity.update(await readFile(join(frontendRoot, input)));
  const image = `databasus-browser-ui:${identity.digest('hex').slice(0, 24)}`;

  const runId = randomUUID();
  const container = `databasus-browser-ui-${runId}`;
  const temporaryRoot = await mkdtemp(join(tmpdir(), 'databasus-browser-ui-'));
  const workspace = join(temporaryRoot, 'frontend');
  const controller = new AbortController();
  let isInterrupted = false;
  let resultStatus = 1;
  let operationError: unknown;

  const interrupt = () => {
    isInterrupted = true;
    controller.abort();
  };
  process.on('SIGINT', interrupt);
  process.on('SIGTERM', interrupt);

  try {
    const context = join(temporaryRoot, 'image');
    await mkdir(context);
    for (const input of ['package.json', 'pnpm-lock.yaml'])
      await cp(join(frontendRoot, input), join(context, input));
    await cp(join(frontendRoot, 'browser/Dockerfile'), join(context, 'Dockerfile'));

    const built = await runChild(
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
    if (built) resultStatus = built;
    else {
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
      await mkdir(join(temporaryRoot, 'home'));
      console.log(
        `Browser environment: ${environment.playwrightImage}; Node ${environment.nodeVersion}; pnpm ${environment.pnpmVersion}; run ${runId}`,
      );

      const status = await runChild(
        'docker',
        [
          'run',
          '--rm',
          '--init',
          '--name',
          container,
          '--platform',
          environment.platform,
          '--shm-size=1g',
          '--user',
          `${process.getuid?.() ?? 1000}:${process.getgid?.() ?? 1000}`,
          '--volume',
          `${temporaryRoot}:/work:Z`,
          '--workdir',
          '/work/frontend',
          '--env',
          'HOME=/work/home',
          '--env',
          'CI=1',
          '--env',
          `UPDATE_REFERENCES=${isUpdatingReferences ? '1' : '0'}`,
          image,
          'sh',
          '-c',
          'cp -a /opt/dependencies/node_modules ./node_modules && node tooling/browserInside.ts',
        ],
        { signal: controller.signal },
      );
      if (isUpdatingReferences && status === 0)
        await cp(join(workspace, 'browser/references'), join(frontendRoot, 'browser/references'), {
          recursive: true,
        });
      resultStatus = status;
    }
  } catch (error) {
    if (isInterrupted) resultStatus = 130;
    else operationError = error;
  } finally {
    try {
      const removed = await runChild('docker', ['rm', '--force', container]);
      if (removed)
        operationError ??= new Error(`Could not remove owned browser container: ${container}`);
    } catch (error) {
      operationError ??= error;
    }

    const artifacts = join(frontendRoot, 'browser-results', runId);
    try {
      await mkdir(artifacts, { recursive: true });
      await cp(join(workspace, 'test-results'), artifacts, { recursive: true });
    } catch (error) {
      if ((error as NodeJS.ErrnoException).code !== 'ENOENT')
        console.error('Could not retain browser artifacts:', error);
    }

    try {
      await rm(temporaryRoot, { recursive: true, force: true });
    } catch (error) {
      operationError ??= error;
    }

    process.off('SIGINT', interrupt);
    process.off('SIGTERM', interrupt);
  }
  if (operationError) throw operationError;
  return resultStatus;
};

if (process.argv[1] && pathToFileURL(resolve(process.argv[1])).href === import.meta.url) {
  try {
    process.exitCode = await runBrowser({
      frontendRoot: resolve(dirname(fileURLToPath(import.meta.url)), '..'),
      args: process.argv.slice(2),
    });
  } catch (error) {
    console.error(error instanceof Error ? error.message : error);
    process.exitCode = 1;
  }
}

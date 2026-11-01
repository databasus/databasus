import { spawn } from 'node:child_process';
import { mkdtemp, mkdir, writeFile, readdir, rm, cp } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join, resolve } from 'node:path';
import { pathToFileURL } from 'node:url';
import { describe, expect, it } from 'vitest';

const runFixture = async ({
  mode,
  isInterrupt = false,
}: {
  mode: string;
  isInterrupt?: boolean;
}) => {
  const directory = await mkdtemp(join(tmpdir(), 'databasus-browser-fixture-'));
  const root = join(directory, 'frontend');
  const markers = join(directory, 'containers');
  const binary = join(directory, 'bin');

  try {
    await Promise.all([
      mkdir(join(root, 'browser'), { recursive: true }),
      mkdir(markers),
      mkdir(binary),
    ]);

    for (const file of [
      'browser/Dockerfile',
      'browser/environment.json',
      'package.json',
      'pnpm-lock.yaml',
    ])
      await cp(resolve(file), join(root, file));
    await cp(resolve('../mise.toml'), join(directory, 'mise.toml'));

    const docker = `#!${process.execPath}
const fs = require('node:fs');
const path = require('node:path');
const http = require('node:http');
const args = process.argv.slice(2);
const directory = process.env.FIXTURE_CONTAINERS;
if(args[0] === 'build') process.exit(0);
if(args[0] === 'rm') { const marker=path.join(directory,args.at(-1)); if(fs.existsSync(marker)){ const pid=Number(fs.readFileSync(marker)); try { process.kill(pid,'SIGTERM'); } catch {} fs.unlinkSync(marker); } process.exit(0); }
const name = args[args.indexOf('--name')+1];
const mount = args[args.indexOf('--volume')+1].split(':/work')[0];
fs.writeFileSync(path.join(directory,name),String(process.pid));
console.log('OWNED_READY '+mount);
if(process.env.FIXTURE_MODE === 'scenario') { console.error('deliberate scenario failure'); process.exit(17); }
if(process.env.FIXTURE_MODE === 'child-exit') { console.error('deliberate child exit'); process.exit(23); }
if(process.env.FIXTURE_MODE === 'success') process.exit(0);
http.createServer((request,response)=>response.end()).listen(0,'127.0.0.1');
`;
    await writeFile(join(binary, 'docker'), docker, { mode: 0o755 });

    const entry = join(directory, 'fixture.mjs');
    await writeFile(
      entry,
      `import { runBrowser } from ${JSON.stringify(pathToFileURL(resolve('tooling/runBrowser.ts')).href)}; process.exitCode = await runBrowser({ frontendRoot: ${JSON.stringify(root)}, args: ['--project=ui'] });`,
    );

    const child = spawn(process.execPath, [entry], {
      env: {
        ...process.env,
        PATH: `${binary}:${process.env.PATH}`,
        FIXTURE_CONTAINERS: markers,
        FIXTURE_MODE: mode,
      },
      stdio: ['ignore', 'pipe', 'pipe'],
    });

    let output = '';
    let runRoot: string | undefined;
    const deadline = setTimeout(() => child.kill('SIGTERM'), 10_000);

    try {
      const exitCode = await new Promise<number | null>((resolveExit, reject) => {
        child.once('error', reject);
        child.stdout.on('data', (chunk) => {
          output += chunk;
          const ready = output.match(/OWNED_READY ([^\n]+)/);
          if (ready && !runRoot) {
            runRoot = ready[1];
            if (isInterrupt) child.kill('SIGINT');
          }
        });
        child.stderr.on('data', (chunk) => {
          output += chunk;
        });
        child.once('exit', resolveExit);
      });

      expect(await readdir(markers)).toEqual([]);
      expect(runRoot).toBeTruthy();
      await expect(readdir(runRoot!)).rejects.toMatchObject({ code: 'ENOENT' });

      return { exitCode, output };
    } finally {
      clearTimeout(deadline);
      if (child.exitCode === null && child.signalCode === null) child.kill('SIGTERM');
    }
  } finally {
    await rm(directory, { recursive: true, force: true });
  }
};

describe('Browser runner resource ownership', () => {
  it.each([
    { mode: 'scenario', status: 17, diagnostic: 'deliberate scenario failure' },
    { mode: 'child-exit', status: 23, diagnostic: 'deliberate child exit' },
  ])(
    'propagates failure and removes owned resources when $mode fails',
    async ({ mode, status, diagnostic }) => {
      const failure = await runFixture({ mode });
      expect(failure.exitCode).toBe(status);
      expect(failure.output).toContain(diagnostic);
      expect((await runFixture({ mode: 'success' })).exitCode).toBe(0);
    },
  );
  it('cleans owned containers and temporary state when the runner is interrupted', async () => {
    const interrupted = await runFixture({ mode: 'pending', isInterrupt: true });
    expect(interrupted.exitCode).toBe(130);
    expect((await runFixture({ mode: 'success' })).exitCode).toBe(0);
  });
  it('rejects unknown projects when no supported project is selected', async () => {
    const { runBrowser } = await import('./runBrowser.ts');
    await expect(
      runBrowser({ frontendRoot: resolve('.'), args: ['--project=unknown'] }),
    ).rejects.toThrow('Select exactly');
  });
});

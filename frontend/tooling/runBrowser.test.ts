import { spawn } from 'node:child_process';
import { mkdtemp, mkdir, writeFile, readdir, rm, cp } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join, resolve } from 'node:path';
import { pathToFileURL } from 'node:url';
import { describe, expect, it } from 'vitest';

const runFixture = async ({
  mode,
  isInterrupt = false,
  architecture = process.arch,
}: {
  mode: string;
  isInterrupt?: boolean;
  architecture?: string;
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
if(args[0] === 'build') { console.log('BUILD_ARGS '+JSON.stringify(args)); if(process.env.FIXTURE_MODE === 'checksum') { console.error('sha256sum: WARNING: computed checksum did NOT match'); process.exit(9); } process.exit(0); }
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
      `import { runBrowser } from ${JSON.stringify(pathToFileURL(resolve('tooling/runBrowser.ts')).href)}; process.exitCode = await runBrowser({ frontendRoot: ${JSON.stringify(root)}, args: ['--project=ui'], architecture: ${JSON.stringify(architecture)} });`,
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
      if (mode === 'checksum') expect(runRoot).toBeUndefined();
      else {
        expect(runRoot).toBeTruthy();
        await expect(readdir(runRoot!)).rejects.toMatchObject({ code: 'ENOENT' });
      }

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
    {
      architecture: 'x64',
      platform: 'linux/amd64',
      checksum: '6e1db87ef58b8819e5d5402eff1536491b18edd8eb7bee5ef7897876e88dc5ff',
    },
    {
      architecture: 'arm64',
      platform: 'linux/arm64',
      checksum: '724282c3b43aec998aa9527380465b45d229e021b58035f5f4f63095eabfe5d5',
    },
  ])(
    'selects native tools when running on $architecture',
    async ({ architecture, platform, checksum }) => {
      const nativeRun = await runFixture({ mode: 'success', architecture });

      expect(nativeRun.exitCode).toBe(0);
      expect(nativeRun.output).toContain(platform);
      expect(nativeRun.output).toContain(`NODE_ARCH=${architecture}`);
      expect(nativeRun.output).toContain(`NODE_ARCHIVE_SHA256=${checksum}`);
      const otherRun = await runFixture({
        mode: 'success',
        architecture: architecture === 'x64' ? 'arm64' : 'x64',
      });
      const imageTag = (output: string) => output.match(/databasus-browser-ui:[a-f0-9]+/)?.[0];
      expect(imageTag(nativeRun.output)).toBeTruthy();
      expect(imageTag(nativeRun.output)).not.toBe(imageTag(otherRun.output));
    },
  );
  it('fails before scenarios when the downloaded Node archive fails checksum validation', async () => {
    const failedBuild = await runFixture({ mode: 'checksum' });

    expect(failedBuild.exitCode).toBe(9);
    expect(failedBuild.output).toContain('checksum did NOT match');
  });
  it('rejects an unsupported architecture before starting Docker', async () => {
    const { runBrowser } = await import('./runBrowser.ts');

    await expect(
      runBrowser({ frontendRoot: resolve('.'), args: ['--project=ui'], architecture: 'riscv64' }),
    ).rejects.toThrow('Unsupported browser architecture: riscv64');
  });
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

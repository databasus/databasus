import { spawn } from 'node:child_process';
import { execFile } from 'node:child_process';
import { once } from 'node:events';
import { mkdir, mkdtemp, readFile, readdir, rm, writeFile } from 'node:fs/promises';
import { createServer } from 'node:http';
import { tmpdir } from 'node:os';
import { join, resolve } from 'node:path';
import { pathToFileURL } from 'node:url';
import { promisify } from 'node:util';
import { describe, expect, it, vi } from 'vitest';

const execute = promisify(execFile);
const startPilot = (probe?: string) => {
  const child = spawn(process.execPath, [resolve('tooling/runBrowser.ts'), '--project=e2e'], {
    env: { ...process.env, DATABASUS_BROWSER_PROBE: probe },
    stdio: ['ignore', 'pipe', 'pipe'],
  });
  let output = '';
  let isInterrupted = false;
  const record = (chunk: Buffer) => {
    output += chunk.toString();
    if (probe === 'interrupt' && output.includes('E2E_INTERRUPT_READY') && !isInterrupted) {
      isInterrupted = true;
      child.kill('SIGTERM');
    }
  };
  child.stdout.on('data', record);
  child.stderr.on('data', record);
  const deadline = setTimeout(() => child.kill('SIGTERM'), 120_000);
  const completed = new Promise<{ exitCode: number | null; output: string }>(
    (resolveExit, reject) => {
      child.once('error', reject);
      child.once('exit', (exitCode) => {
        clearTimeout(deadline);
        resolveExit({ exitCode, output });
      });
    },
  );
  return { child, completed };
};

const assertReleased = async (output: string) => {
  const ownership = output.match(/Browser E2E owner: ([\w-]+); directory: ([^\n]+)/);
  expect(ownership, output.slice(-2000)).toBeTruthy();
  await expect(readdir(ownership![2])).rejects.toMatchObject({ code: 'ENOENT' });
  await expect(
    execute('docker', ['inspect', `databasus-browser-e2e-${ownership![1]}`]),
  ).rejects.toBeTruthy();
  const gateway = output.match(/Browser gateway ready: (http:\/\/127\.0\.0\.1:\d+)/);
  if (gateway)
    await expect(fetch(gateway[1], { signal: AbortSignal.timeout(1000) })).rejects.toBeTruthy();
  const owner = output.match(/Browser owner PID: (\d+)/);
  if (owner) expect(() => process.kill(Number(owner[1]), 0)).toThrow();
  const evidenceDirectory = resolve('browser-results', `e2e-${ownership![1]}`);
  expect(await readdir(evidenceDirectory)).toEqual(['status.json']);
  const evidence = await readFile(resolve(evidenceDirectory, 'status.json'), 'utf8');
  expect(evidence).not.toMatch(
    /test-password-123|oauth-session|private-code|private-state|Authorization|synthetic-secret-marker/,
  );
};

describe('Real-API browser lifecycle', () => {
  it.each([
    { probe: 'scenario', diagnostic: 'deliberate E2E scenario failure after readiness' },
    { probe: 'readiness', diagnostic: 'E2E readiness failure: HTTP 404' },
    { probe: 'interrupt', diagnostic: 'E2E_INTERRUPT_READY' },
  ])(
    'releases owned resources when the public entry encounters $probe',
    async ({ probe, diagnostic }) => {
      const unrelated = createServer((_, response) => response.end('unrelated-service'));
      unrelated.listen(0, '127.0.0.1');
      await once(unrelated, 'listening');
      const address = unrelated.address();
      if (!address || typeof address === 'string')
        throw new Error('Unrelated service has no address');
      const pilot = startPilot(probe);
      try {
        const stopped = await pilot.completed;
        expect(stopped.exitCode).not.toBe(0);
        expect(stopped.output).toContain(diagnostic);
        await assertReleased(stopped.output);
        expect(await (await fetch(`http://127.0.0.1:${address.port}`)).text()).toBe(
          'unrelated-service',
        );
      } finally {
        if (pilot.child.exitCode === null && pilot.child.signalCode === null)
          pilot.child.kill('SIGTERM');
        await new Promise<void>((resolveClose) => unrelated.close(() => resolveClose()));
      }
    },
    180_000,
  );
  it('isolates application state and resources when two public runs overlap', async () => {
    const firstPilot = startPilot();
    const secondPilot = startPilot();
    try {
      const pilots = await Promise.all([firstPilot.completed, secondPilot.completed]);
      for (const pilot of pilots) {
        expect(pilot.exitCode, pilot.output.slice(-3000)).toBe(0);
        await assertReleased(pilot.output);
      }
      expect(pilots[0].output.match(/Browser gateway ready: (\S+)/)?.[1]).not.toBe(
        pilots[1].output.match(/Browser gateway ready: (\S+)/)?.[1],
      );
    } finally {
      for (const pilot of [firstPilot, secondPilot])
        if (pilot.child.exitCode === null && pilot.child.signalCode === null)
          pilot.child.kill('SIGTERM');
    }
  }, 180_000);
});

it('fails the supervisor when the external container runtime refuses cleanup and still lists the owned container', async () => {
  const directory = await mkdtemp(join(tmpdir(), 'e2e-cleanup-refusal-'));
  const frontendRoot = join(directory, 'frontend');
  const binary = join(directory, 'bin');
  try {
    await Promise.all([mkdir(frontendRoot), mkdir(join(directory, 'backend')), mkdir(binary)]);
    await writeFile(
      join(binary, 'go'),
      `#!${process.execPath}
process.exit(0);
`,
      { mode: 0o755 },
    );
    await writeFile(
      join(binary, 'docker'),
      `#!${process.execPath}
if(process.argv[2] === 'rm') { console.error('container removal refused'); process.exit(1); }
console.log('owned-container-id');
`,
      { mode: 0o755 },
    );
    const entry = join(directory, 'entry.mjs');
    await writeFile(
      entry,
      `import { runBrowser } from ${JSON.stringify(pathToFileURL(resolve('tooling/runBrowser.ts')).href)}; process.exitCode = await runBrowser({ frontendRoot: ${JSON.stringify(frontendRoot)}, args: ['--project=e2e'] });`,
    );
    let output = '';
    try {
      await execute(process.execPath, [entry], {
        env: { ...process.env, PATH: `${binary}:${process.env.PATH}` },
        timeout: 15_000,
      });
      throw new Error('Supervisor accepted incomplete cleanup');
    } catch (error) {
      const failure = error as Error & { code: number; stdout: string; stderr: string };
      expect(failure.code).toBe(1);
      expect(failure.stderr).toContain('Owned browser container remains after cleanup');
      output = failure.stdout;
    }
    const ownership = output.match(/Browser E2E owner: ([\w-]+); directory: ([^\n]+)/);
    expect(ownership).toBeTruthy();
    const status = JSON.parse(
      await readFile(
        join(frontendRoot, 'browser-results', `e2e-${ownership![1]}`, 'status.json'),
        'utf8',
      ),
    );
    expect(status.exitCode).toBe(1);
    await expect(readdir(ownership![2])).rejects.toMatchObject({ code: 'ENOENT' });
  } finally {
    await rm(directory, { recursive: true, force: true });
  }
});

it.each(['SIGINT', 'SIGTERM'] as const)(
  'retains an interrupted status when %s arrives after the test driver succeeds during container cleanup',
  async (signal) => {
    const directory = await mkdtemp(join(tmpdir(), 'e2e-cleanup-interrupt-'));
    const frontendRoot = join(directory, 'frontend');
    const binary = join(directory, 'bin');
    const cleanupReady = join(directory, 'cleanup-ready');
    const cleanupRelease = join(directory, 'cleanup-release');
    let child: ReturnType<typeof spawn> | undefined;
    let completed: Promise<[number | null, NodeJS.Signals | null]> | undefined;
    const stopSupervisor = () => {
      if (child?.pid && child.exitCode === null && child.signalCode === null) {
        try {
          process.kill(-child.pid, 'SIGKILL');
        } catch (error) {
          if ((error as NodeJS.ErrnoException).code !== 'ESRCH') throw error;
        }
      }
    };
    try {
      await Promise.all([mkdir(frontendRoot), mkdir(join(directory, 'backend')), mkdir(binary)]);
      await writeFile(join(binary, 'go'), `#!${process.execPath}\nprocess.exit(0);\n`, {
        mode: 0o755,
      });
      await writeFile(
        join(binary, 'docker'),
        `#!${process.execPath}
import { existsSync, watch, writeFileSync } from 'node:fs';
if (process.argv[2] === 'rm') {
  const watcher = watch(${JSON.stringify(directory)}, () => {
    if (existsSync(${JSON.stringify(cleanupRelease)})) watcher.close();
  });
  writeFileSync(${JSON.stringify(cleanupReady)}, 'ready');
}
`,
        { mode: 0o755 },
      );
      const entry = join(directory, 'entry.mjs');
      await writeFile(
        entry,
        `import { runBrowser } from ${JSON.stringify(pathToFileURL(resolve('tooling/runBrowser.ts')).href)};
process.on(${JSON.stringify(signal)}, () => process.send('interrupted'));
process.exitCode = await runBrowser({ frontendRoot: ${JSON.stringify(frontendRoot)}, args: ['--project=e2e'] });
process.disconnect();
`,
      );
      child = spawn(process.execPath, [entry], {
        env: { ...process.env, PATH: `${binary}:${process.env.PATH}` },
        stdio: ['ignore', 'pipe', 'pipe', 'ipc'],
        detached: true,
        timeout: 15_000,
        killSignal: 'SIGKILL',
      });
      completed = once(child, 'exit') as Promise<[number | null, NodeJS.Signals | null]>;
      let output = '';
      child.stdout!.on('data', (chunk: Buffer) => (output += chunk.toString()));
      child.stderr!.on('data', (chunk: Buffer) => (output += chunk.toString()));
      await vi.waitFor(async () => expect(await readFile(cleanupReady, 'utf8')).toBe('ready'), {
        timeout: 5_000,
      });
      const interrupted = once(child, 'message');
      expect(child.kill(signal)).toBe(true);
      expect(
        await Promise.race([
          interrupted,
          completed.then(() => {
            throw new Error('Supervisor exited before acknowledging interruption');
          }),
        ]),
      ).toEqual(['interrupted', undefined]);
      await writeFile(cleanupRelease, 'release');

      const [exitCode, exitSignal] = await completed;
      expect(exitSignal, output).toBeNull();
      expect(exitCode, output).toBe(130);
      const ownership = output.match(/Browser E2E owner: ([\w-]+); directory: ([^\n]+)/);
      expect(ownership, output).toBeTruthy();
      const status = JSON.parse(
        await readFile(
          join(frontendRoot, 'browser-results', `e2e-${ownership![1]}`, 'status.json'),
          'utf8',
        ),
      );
      expect(status).toMatchObject({ exitCode: 130, isInterrupted: true });
      await expect(readdir(ownership![2])).rejects.toMatchObject({ code: 'ENOENT' });
    } finally {
      try {
        await writeFile(cleanupRelease, 'release');
        stopSupervisor();
        await completed;
      } finally {
        await rm(directory, { recursive: true, force: true });
      }
    }
  },
  20_000,
);

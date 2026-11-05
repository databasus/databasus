import { mkdtemp, readFile, rm, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { expect, it } from 'vitest';

import { waitForCondition } from '../src/shared/lib/test/conditionWait';
import { runChild } from './runChild';

it('waits for descendant shutdown after an interrupted process-group driver exits', async () => {
  const directory = await mkdtemp(join(tmpdir(), 'browser-process-group-'));
  const descendantPath = join(directory, 'descendant.mjs');
  const driverPath = join(directory, 'driver.mjs');
  const releasePath = join(directory, 'release');

  await writeFile(
    descendantPath,
    `
import { createServer } from 'node:http';
import { watch, writeFileSync } from 'node:fs';
const directory = ${JSON.stringify(directory)};
const server = createServer((_, response) => response.end('owned descendant'));
process.on('SIGTERM', () => {
  const watcher = watch(directory, (_, name) => {
    if (name !== 'release') return;
    watcher.close();
    server.close(() => process.exit(0));
  });
  writeFileSync(directory + '/stopping', 'cleanup started');
});
server.listen(0, '127.0.0.1', () => writeFileSync(directory + '/ready.json', JSON.stringify({ driverPid: process.ppid, address: 'http://127.0.0.1:' + server.address().port })));
`,
  );
  await writeFile(
    driverPath,
    `import { spawn } from 'node:child_process'; spawn(process.execPath, [${JSON.stringify(descendantPath)}], { stdio: 'inherit' });`,
  );

  const controller = new AbortController();
  let hasCompleted = false;
  const shutdown = runChild(process.execPath, [driverPath], {
    signal: controller.signal,
    shouldStopProcessGroup: true,
  }).then((exitCode) => {
    hasCompleted = true;
    return exitCode;
  });

  let driverPid: number | undefined;
  try {
    const ownedProcess = await waitForCondition(
      async () =>
        JSON.parse(await readFile(join(directory, 'ready.json'), 'utf8')) as {
          driverPid: number;
          address: string;
        },
    );
    driverPid = ownedProcess.driverPid;

    controller.abort();
    await waitForCondition(async () =>
      expect(await readFile(join(directory, 'stopping'), 'utf8')).toBe('cleanup started'),
    );
    await waitForCondition(() => expect(() => process.kill(ownedProcess.driverPid, 0)).toThrow());

    expect(hasCompleted).toBe(false);
    expect((await fetch(ownedProcess.address)).status).toBe(200);

    await writeFile(releasePath, 'finish cleanup');
    expect(await shutdown).toBe(130);
    await expect(fetch(ownedProcess.address)).rejects.toBeTruthy();
  } finally {
    controller.abort();
    await writeFile(releasePath, 'finish cleanup');
    if (driverPid) {
      try {
        process.kill(-driverPid, 'SIGKILL');
      } catch {}
    }
    await shutdown.catch(() => {});
    await rm(directory, { recursive: true, force: true });
  }
});

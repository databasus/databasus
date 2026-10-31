import { randomUUID } from 'node:crypto';
import { mkdir, mkdtemp, rm, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join, resolve } from 'node:path';

import { removeBrowserContainer } from './removeBrowserContainer.ts';
import { runChild } from './runChild.ts';

export const runBrowserE2E = async (frontendRoot: string): Promise<number> => {
  const runId = randomUUID();
  const runRoot = await mkdtemp(join(tmpdir(), 'databasus-browser-e2e-'));
  const controller = new AbortController();
  let isInterrupted = false;
  const interrupt = () => {
    isInterrupted = true;
    controller.abort();
  };
  process.on('SIGINT', interrupt);
  process.on('SIGTERM', interrupt);
  console.log(`Browser E2E owner: ${runId}; directory: ${runRoot}`);
  let exitCode = 1;
  let failure: unknown;
  try {
    exitCode = await runChild(
      'go',
      ['test', '-race', '-count=1', '-timeout=30m', '-v', './internal/app/browsertests'],
      {
        cwd: resolve(frontendRoot, '../backend'),
        signal: controller.signal,
        shouldStopProcessGroup: true,
        env: {
          ...process.env,
          DATABASUS_BROWSER_RUN_ROOT: runRoot,
          DATABASUS_BROWSER_RUN_ID: runId,
        },
      },
    );
  } catch (error) {
    failure = error;
    exitCode = isInterrupted ? 130 : 1;
  }
  try {
    await removeBrowserContainer(`databasus-browser-e2e-${runId}`);
  } catch (error) {
    failure ??= error;
    exitCode = exitCode || 1;
  }
  try {
    await rm(runRoot, { recursive: true, force: true });
  } catch (error) {
    failure ??= error;
    exitCode = exitCode || 1;
  }
  try {
    const evidence = join(frontendRoot, 'browser-results', `e2e-${runId}`);
    await mkdir(evidence, { recursive: true });
    while (true) {
      if (isInterrupted) exitCode = 130;
      const status: { runId: string; exitCode: number; isInterrupted: boolean } = {
        runId,
        exitCode,
        isInterrupted,
      };
      await writeFile(join(evidence, 'status.json'), JSON.stringify(status));
      if (status.isInterrupted === isInterrupted) break;
    }
  } finally {
    process.off('SIGINT', interrupt);
    process.off('SIGTERM', interrupt);
  }
  if (failure) throw failure;
  return exitCode;
};

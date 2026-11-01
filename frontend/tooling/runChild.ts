import { spawn } from 'node:child_process';
import { setTimeout as wait } from 'node:timers/promises';

const waitForProcessGroup = async (processGroupId: number): Promise<void> => {
  const deadline = Date.now() + 12_000;
  while (true) {
    try {
      process.kill(-processGroupId, 0);
    } catch (error) {
      if ((error as NodeJS.ErrnoException).code === 'ESRCH') return;
      throw error;
    }
    if (Date.now() >= deadline)
      throw new Error(`Owned process group did not stop: ${processGroupId}`);
    await wait(25);
  }
};

export const runChild = (
  command: string,
  args: string[],
  options: {
    cwd?: string;
    signal?: AbortSignal;
    env?: NodeJS.ProcessEnv;
    shouldStopProcessGroup?: boolean;
  } = {},
): Promise<number> =>
  new Promise((resolveExit, reject) => {
    const { shouldStopProcessGroup, signal, ...spawnOptions } = options;
    const child = spawn(command, args, {
      ...spawnOptions,
      detached: shouldStopProcessGroup,
      stdio: 'inherit',
    });

    let deadline: ReturnType<typeof setTimeout> | undefined;
    const stop = () => {
      if (!child.pid) return;
      try {
        if (shouldStopProcessGroup) process.kill(-child.pid, 'SIGTERM');
        else child.kill('SIGTERM');
        deadline = setTimeout(() => {
          try {
            if (shouldStopProcessGroup) process.kill(-child.pid!, 'SIGKILL');
            else child.kill('SIGKILL');
          } catch {}
        }, 10_000);
      } catch {}
    };

    signal?.addEventListener('abort', stop, { once: true });
    if (signal?.aborted) stop();
    const cleanup = () => {
      signal?.removeEventListener('abort', stop);
      if (deadline) clearTimeout(deadline);
    };

    child.once('error', (error) => {
      cleanup();
      reject(error);
    });
    child.once('exit', (code, exitSignal) => {
      void (async () => {
        try {
          if (shouldStopProcessGroup && child.pid) {
            if (!deadline) stop();
            await waitForProcessGroup(child.pid);
          }
          resolveExit(signal?.aborted ? 130 : (code ?? (exitSignal ? 130 : 1)));
        } catch (error) {
          reject(error);
        } finally {
          cleanup();
        }
      })();
    });
  });

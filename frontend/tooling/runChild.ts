import { spawn } from 'node:child_process';

export const runChild = (
  command: string,
  args: string[],
  options: { cwd?: string; signal?: AbortSignal } = {},
): Promise<number> =>
  new Promise((resolveExit, reject) => {
    const child = spawn(command, args, { ...options, stdio: 'inherit' });
    child.once('error', reject);
    child.once('exit', (code, signal) => resolveExit(code ?? (signal ? 130 : 1)));
  });

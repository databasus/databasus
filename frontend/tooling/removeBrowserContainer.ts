import { execFile } from 'node:child_process';
import { promisify } from 'node:util';

const execute = promisify(execFile);

export const removeBrowserContainer = async (container: string): Promise<void> => {
  let removalFailure: unknown;
  try {
    await execute('docker', ['rm', '--force', container], { timeout: 10_000 });
  } catch (error) {
    removalFailure = error;
  }
  const { stdout } = await execute(
    'docker',
    ['container', 'ls', '--all', '--filter', `name=^/${container}$`, '--format', '{{.ID}}'],
    { timeout: 10_000 },
  );
  if (stdout.trim())
    throw new Error(`Owned browser container remains after cleanup: ${container}`, {
      cause: removalFailure,
    });
};

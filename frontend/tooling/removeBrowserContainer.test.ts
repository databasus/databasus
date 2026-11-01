import { execFile } from 'node:child_process';
import { mkdtemp, rm, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join, resolve } from 'node:path';
import { pathToFileURL } from 'node:url';
import { promisify } from 'node:util';
import { expect, it } from 'vitest';

const execute = promisify(execFile);
it.each([
  { mode: 'absent', isSuccessful: true },
  { mode: 'removed', isSuccessful: true },
  { mode: 'removal-refused', isSuccessful: false },
  { mode: 'listing-refused', isSuccessful: false },
])('verifies container absence when cleanup encounters $mode', async ({ mode, isSuccessful }) => {
  const directory = await mkdtemp(join(tmpdir(), 'browser-container-cleanup-'));
  try {
    await writeFile(
      join(directory, 'docker'),
      `#!${process.execPath}
if (process.argv[2] === 'rm') process.exit(${mode === 'removed' ? 0 : 1});
if (${JSON.stringify(mode)} === 'listing-refused') { console.error('daemon unavailable'); process.exit(1); }
if (${JSON.stringify(mode)} === 'removal-refused') console.log('owned-container-id');
`,
      { mode: 0o755 },
    );

    const entry = join(directory, 'cleanup.mjs');
    await writeFile(
      entry,
      `import { removeBrowserContainer } from ${JSON.stringify(pathToFileURL(resolve('tooling/removeBrowserContainer.ts')).href)}; await removeBrowserContainer('owned-probe');`,
    );

    const cleanup = execute(process.execPath, [entry], {
      env: { ...process.env, PATH: `${directory}:${process.env.PATH}` },
    });

    if (isSuccessful) await expect(cleanup).resolves.toMatchObject({ stdout: '' });
    else
      await expect(cleanup).rejects.toMatchObject({
        stderr: expect.stringContaining(
          mode === 'removal-refused' ? 'Owned browser container remains' : 'daemon unavailable',
        ),
      });
  } finally {
    await rm(directory, { recursive: true, force: true });
  }
});

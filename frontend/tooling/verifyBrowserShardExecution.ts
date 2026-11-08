import { execFile } from 'node:child_process';
import { cp, mkdir, mkdtemp, readFile, rm, symlink, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { basename, join, resolve } from 'node:path';
import { promisify } from 'node:util';

const execute = promisify(execFile);
const frontendRoot = resolve('.');
const directory = await mkdtemp(join(tmpdir(), 'databasus-inner-shards-'));

try {
  await cp(join(frontendRoot, 'browser'), join(directory, 'browser'), {
    recursive: true,
    filter: (path) => !['scenarios', 'references'].includes(basename(path)),
  });
  await mkdir(join(directory, 'browser/scenarios'));
  for (const name of ['node_modules', 'src'])
    await symlink(join(frontendRoot, name), join(directory, name), 'dir');
  for (const name of ['playwright.config.ts', 'tsconfig.node.json'])
    await cp(join(frontendRoot, name), join(directory, name));
  await writeFile(
    join(directory, 'package.json'),
    JSON.stringify({ type: 'module', scripts: { build: 'vite build' } }),
  );
  await writeFile(
    join(directory, 'index.html'),
    '<!doctype html><html lang="en"><head><title>Inner runner fixture</title></head><body><p>Inner runner fixture</p></body></html>',
  );
  await writeFile(
    join(directory, 'browser/scenarios/selection.spec.ts'),
    `import { appendFileSync } from 'node:fs';
import { test } from '@playwright/test';
for (const identity of [1, 2, 3, 4, 5, 6]) {
  test('records selected case ' + identity, () => {
    appendFileSync(process.env.BROWSER_SHARD_EXECUTION_RECORD, identity + '\\n');
  });
}
`,
  );

  const selections = [
    { shard: '', identities: ['1', '2', '3', '4', '5', '6'] },
    { shard: '1/3', identities: ['1', '2'] },
    { shard: '2/3', identities: ['3', '4'] },
    { shard: '3/3', identities: ['5', '6'] },
  ];
  for (const [index, selection] of selections.entries()) {
    const record = join(directory, `executed-${index}.txt`);
    try {
      await execute(process.execPath, [join(frontendRoot, 'tooling/browserInside.ts')], {
        cwd: directory,
        env: {
          ...process.env,
          BROWSER_SHARD: selection.shard,
          BROWSER_SHARD_EXECUTION_RECORD: record,
          UPDATE_REFERENCES: '0',
        },
        maxBuffer: 4 * 1024 * 1024,
      });
    } catch (error) {
      const failure = error as Error & { stdout?: string; stderr?: string };
      throw new Error(
        `Inner browser runner failed for ${selection.shard || 'complete'}:\n${failure.stdout ?? ''}${failure.stderr ?? ''}`,
        { cause: error },
      );
    }
    const identities = (await readFile(record, 'utf8')).trim().split('\n').sort();
    if (JSON.stringify(identities) !== JSON.stringify(selection.identities))
      throw new Error(
        `Inner browser runner ${selection.shard || 'complete'} executed [${identities.join(', ')}]; required [${selection.identities.join(', ')}]`,
      );
  }
  console.log('Inner browser runner coverage passed: complete suite and three disjoint shards.');
} finally {
  await rm(directory, { recursive: true, force: true });
}

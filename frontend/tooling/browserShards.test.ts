import { execFile } from 'node:child_process';
import { cp, mkdtemp, rm, symlink, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { basename, join, resolve } from 'node:path';
import { promisify } from 'node:util';
import { describe, expect, it, onTestFinished } from 'vitest';

const execute = promisify(execFile);

type ListedSuite = {
  title: string;
  suites?: ListedSuite[];
  specs?: { title: string; tests: { projectName: string }[] }[];
};

const listTests = async ({
  frontendRoot,
  shard,
  signal,
}: {
  frontendRoot: string;
  shard?: string;
  signal: AbortSignal;
}): Promise<string[]> => {
  const { stdout } = await execute(
    process.execPath,
    [
      resolve('node_modules/@playwright/test/cli.js'),
      'test',
      '--project=ui',
      '--list',
      '--reporter=json',
      ...(shard ? [`--shard=${shard}`] : []),
    ],
    { cwd: frontendRoot, maxBuffer: 10 * 1024 * 1024, signal },
  );
  const report: { suites: ListedSuite[] } = JSON.parse(stdout);
  const identities: string[] = [];
  const visit = (suite: ListedSuite, parents: string[]) => {
    const titles = [...parents, suite.title];
    for (const spec of suite.specs ?? [])
      for (const test of spec.tests)
        identities.push(JSON.stringify([test.projectName, ...titles, spec.title]));
    for (const child of suite.suites ?? []) visit(child, titles);
  };
  for (const suite of report.suites) visit(suite, []);
  return identities;
};

describe('Browser UI shard coverage', () => {
  it('covers the complete project once and discovers a newly added scenario', async () => {
    const controller = new AbortController();
    onTestFinished(() => controller.abort());
    const directory = await mkdtemp(join(tmpdir(), 'databasus-browser-shards-'));
    try {
      await cp(resolve('browser'), join(directory, 'browser'), {
        recursive: true,
        filter: (path) => basename(path) !== 'references',
      });
      for (const name of ['node_modules', 'src'])
        await symlink(resolve(name), join(directory, name), 'dir');
      for (const name of ['package.json', 'playwright.config.ts', 'tsconfig.node.json'])
        await cp(resolve(name), join(directory, name));
      const baseline = await listTests({ frontendRoot: directory, signal: controller.signal });
      await writeFile(
        join(directory, 'browser/scenarios/discovery.spec.ts'),
        "import { test } from '@playwright/test';\ntest('discovers an added UI scenario', () => {});\n",
      );
      const complete = await listTests({ frontendRoot: directory, signal: controller.signal });
      const shards: string[][] = [];
      for (const index of [1, 2, 3])
        shards.push(
          await listTests({
            frontendRoot: directory,
            shard: `${index}/3`,
            signal: controller.signal,
          }),
        );
      const combined = shards.flat();

      expect(baseline.length).toBeGreaterThan(0);
      expect(complete).toHaveLength(baseline.length + 1);
      expect(new Set(combined).size).toBe(combined.length);
      expect([...combined].sort()).toEqual([...complete].sort());
      expect(shards.every((shard) => shard.length > 0)).toBe(true);
    } finally {
      await rm(directory, { recursive: true, force: true });
    }
  });
});

/* eslint-disable i18next/no-literal-string -- Generates isolated Vitest source files and CLI arguments. */
import { execFile } from 'node:child_process';
import { mkdtemp, rm, writeFile } from 'node:fs/promises';
import { dirname, join, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { promisify, stripVTControlCharacters } from 'node:util';

const frontendRoot = resolve(dirname(fileURLToPath(import.meta.url)), '../../../..');
const runFile = promisify(execFile);

interface IsolatedScenarioOptions {
  scenario: string;
  environment?: 'node' | 'happy-dom';
}

export const runIsolatedScenario = async ({
  scenario,
  environment = 'node',
}: IsolatedScenarioOptions) => {
  const directory = await mkdtemp(join(frontendRoot, 'node_modules/.http-fixture-'));
  try {
    const fixture = join(directory, 'http.test.ts');
    await writeFile(
      fixture,
      `import { it, expect } from 'vitest';
import { http, HttpResponse } from 'msw';
import { setUpMockApi, mockApi } from ${JSON.stringify(join(frontendRoot, 'src/shared/lib/test/mockApi.ts'))};
const server = setUpMockApi();\n${scenario}`,
    );

    // The fixture lives outside src so the regular suite cannot collect intentional failures.
    const config = join(directory, 'vitest.config.mjs');
    await writeFile(
      config,
      `export default { root: ${JSON.stringify(frontendRoot)}, resolve: { alias: { '@': ${JSON.stringify(join(frontendRoot, 'src'))} } }, test: { include: [${JSON.stringify(fixture)}], exclude: [], environment: ${JSON.stringify(environment)}, maxWorkers: 1, fileParallelism: false } };`,
    );

    // The child inherits the parent's color settings; assertions read the plain summary text.
    try {
      const output = await runFile(
        process.execPath,
        [join(frontendRoot, 'node_modules/vitest/vitest.mjs'), 'run', '--config', config],
        { cwd: frontendRoot, timeout: 50_000 },
      );
      return { code: 0, output: stripVTControlCharacters(output.stdout + output.stderr) };
    } catch (error) {
      const failure = error as Error & {
        code?: number;
        stdout?: string;
        stderr?: string;
        killed?: boolean;
      };
      if (failure.killed || typeof failure.code !== 'number') throw error;
      return {
        code: failure.code,
        output: stripVTControlCharacters(`${failure.stdout}${failure.stderr}`),
      };
    }
  } finally {
    await rm(directory, { recursive: true, force: true });
  }
};

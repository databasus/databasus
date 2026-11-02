import { spawnSync } from 'node:child_process';
import { mkdtempSync, mkdirSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { dirname, join, resolve } from 'node:path';
import { afterEach, describe, expect, it } from 'vitest';

import { compareVersions } from '@/shared/lib/version';

import { checkFrontendArchitecture } from '../../../../lint/fsd';

interface BoundaryCase {
  name: string;
  importer: string;
  content: string;
}

interface FixtureFileOptions {
  path: string;
  content: string;
}

const fixtures: string[] = [];
const createProject = (files: Record<string, string>) => {
  const directory = mkdtempSync(join(tmpdir(), 'databasus-fsd-'));
  fixtures.push(directory);

  const write = ({ path, content }: FixtureFileOptions) => {
    mkdirSync(dirname(path), { recursive: true });
    writeFileSync(path, content);
  };

  write({
    path: join(directory, 'tsconfig.json'),
    content: JSON.stringify({ compilerOptions: { baseUrl: '.', paths: { '@/*': ['src/*'] } } }),
  });

  for (const [path, content] of Object.entries(files))
    write({ path: join(directory, 'src', path), content });
  return join(directory, 'src');
};

afterEach(() =>
  fixtures.splice(0).forEach((directory) => rmSync(directory, { recursive: true, force: true })),
);

const runArchitectureLint = (root: string) =>
  spawnSync(process.execPath, [resolve(__dirname, '../../../../lint/fsd.ts'), root], {
    encoding: 'utf8',
  });

const domain = {
  'entities/database/index.ts': 'export const database = 1; export type Database = string;',
  'entities/database/model/connection.ts': 'export const connection = 1;',
};

describe('frontend architecture', () => {
  it('resolves an allowed alias in tests and the checker when the dependency points downward', async () => {
    expect(compareVersions('1.0.0', '1.0.0')).toBe(0);
    const root = createProject({
      ...domain,
      'features/connect/index.ts':
        "import {database} from '@/entities/database'; export {database};",
    });
    const findings = await checkFrontendArchitecture(root);
    expect(
      findings.filter(
        (finding) => finding.rule.includes('import') || finding.rule.includes('dependency'),
      ),
    ).toEqual([]);
  });

  it.each([
    {
      name: 'upward',
      importer: 'shared/api/client.ts',
      content: "import {database} from '@/entities/database';",
    },
    {
      name: 'cross-slice',
      importer: 'features/other/index.ts',
      content: "import {feature} from '@/features/connect';",
    },
    {
      name: 'private',
      importer: 'features/other/index.ts',
      content: "import {connection} from '@/entities/database/model/connection';",
    },
    {
      name: 'type-only',
      importer: 'shared/api/client.ts',
      content: "import type {Database} from '@/entities/database';",
    },
    {
      name: 'dynamic',
      importer: 'shared/api/client.ts',
      content: "const database = import('@/entities/database');",
    },
    {
      name: 're-export',
      importer: 'shared/api/index.ts',
      content: "export {database} from '@/entities/database';",
    },
  ] satisfies BoundaryCase[])(
    'rejects $name dependencies when they violate a boundary',
    async ({ importer, content }) => {
      const root = createProject({
        ...domain,
        'features/connect/index.ts': 'export const feature = 1;',
        'features/connect/model/action.ts': 'export const action=1;',
        'features/other/model/action.ts': 'export const action=1;',
        [importer]: content,
      });
      const findings = await checkFrontendArchitecture(root);
      expect(
        findings.some(
          (finding) =>
            finding.source === importer &&
            (finding.rule.includes('import') || finding.rule.includes('dependency')),
        ),
      ).toBe(true);
    },
  );

  it.each([
    {
      name: 'type-only declaration',
      content: "import type {connection} from '@/entities/database/model/connection';",
    },
    {
      name: 'dynamic import',
      content: "const connection = import('@/entities/database/model/connection');",
    },
    {
      name: 'alias import-type expression',
      content:
        "export type Connection = typeof import('@/entities/database/model/connection').connection;",
    },
    {
      name: 'relative import-type expression',
      content:
        "export type Connection = typeof import('../../entities/database/model/connection').connection;",
    },
  ])('rejects private access with $name from a permitted lower layer', async ({ content }) => {
    const findings = await checkFrontendArchitecture(
      createProject({
        ...domain,
        'features/connect/index.ts': content,
      }),
    );
    expect(findings).toContainEqual({
      rule: 'project/private-slice-import',
      source: 'features/connect/index.ts',
      target: 'entities/database/model/connection.ts',
    });
  });

  it('rejects a cross import when a slice has no segments yet', async () => {
    const root = createProject({
      'features/first/index.ts': "import {second} from '../second';",
      'features/second/index.ts': 'export const second=1;',
    });
    expect(await checkFrontendArchitecture(root)).toContainEqual({
      rule: 'project/forbidden-dependency',
      source: 'features/first/index.ts',
      target: 'features/second/index.ts',
    });
  });

  it('exits with a failure and names the rule and source when one finding exists', () => {
    const root = createProject({
      'shared/ui/index.ts': 'export const control = 1;',
      'constants.ts': 'export const value = 1;',
    });

    const lintRun = runArchitectureLint(root);

    expect(lintRun.status).not.toBe(0);
    expect(lintRun.stderr).toContain('project/file-outside-layers: constants.ts');
  });

  it('exits with a failure and names the rule and source when a forbidden dependency is added', () => {
    const root = createProject({
      ...domain,
      'shared/api/client.ts': "import {database} from '@/entities/database'; export {database};",
    });

    const lintRun = runArchitectureLint(root);

    expect(lintRun.status).not.toBe(0);
    expect(lintRun.stderr).toMatch(
      /(fsd\/forbidden-imports|project\/forbidden-dependency): shared\/api\/client\.ts/,
    );
  });

  it('exits successfully when no finding exists', () => {
    const lintRun = runArchitectureLint(
      createProject({ 'shared/ui/index.ts': 'export const control = 1;' }),
    );

    expect(lintRun.stderr).toBe('');
    expect(lintRun.status).toBe(0);
  });

  it.each([
    { placement: 'directly under the source root', module: 'constants.ts' },
    { placement: 'in a folder beside the layers', module: 'helpers/format.ts' },
  ])('rejects an import-free module $placement', async ({ module }) => {
    const root = createProject({ ...domain, [module]: 'export const value = 1;' });

    expect(await checkFrontendArchitecture(root)).toContainEqual({
      rule: 'project/file-outside-layers',
      source: module,
    });
  });

  it('accepts the entry module, the ambient declaration and the global style sheet', async () => {
    const root = createProject({
      ...domain,
      'main.tsx': 'export {};',
      'vite-env.d.ts': 'export {};',
      'index.css': 'body {}',
    });

    expect(
      (await checkFrontendArchitecture(root)).filter(
        (finding) => finding.rule === 'project/file-outside-layers',
      ),
    ).toEqual([]);
  });

  it('preserves semantic identity when unrelated lines move', async () => {
    const files = {
      ...domain,
      'features/other/index.ts': "import {connection} from '@/entities/database/model/connection';",
    };
    const first = await checkFrontendArchitecture(createProject(files));
    const shifted = await checkFrontendArchitecture(
      createProject({
        ...files,
        'features/other/index.ts': '\n\n' + files['features/other/index.ts'],
      }),
    );
    expect(shifted).toEqual(first);
  });
  it.each(['\n', ' '])(
    'counts an import and re-export of the same target separately with separator %j',
    async (separator) => {
      const imports = "import {database} from '@/entities/database';";
      const reexports = "export {database} from '@/entities/database';";

      const findings = await checkFrontendArchitecture(
        createProject({
          ...domain,
          'shared/api/client.ts': imports + separator + reexports,
        }),
      );

      const violations = findings.filter(
        (finding) =>
          finding.source === 'shared/api/client.ts' && finding.rule.includes('forbidden'),
      );
      expect(violations).toHaveLength(2);
    },
  );
});

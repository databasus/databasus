import { spawnSync } from 'node:child_process';
import { mkdtempSync, mkdirSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { dirname, join, resolve } from 'node:path';
import { afterEach, describe, expect, it } from 'vitest';

import { checkFrontendArchitecture } from '../lint/fsd';

const workspaces: string[] = [];
const importer = 'shared/lib/consumer.ts';
const createProject = (files: Record<string, string>) => {
  const workspace = mkdtempSync(join(tmpdir(), 'databasus-source-alias-'));
  workspaces.push(workspace);
  const root = join(workspace, 'src');
  const projectFiles = {
    'tsconfig.json': JSON.stringify({ compilerOptions: { paths: { '@/*': ['./src/*'] } } }),
    'src/shared/lib/module.ts': 'export const module = 1; export type Module = string;',
    ...files,
  };

  for (const [file, content] of Object.entries(projectFiles)) {
    const path = join(workspace, file);
    mkdirSync(dirname(path), { recursive: true });
    writeFileSync(path, content);
  }

  return root;
};

afterEach(() => {
  for (const workspace of workspaces.splice(0)) rmSync(workspace, { recursive: true, force: true });
});

const runArchitectureLint = (root: string) =>
  spawnSync(process.execPath, [resolve(__dirname, '../lint/fsd.ts'), root], {
    encoding: 'utf8',
    timeout: 50_000,
  });

describe('source alias', () => {
  it.each([
    { form: 'sibling import', content: "import { module } from './module';" },
    { form: 'parent import', content: "import { module } from '../lib/module';" },
    { form: 'type-only import', content: "import type { Module } from './module';" },
    { form: 'inline type import', content: "import { type Module } from './module';" },
    { form: 'named re-export', content: "export { module } from './module';" },
    { form: 'star re-export', content: "export * from './module';" },
    { form: 'type-only re-export', content: "export type { Module } from './module';" },
    { form: 'import-type expression', content: "export type Module = import('./module').Module;" },
    {
      form: 'typeof import expression',
      content: "export type Module = typeof import('./module');",
    },
    { form: 'dynamic string import', content: "void import('./module');" },
    { form: 'dynamic template import', content: 'void import(`./module`);' },
    { form: 'stylesheet import', content: "import './theme.css';" },
    { form: 'missing target', content: "import './missing';" },
    { form: 'directory import', content: "import '.';" },
    { form: 'source-root import', content: "import '../..';" },
  ])('rejects a relative internal reference with $form', async ({ content }) => {
    const root = createProject({ [`src/${importer}`]: content });

    const findings = await checkFrontendArchitecture(root);

    expect(findings.filter((finding) => finding.rule === 'project/source-alias')).toEqual([
      expect.objectContaining({ source: importer }),
    ]);
  });

  it('exits with a diagnostic when a relative internal import is added', () => {
    const root = createProject({ [`src/${importer}`]: "import { module } from './module';" });

    const lintRun = runArchitectureLint(root);

    expect(lintRun.error).toBeUndefined();
    expect(lintRun.signal).toBeNull();
    expect(lintRun.status).toBe(1);
    expect(lintRun.stderr).toMatch(/project\/source-alias: shared\/lib\/consumer\.ts:1:\d+/);
    expect(lintRun.stderr).toContain('./module');
  });

  it.each([
    { form: 'direct import', content: "import { module } from '@/shared/lib/module';" },
    { form: 'type-only import', content: "import type { Module } from '@/shared/lib/module';" },
    { form: 're-export', content: "export { module } from '@/shared/lib/module';" },
    { form: 'star re-export', content: "export * from '@/shared/lib/module';" },
    { form: 'type-only re-export', content: "export type { Module } from '@/shared/lib/module';" },
    {
      form: 'import-type expression',
      content: "export type Module = import('@/shared/lib/module').Module;",
    },
    { form: 'dynamic import', content: "void import('@/shared/lib/module');" },
    { form: 'dynamic template import', content: 'void import(`@/shared/lib/module`);' },
    { form: 'stylesheet import', content: "import '@/shared/lib/theme.css';" },
    { form: 'missing target', content: "import '@/shared/lib/missing';" },
    { form: 'package import', content: "import { useState } from 'react';" },
    { form: 'package subpath import', content: "import dayjsUtc from 'dayjs/plugin/utc';" },
    {
      form: 'scoped package import',
      content: "import { LoadingOutlined } from '@ant-design/icons';",
    },
    { form: 'external tool import', content: "import { tool } from '../../../lint/tool';" },
    { form: 'neighboring directory import', content: "import '../../../src-neighbor/module';" },
    { form: 'module-relative resource', content: "new URL('./font.woff2', import.meta.url);" },
    { form: 'computed import', content: "void import('./module' + '.ts');" },
    { form: 'computed template import', content: 'void import(`./${moduleName}`);' },
    {
      form: 'fixture source string',
      content: 'const fixture = "import module from \'./module\';";',
    },
  ])(
    'accepts a reference with $form when it is outside the relative-import restriction',
    async ({ content }) => {
      const root = createProject({
        [`src/${importer}`]: content,
        'src/shared/lib/theme.css': '@font-face { src: url(./font.woff2); }',
        'lint/tool.ts': 'export const tool = 1;',
        'src-neighbor/module.ts': 'export const module = 1;',
      });

      const findings = await checkFrontendArchitecture(root);

      expect(findings).toEqual([]);
    },
  );

  it('rejects only the internal reference when the same test imports external tooling', async () => {
    const root = createProject({
      'src/shared/lib/consumer.test.ts':
        "import { tool } from '../../../lint/tool';\nimport { module } from './module';",
      'lint/tool.ts': 'export const tool = 1;',
    });

    const findings = await checkFrontendArchitecture(root);

    expect(findings).toEqual([
      {
        rule: 'project/source-alias',
        source: 'shared/lib/consumer.test.ts',
        target: './module',
        line: 2,
        column: 24,
      },
    ]);
  });

  it('ignores relative imports when their source is standalone tooling', async () => {
    const root = createProject({
      'tooling/runner.ts': "import { module } from '../src/shared/lib/module';",
    });

    const findings = await checkFrontendArchitecture(root);

    expect(findings).toEqual([]);
  });

  it('accepts a private target when the importer belongs to the same slice', async () => {
    const root = createProject({
      'src/features/connect/index.ts':
        "export { connect } from '@/features/connect/model/connect';",
      'src/features/connect/model/connect.ts': 'export const connect = 1;',
    });

    const findings = await checkFrontendArchitecture(root);

    expect(findings).toEqual([]);
  });

  it.each([
    {
      direction: 'upward',
      importer: 'shared/lib/consumer.ts',
      relativePath: '../../entities/database',
      aliasPath: '@/entities/database',
      dependencyRule: 'forbidden',
    },
    {
      direction: 'cross-slice',
      importer: 'features/connect/index.ts',
      relativePath: '../database',
      aliasPath: '@/features/database',
      dependencyRule: 'forbidden',
    },
    {
      direction: 'private',
      importer: 'features/connect/index.ts',
      relativePath: '../../entities/database/model/connection',
      aliasPath: '@/entities/database/model/connection',
      dependencyRule: 'private',
    },
  ])(
    'preserves boundary failures when a $direction dependency uses either spelling',
    async ({ importer, relativePath, aliasPath, dependencyRule }) => {
      for (const specifier of [relativePath, aliasPath]) {
        const root = createProject({
          [`src/${importer}`]: `import { database } from '${specifier}';`,
          'src/entities/database/index.ts': 'export const database = 1;',
          'src/entities/database/model/connection.ts': 'export const database = 1;',
          'src/features/database/index.ts': 'export const database = 1;',
        });

        const findings = await checkFrontendArchitecture(root);

        expect(findings.some((finding) => finding.rule.includes(dependencyRule))).toBe(true);
        expect(findings.some((finding) => finding.rule === 'project/source-alias')).toBe(
          specifier === relativePath,
        );
      }
    },
  );

  it('reports each occurrence when an import and re-export share a relative target', async () => {
    const root = createProject({
      [`src/${importer}`]: "import { module } from './module'; export { module } from './module';",
    });

    const findings = await checkFrontendArchitecture(root);

    expect(findings.filter((finding) => finding.rule === 'project/source-alias')).toHaveLength(2);
  });

  it('exits successfully when all internal references use the alias', () => {
    const root = createProject({
      [`src/${importer}`]: "import { module } from '@/shared/lib/module';",
    });

    const lintRun = runArchitectureLint(root);

    expect(lintRun.error).toBeUndefined();
    expect(lintRun.signal).toBeNull();
    expect(lintRun.status).toBe(0);
    expect(lintRun.stderr).toBe('');
    expect(lintRun.stdout).toContain('Architecture checks passed.');
  });
});

import { spawnSync } from 'node:child_process';
import { mkdtempSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join, resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

const frontendRoot = resolve(__dirname, '..');

const lintFixture = (content: string) => {
  const directory = mkdtempSync(join(tmpdir(), 'databasus-lint-rules-'));

  try {
    const fixture = join(directory, 'Fixture.tsx');
    writeFileSync(fixture, content);

    const lintRun = spawnSync(
      process.execPath,
      [
        resolve(frontendRoot, 'node_modules/oxlint/bin/oxlint'),
        '-c',
        'oxlint.config.ts',
        '-f',
        'unix',
        fixture,
      ],
      { cwd: frontendRoot, encoding: 'utf8' },
    );

    if (lintRun.error) throw lintRun.error;
    if (lintRun.signal || lintRun.status === null) {
      throw new Error(`oxlint terminated with ${lintRun.signal}: ${lintRun.stderr}`);
    }

    return lintRun;
  } finally {
    rmSync(directory, { recursive: true, force: true });
  }
};

describe('frontend lint rules', () => {
  it.each([
    {
      finding: 'an omitted effect dependency',
      rule: 'exhaustive-deps',
      content: `import { useEffect } from 'react';
export const Fixture = ({ value }: { value: string }) => {
  useEffect(() => {
    console.log(value);
  }, []);
  return <div />;
};`,
    },
    {
      finding: 'a reference read during rendering',
      rule: 'react(refs)',
      content: `import { useRef } from 'react';
export const Fixture = () => {
  const counter = useRef(0);
  return <div>{counter.current}</div>;
};`,
    },
    {
      finding: 'a reference written during rendering',
      rule: 'react(refs)',
      content: `import { useRef } from 'react';
export const Fixture = ({ value }: { value: string }) => {
  const latestValue = useRef(value);
  latestValue.current = value;
  return <div />;
};`,
    },
    {
      finding: 'an impure call during rendering',
      rule: 'react(purity)',
      content: `export const Fixture = () => <div>{Date.now()}</div>;`,
    },
    {
      finding: 'a mutation during rendering',
      rule: 'react(immutability)',
      content: `export const Fixture = ({ items }: { items: string[] }) => {
  items[0] = 'item';
  return <div />;
};`,
    },
    {
      finding: 'a component file exporting a non-component',
      rule: 'only-export-components',
      content: `export const getValue = () => 1;
export const Fixture = () => <div />;`,
    },
    {
      finding: 'a clickable div',
      rule: 'jsx-a11y(click-events-have-key-events)',
      content: `export const Fixture = () => <div onClick={() => {}} />;`,
    },
    {
      finding: 'an image without alt',
      rule: 'jsx-a11y(alt-text)',
      content: `export const Fixture = () => <img src="/logo.svg" />;`,
    },
  ])('rejects $finding', ({ rule, content }) => {
    const lintRun = lintFixture(content);

    expect(lintRun.status).not.toBe(0);
    expect(lintRun.stdout).toContain(rule);
  });

  it('accepts a single-line disable and still rejects the same finding on the next line', () => {
    const disabledLine = `  // eslint-disable-next-line react/purity -- the fixture states a reason
  const first = Date.now();`;

    const onlyDisabledRun = lintFixture(`export const Fixture = () => {
${disabledLine}
  return <div>{first}</div>;
};`);
    const nextLineRun = lintFixture(`export const Fixture = () => {
${disabledLine}
  const second = Date.now();
  return <div>{first + second}</div>;
};`);

    expect(onlyDisabledRun.status).toBe(0);
    expect(nextLineRun.status).not.toBe(0);
    expect(nextLineRun.stdout).toContain('Fixture.tsx:4:');
    expect(nextLineRun.stdout).not.toContain('Fixture.tsx:3:');
  });

  it('accepts a component that follows the rules', () => {
    expect(lintFixture(`export const Fixture = () => <img src="/logo.svg" alt="" />;`).status).toBe(
      0,
    );
  });
});

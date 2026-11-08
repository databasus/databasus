import { spawnSync } from 'node:child_process';
import { mkdtempSync, mkdirSync, writeFileSync, rmSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join, resolve } from 'node:path';
import ts from 'typescript';
import { describe, expect, it } from 'vitest';

import { checkStyleSource } from '../lint/styleChecks.ts';

const check = ({
  content,
  source = 'shared/ui/Control.tsx',
}: {
  content: string;
  source?: string;
}) => checkStyleSource({ content, source });

describe('Style ownership checks', () => {
  it.each(['#f00', 'rebeccapurple', 'rgb(1 2 3)', 'hsl(30 20% 30%)', 'oklch(50% .2 40)'])(
    'rejects direct colors when component presentation contains %s',
    (color) => {
      expect(check({ content: `<div style={{ color: '${color}' }} />` })).toMatchObject([
        { rule: 'direct-color' },
      ]);
    },
  );
  it('rejects direct colors when CSS declarations define presentation', () => {
    expect(
      check({ content: 'a { color: navy; background: rgb(0 0 0); }', source: 'index.css' }),
    ).toHaveLength(2);
    expect(
      check({
        content: `const command = 'echo #fff'; const value = 'red'; <div title={value} className="p-2 text-content bg-surface" style={{ color: 'var(--app-content)', border: '1px solid currentColor' }} />`,
      }),
    ).toEqual([]);
    expect(
      check({
        content: 'a { background: url("red"); color: var(--app-content); }',
        source: 'index.css',
      }),
    ).toEqual([]);
  });
  it('rejects direct colors when utilities use variants, important modifiers or arbitrary values', () => {
    expect(
      check({
        content:
          '<div className="dark:hover:!text-rose-600 bg-[rgba(0,0,0,.2)] text-[red] bg-white border-[#fff] p-2 text-content" />',
      }),
    ).toHaveLength(5);
    expect(
      check({
        content: '<div className="border-x-red-500 ring-offset-blue-400 text-[#abc]/50" />',
      }),
    ).toHaveLength(3);
    expect(
      check({
        content:
          '<div className="text-[length:var(--text-size)] bg-[var(--app-surface)] min-w-0" />',
      }),
    ).toEqual([]);
  });
  it('rejects direct colors when DOM styles are assigned or set imperatively', () => {
    expect(
      check({
        content:
          "control.style.color = 'red'; control.style['backgroundColor'] = '#fff'; control.style.setProperty('border-color', 'navy');",
      }),
    ).toHaveLength(3);
  });
  it('rejects direct colors when presentation references local constants', () => {
    expect(
      check({
        content: `const danger = 'red'; const classes = 'text-blue-600'; <div className={classes} style={{ color: danger }} />`,
      }),
    ).toHaveLength(2);
  });
  it('counts each use when consumers reference the same presentation object', () => {
    const declaration = "const style = { color: 'red' };";
    expect(check({ content: `${declaration} <div style={style} />` })).toHaveLength(1);
    expect(
      check({ content: `${declaration} <div style={style} /><div style={style} />` }),
    ).toHaveLength(2);
  });
  it('rejects direct colors when presentation references imported constants', () => {
    const root = mkdtempSync(join(tmpdir(), 'databasus-style-'));
    try {
      const palette = join(root, 'palette.ts');
      const control = join(root, 'Control.tsx');
      const content = `import { danger } from './palette'; export const control = <div style={{ color: danger }} />;`;

      writeFileSync(palette, `export const danger = 'red';`);
      writeFileSync(control, content);

      const program = ts.createProgram([palette, control], { jsx: ts.JsxEmit.ReactJSX });
      expect(
        checkStyleSource({ source: 'shared/ui/Control.tsx', fileName: control, content, program }),
      ).toHaveLength(1);
    } finally {
      rmSync(root, { recursive: true, force: true });
    }
  });
  it('permits direct colors when the source is an exact theme or chart owner', () => {
    const content = `<span style={{ color: '#123456' }} />`;
    expect(check({ content, source: 'shared/theme/themeTokens.ts' })).toEqual([]);
    expect(
      check({
        content: content,
        source: 'pages/databases/wal-monitoring/lib/chartColors.ts',
      }),
    ).toEqual([]);
    expect(check({ content: content, source: 'shared/theme/Control.tsx' })).toHaveLength(1);
  });
  it('requires semantic colors when Mantine presentation props and public variables set colors', () => {
    expect(check({ content: '<Button c="red" bg="#fff" color="navy" />' })).toHaveLength(3);
    expect(
      check({
        content: '<Button c="var(--app-content)" bg="var(--app-surface)" color="danger" />',
      }),
    ).toEqual([]);
    expect(
      check({
        content: '<Button vars={() => ({ root: { "--button-bg": "#fff" } })} />',
      }),
    ).toHaveLength(1);
    expect(check({ content: '<Button classNames={{ root: "text-red-500" }} />' })).toHaveLength(1);
    expect(
      check({
        content: '.input { --input-bd: #fff; }',
        source: 'shared/ui/Control.module.css',
      }),
    ).toHaveLength(1);
    expect(
      check({
        content: '.input { --input-bd: var(--app-border); }',
        source: 'shared/ui/Control.module.css',
      }),
    ).toEqual([]);
  });
  it('rejects focus removal when the same control has no visible replacement', () => {
    expect(check({ content: "<button style={{ outline: 'none' }} />" })).toHaveLength(1);
    expect(check({ content: '<button style={{ outline: 0 }} />' })).toHaveLength(1);
    expect(check({ content: '<button className="outline-none" />' })).toMatchObject([
      { rule: 'invisible-focus' },
    ]);
    expect(
      check({
        content:
          '<button className="outline-none focus-visible:outline-2 focus-visible:outline-accent" />',
      }),
    ).toEqual([]);
    expect(check({ content: 'button { outline: none; }', source: 'index.css' })).toHaveLength(1);
    expect(
      check({
        content:
          'button { outline: none; } button:focus-visible { outline: 2px solid var(--app-focus); }',
        source: 'index.css',
      }),
    ).toEqual([]);
  });
  it('permits technical objects without a presentation consumer', () => {
    expect(
      check({
        content: "export const diagnostic = { color: 'red', background: 'navy', fill: '#fff' };",
      }),
    ).toEqual([]);
  });
  it.each([
    'borderTop',
    'borderRight',
    'borderBottom',
    'borderLeft',
    'borderBlock',
    'borderBlockStart',
    'borderBlockEnd',
    'borderInline',
    'borderInlineStart',
    'borderInlineEnd',
  ])('rejects direct colors when the %s shorthand defines a border', (property) => {
    expect(check({ content: `<div style={{ ${property}: '1px solid red' }} />` })).toMatchObject([
      { rule: 'direct-color' },
    ]);
  });
  it.each([
    {
      source: 'index.css',
      content: 'button:focus-visible { outline: none; box-shadow: 0 0 0 0 transparent; }',
    },
    {
      source: 'index.css',
      content: 'button { outline: none; } button:focus-visible { outline: 2px solid transparent; }',
    },
    {
      source: 'index.css',
      content: 'button:focus-visible { outline: none; box-shadow: 0 0 0 0 var(--app-focus); }',
    },
    { content: `<button style={{ outline: 'none', boxShadow: '0 0 0 0 var(--app-focus)' }} />` },
    { content: `<button style={{ outline: 'none', boxShadow: '0 0 0 2px rgba(0,0,0,0)' }} />` },
    {
      content:
        '<button className="outline-none focus-visible:outline-2 focus-visible:outline-transparent" />',
    },
    {
      content:
        '<button className="outline-none focus-visible:ring-2 focus-visible:ring-transparent" />',
    },
    {
      content: '<button className="outline-none focus-visible:outline-[2px_solid_transparent]" />',
    },
    {
      content:
        '<button className="outline-none focus-visible:outline-[0px] focus-visible:outline-accent" />',
    },
    {
      content:
        '<button className="outline-none focus-visible:ring-2 focus-visible:ring-[rgba(0,0,0,0)]" />',
    },
  ])('rejects invisible focus replacements when presentation is $content', (fixture) => {
    expect(check(fixture).some((finding) => finding.rule === 'invisible-focus')).toBe(true);
  });
  it.each([
    { content: '<button style={{ outlineWidth: 0 }} />' },
    { content: '<button style={{ outlineWidth: "0px" }} />' },
    { content: '<button style={{ outlineStyle: "none" }} />' },
    { content: '<button style={{ outlineColor: "transparent" }} />' },
    { source: 'index.css', content: 'button:focus-visible { outline-width: 0; }' },
    { source: 'index.css', content: 'button:focus { outline-width: 0px; }' },
    { source: 'index.css', content: 'button:focus-visible { outline-style: none; }' },
    { source: 'index.css', content: 'button:focus-visible { outline-color: transparent; }' },
    { content: '<button style={{ outline: "2px solid var(--app-focus)", outlineWidth: 0 }} />' },
    {
      source: 'index.css',
      content: 'button:focus-visible { outline: 2px solid var(--app-focus); outline-width: 0; }',
    },
    {
      source: 'index.css',
      content: 'button { outline: none; } button:focus-visible { outline-width: 2px; }',
    },
  ])('rejects focus removal when outline longhands are $content', (fixture) => {
    expect(check(fixture).some((finding) => finding.rule === 'invisible-focus')).toBe(true);
  });
  it.each([
    { content: '<button style={{ outlineWidth: 0, boxShadow: "0 0 0 2px var(--app-focus)" }} />' },
    {
      source: 'index.css',
      content: 'button:focus-visible { outline-width: 0; box-shadow: 0 0 0 2px var(--app-focus); }',
    },
    {
      source: 'index.css',
      content:
        'button { outline-width: 0; } button:focus-visible { outline-width: 2px; outline-style: solid; outline-color: var(--app-focus); }',
    },
    {
      content:
        '<button style={{ outlineWidth: 2, outlineStyle: "solid", outlineColor: "var(--app-focus)" }} />',
    },
  ])('permits focus styling when longhands retain a visible replacement in $content', (fixture) => {
    expect(check(fixture)).toEqual([]);
  });
  it.each([
    {
      content:
        '<button className="outline-none! focus-visible:outline-2 focus-visible:outline-accent" />',
    },
    {
      content:
        '<button className="!outline-none focus-visible:outline-2 focus-visible:outline-solid focus-visible:outline-accent" />',
    },
    {
      source: 'index.css',
      content:
        'button:focus-visible { outline: none; } @media (min-width: 640px) { button:focus-visible { outline: 2px solid var(--app-focus); } }',
    },
    {
      source: 'index.css',
      content:
        'button:focus-visible { outline: none; } @supports (display: grid) { button:focus-visible { outline: 2px solid var(--app-focus); } }',
    },
    { content: "button.style.outline = 'none';" },
    { content: "button.style.setProperty('outline', 'none');" },
    { content: 'button.style.outlineWidth = 0;' },
    { content: "button.style.setProperty('outline-width', '0');" },
  ])('rejects lost focus when the presentation is $content', (fixture) => {
    expect(check(fixture).some((finding) => finding.rule === 'invisible-focus')).toBe(true);
  });
  it.each([
    'button { @apply bg-red-500; }',
    'button { @apply dark:hover:text-rose-600 bg-[rgba(0,0,0,.2)]; }',
    'button { @apply outline-none! focus-visible:outline-2 focus-visible:outline-accent; }',
  ])('rejects stylesheet bypasses when utilities are applied through $content', (content) => {
    expect(check({ source: 'index.css', content }).length).toBeGreaterThan(0);
  });
  it.each([
    {
      content:
        '<button className="outline-none! focus-visible:outline-2 focus-visible:outline-solid! focus-visible:outline-accent" />',
    },
    {
      source: 'index.css',
      content:
        '@media (min-width: 640px) { button:focus-visible { outline: none; box-shadow: 0 0 0 2px var(--app-focus); } }',
    },
    {
      source: 'index.css',
      content:
        '@media (min-width: 640px) { button { outline: none; } button:focus-visible { outline: 2px solid var(--app-focus); } }',
    },
    {
      source: 'index.css',
      content:
        'button { @apply bg-surface text-content outline-none focus-visible:outline-2 focus-visible:outline-accent; }',
    },
    {
      content:
        "button.style.outline = 'none'; button.style.boxShadow = '0 0 0 2px var(--app-focus)';",
    },
    {
      content:
        "button.style.setProperty('outline', 'none'); button.style.setProperty('box-shadow', '0 0 0 2px var(--app-focus)');",
    },
  ])('permits visible focus when scoped presentation is $content', (fixture) => {
    expect(check(fixture)).toEqual([]);
  });
  it('rejects direct colors when presentation uses a shorthand property', () => {
    expect(check({ content: "const color = 'red'; <div style={{ color }} />" })).toMatchObject([
      { rule: 'direct-color' },
    ]);
  });
  it.each([
    "const outline = 'none'; <button style={{ outline }} />",
    'const outline = 0; <button style={{ outline }} />',
    "const base = { outline: '2px solid var(--app-focus)' }; <button style={{ ...base, outline: 'none' }} />",
    "<button style={{ outline: isHidden ? 'none' : '2px solid var(--app-focus)' }} />",
    "const outline = isHidden ? 'none' : '2px solid var(--app-focus)'; <button style={{ outline }} />",
    "<button style={isHidden ? { outline: 'none' } : { outline: '2px solid var(--app-focus)' }} />",
    "if (isNarrow) button.style.outline = 'none'; else button.style.outline = '2px solid var(--app-focus)';",
    "isNarrow ? button.style.outline = 'none' : button.style.outline = '2px solid var(--app-focus)';",
    "button.style.outline = isNarrow ? 'none' : '2px solid var(--app-focus)';",
    '<button className="outline-none focus-visible:outline-2! focus-visible:outline-accent" />',
    '<button className="outline-none focus-visible:!outline-2 focus-visible:outline-accent" />',
  ])('rejects missing focus when a shorthand or conditional presentation is %s', (content) => {
    expect(check({ content }).some((finding) => finding.rule === 'invisible-focus')).toBe(true);
  });
  it.each([
    "const color = 'var(--app-content)'; <div style={{ color }} />",
    "const base = { outline: 'none' }; <button style={{ ...base, outline: '2px solid var(--app-focus)' }} />",
    "const outline = 'none'; const boxShadow = '0 0 0 2px var(--app-focus)'; <button style={{ outline, boxShadow }} />",
    "<button style={{ outline: isHidden ? 'none' : '2px solid var(--app-focus)', boxShadow: '0 0 0 2px var(--app-focus)' }} />",
    "<button style={isNarrow ? { outline: 'none', boxShadow: '0 0 0 2px var(--app-focus)' } : { outline: '2px solid var(--app-focus)' }} />",
    "if (isNarrow) { button.style.outline = 'none'; button.style.boxShadow = '0 0 0 2px var(--app-focus)'; } else button.style.outline = '2px solid var(--app-focus)';",
    '<button className="outline-none focus-visible:outline-2! focus-visible:outline-solid focus-visible:outline-accent" />',
    '<button className="outline-none focus-visible:outline-[2px_solid_var(--app-focus)]!" />',
    '<button className="outline-none! focus-visible:outline-[2px_solid_var(--app-focus)]!" />',
  ])(
    'permits presentation when every conditional branch retains visible focus in %s',
    (content) => {
      expect(check({ content })).toEqual([]);
    },
  );
  it.each([
    "<button className={active ? 'outline-none' : 'focus-visible:ring-2'} />",
    "<button className={`outline-${active ? 'none' : '2 focus-visible:ring-2'}`} />",
    "<button className={active ? 'outline-none' : 'focus-visible:outline-2 focus-visible:outline-accent'} />",
    "const classes = active ? 'outline-none' : 'focus-visible:ring-2'; <button className={classes} />",
    "<button className={`p-2 ${active ? 'outline-none' : 'focus-visible:ring-2'}`} />",
    "<button style={{ outline: 'none', boxShadow: active ? '0 0 0 2px var(--focus)' : undefined }} />",
    "const boxShadow = active ? '0 0 0 2px var(--focus)' : undefined; <button style={{ outline: 'none', boxShadow }} />",
    "<button style={{ outline: 'none', boxShadow: active ? undefined : '0 0 0 2px var(--focus)' }} />",
    "<button style={{ outline: 'none', boxShadow: active ? '0 0 0 2px var(--focus)' : null }} />",
    "<button style={{ outline: 'none', boxShadow: active && '0 0 0 2px var(--focus)' }} />",
    "<button style={{ outline: 'none', ...(active ? { boxShadow: '0 0 0 2px var(--focus)' } : undefined) }} />",
    "button.style.outline = 'none'; button.style.boxShadow = active ? '0 0 0 2px var(--focus)' : undefined;",
    "const base = { boxShadow: '0 0 0 2px var(--focus)' }; <button style={{ ...base, outline: 'none', boxShadow: undefined }} />",
  ])(
    'rejects focus removal when mutually exclusive or absent replacements occur in %s',
    (content) => {
      expect(check({ content }).some((finding) => finding.rule === 'invisible-focus')).toBe(true);
    },
  );
  it.each([
    "<button className={active ? 'outline-none focus-visible:ring-2' : undefined} />",
    "const classes = active ? 'outline-none focus-visible:ring-2' : 'outline-none focus-visible:outline-2 focus-visible:outline-accent'; <button className={classes} />",
    "<button className={`p-2 ${active ? 'outline-none focus-visible:ring-2' : 'outline-none focus-visible:outline-2 focus-visible:outline-accent'}`} />",
    "<button style={{ outline: active ? '2px solid var(--focus)' : undefined }} />",
    "<button style={{ outline: undefined, boxShadow: active ? '0 0 0 2px var(--focus)' : undefined }} />",
    "<button style={{ outline: 'none', boxShadow: active ? '0 0 0 2px var(--focus)' : '0 0 0 3px var(--focus)' }} />",
    "<button style={active ? { outline: 'none', boxShadow: '0 0 0 2px var(--focus)' } : undefined} />",
    "<button className={`${active ? 'outline-none' : ''} focus-visible:ring-2`} />",
    "<button className={active && 'outline-none focus-visible:ring-2'} />",
    "<button className={`outline-${active ? 'none' : '0'} focus-visible:ring-2`} />",
    "const base = { outline: 'none' }; <button style={{ ...base, outline: undefined }} />",
    '<button style={{ outline: void 0 }} />',
  ])('permits focus styling when every rendered branch remains visible in %s', (content) => {
    expect(check({ content })).toEqual([]);
  });
  it.each([
    "<button style={{ outline: 'none', ...(active && { boxShadow: '0 0 0 2px var(--focus)' }) }} />",
    "const replacement = active && { boxShadow: '0 0 0 2px var(--focus)' }; <button style={{ outline: 'none', ...replacement }} />",
    "<button style={{ outline: 'none', ...(active || { boxShadow: '0 0 0 2px var(--focus)' }) }} />",
    "<button style={{ outline: 'none', ...(active ?? { boxShadow: '0 0 0 2px var(--focus)' }) }} />",
    '<div className="divide-[red]" />',
    '<input className="placeholder-[#fff]" />',
  ])(
    'rejects style bypasses when a conditional spread or color utility occurs in %s',
    (content) => {
      expect(check({ content }).length).toBeGreaterThan(0);
    },
  );
  it.each(['divide-[red]', 'placeholder-[#fff]'])(
    'rejects arbitrary colors when a stylesheet applies %s',
    (utility) => {
      expect(check({ source: 'index.css', content: `input { @apply ${utility}; }` })).toMatchObject(
        [{ rule: 'direct-color' }],
      );
    },
  );
  it.each([
    "<button style={{ ...(active && { outline: 'none', boxShadow: '0 0 0 2px var(--focus)' }) }} />",
    '<div className="divide-[var(--app-border)] placeholder-[var(--app-muted)]" />',
  ])('permits semantic styling when every supplied branch is safe in %s', (content) => {
    expect(check({ content })).toEqual([]);
  });
});

describe('frontend style checks', () => {
  const runStyleLint = (content: string) => {
    const directory = mkdtempSync(join(tmpdir(), 'databasus-styles-'));

    try {
      mkdirSync(join(directory, 'shared/ui'), { recursive: true });
      writeFileSync(join(directory, 'shared/ui/Control.tsx'), content);

      return spawnSync(process.execPath, [resolve(__dirname, '../lint/styles.ts'), directory], {
        encoding: 'utf8',
      });
    } finally {
      rmSync(directory, { recursive: true, force: true });
    }
  };

  it('exits with a failure and names the rule and source when one finding exists', () => {
    const lintRun = runStyleLint("export const Control = () => <div style={{ color: '#f00' }} />;");

    expect(lintRun.status).not.toBe(0);
    expect(lintRun.stderr).toContain('direct-color');
    expect(lintRun.stderr).toContain('shared/ui/Control.tsx');
  });

  it('exits successfully when no finding exists', () => {
    expect(runStyleLint('export const Control = () => <div className="flex" />;').status).toBe(0);
  });
});

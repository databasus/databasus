import { readFileSync, readdirSync } from 'node:fs';
import { dirname, relative, resolve } from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';
import ts from 'typescript';

import { compareStyleBaseline, type StyleBaselineEntry } from './styleBaseline.ts';
import { checkStyleSource, type StyleFinding } from './styleChecks.ts';

export const checkFrontendStyles = (root: string): StyleFinding[] => {
  const files = readdirSync(root, { recursive: true, withFileTypes: true })
    .filter(
      (entry) =>
        entry.isFile() &&
        /\.(?:tsx?|css)$/.test(entry.name) &&
        !/\.(?:test\.[^.]+|gen\.ts)$/.test(entry.name),
    )
    .map((entry) => resolve(entry.parentPath, entry.name));
  const program = ts.createProgram(
    files.filter((file) => !file.endsWith('.css')),
    {
      jsx: ts.JsxEmit.ReactJSX,
      moduleResolution: ts.ModuleResolutionKind.Bundler,
      module: ts.ModuleKind.ESNext,
    },
  );
  return files.flatMap((file) =>
    checkStyleSource({
      source: relative(root, file).replaceAll('\\', '/'),
      content: readFileSync(file, 'utf8'),
      fileName: file,
      program: file.endsWith('.css') ? undefined : program,
    }),
  );
};

if (process.argv[1] && pathToFileURL(resolve(process.argv[1])).href === import.meta.url) {
  const frontendRoot = resolve(dirname(fileURLToPath(import.meta.url)), '..');
  const findings = checkFrontendStyles(resolve(frontendRoot, 'src'));
  const baseline = JSON.parse(
    readFileSync(resolve(frontendRoot, 'style-baseline.json'), 'utf8'),
  ) as StyleBaselineEntry[];
  const errors = compareStyleBaseline({ findings, baseline });
  if (errors.length) {
    console.error(errors.join('\n'));
    process.exitCode = 1;
  } else
    console.log(
      `Style checks passed (${findings.length} individually recorded migration findings).`,
    );
}

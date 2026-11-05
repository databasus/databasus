import { readFileSync, readdirSync } from 'node:fs';
import { dirname, relative, resolve } from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';
import ts from 'typescript';

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
  const findings = checkFrontendStyles(resolve(process.argv[2] ?? resolve(frontendRoot, 'src')));

  if (findings.length) {
    for (const { rule, source, construct } of findings) {
      console.error(`${rule}: ${source}: ${construct}`);
    }

    process.exitCode = 1;
  } else {
    console.log('Style checks passed.');
  }
}

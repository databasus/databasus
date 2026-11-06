import { dirname, relative, resolve, sep } from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';
import { linter, processConfiguration } from 'steiger';

import configuration from '../steiger.config.ts';
import type { ArchitectureFinding } from './architectureFinding.ts';
import { checkFilesOutsideLayers } from './checkFilesOutsideLayers.ts';
import { checkProjectDependencies } from './checkProjectDependencies.ts';
import { type FrontendDependency, getFrontendDependencies } from './frontendDependencies.ts';
import { getFrontendModuleReferences } from './frontendModuleReferences.ts';
import { checkSourceAlias } from './sourceAlias.ts';

export const checkFrontendArchitecture = async (root: string): Promise<ArchitectureFinding[]> => {
  processConfiguration(configuration, dirname(root));
  const diagnostics = await linter.run(root);
  const references = getFrontendModuleReferences(root);
  const dependencies = getFrontendDependencies(root, references);
  const coveredForbiddenOccurrences = new Set<FrontendDependency>();

  const primaryFindings = diagnostics
    .filter((diagnostic) => diagnostic.severity === 'error')
    .map((diagnostic) => {
      const startLine = diagnostic.location.start?.line;
      const startColumn = diagnostic.location.start?.column ?? 1;

      const dependency = dependencies
        .filter(
          (item) =>
            item.source === diagnostic.location.path &&
            startLine !== undefined &&
            (item.startLine < startLine ||
              (item.startLine === startLine && item.startColumn <= startColumn)) &&
            (item.endLine > startLine ||
              (item.endLine === startLine && item.endColumn >= startColumn)),
        )
        .sort(
          (left, right) =>
            Math.abs(left.column - (diagnostic.location.start?.column ?? 0)) -
            Math.abs(right.column - (diagnostic.location.start?.column ?? 0)),
        )[0];

      if (dependency && diagnostic.ruleName === 'fsd/forbidden-imports')
        coveredForbiddenOccurrences.add(dependency);

      return {
        rule: diagnostic.ruleName,
        source: relative(root, diagnostic.location.path).split(sep).join('/'),
        target: dependency
          ? relative(root, dependency.target).split(sep).join('/')
          : diagnostic.message.split(root).join('<source>'),
      };
    });

  const missingFindings = checkProjectDependencies(root, dependencies, coveredForbiddenOccurrences);
  return [
    ...primaryFindings,
    ...missingFindings,
    ...checkFilesOutsideLayers(root),
    ...checkSourceAlias(root, references),
  ];
};

if (process.argv[1] && pathToFileURL(resolve(process.argv[1])).href === import.meta.url) {
  const frontendRoot = resolve(dirname(fileURLToPath(import.meta.url)), '..');
  const findings = await checkFrontendArchitecture(
    resolve(process.argv[2] ?? resolve(frontendRoot, 'src')),
  );

  if (findings.length) {
    for (const { rule, source, target, line, column } of findings) {
      const location = line === undefined ? source : `${source}:${line}:${column}`;
      console.error(`${rule}: ${location}${target ? ` -> ${target}` : ''}`);
    }

    process.exitCode = 1;
  } else {
    console.log('Architecture checks passed.');
  }
}

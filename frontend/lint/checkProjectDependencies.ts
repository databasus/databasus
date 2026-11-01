import { relative, sep } from 'node:path';

import type { FrontendDependency } from './frontendDependencies.ts';
import type { ArchitectureFinding } from './fsd-baseline.ts';

const LAYERS = ['app', 'pages', 'widgets', 'features', 'entities', 'shared'];
const NESTED_SLICES = [
  'entities/backups/config',
  'entities/backups/logical',
  'entities/backups/physical',
  'entities/backups/shared',
  'entities/verification/agents',
  'entities/verification/config',
  'entities/verification/runs',
  'features/backups/logical',
  'features/backups/physical',
  'features/verification/agents',
  'features/verification/config',
  'features/verification/runs',
];

const MIGRATION_GROUPS = new Set(
  NESTED_SLICES.map((slice) => slice.split('/').slice(0, 2).join('/')),
);

const getLocation = (path: string) => {
  const parts = path.split('/');
  const layer = parts[0];
  const slice =
    layer === 'app' || layer === 'shared'
      ? null
      : (NESTED_SLICES.find((boundary) => path.startsWith(boundary + '/')) ??
        parts
          .slice(0, MIGRATION_GROUPS.has(parts.slice(0, 2).join('/')) && parts.length > 3 ? 3 : 2)
          .join('/'));
  return { layer, slice };
};

export const checkProjectDependencies = (
  root: string,
  dependencies: FrontendDependency[],
  coveredForbiddenOccurrences: ReadonlySet<FrontendDependency> = new Set(),
): ArchitectureFinding[] => {
  const findings: ArchitectureFinding[] = [];
  for (const dependency of dependencies) {
    const source = relative(root, dependency.source).split(sep).join('/');
    const target = relative(root, dependency.target).split(sep).join('/');
    const importer = getLocation(source);
    const imported = getLocation(target);

    if (!LAYERS.includes(importer.layer) || !LAYERS.includes(imported.layer)) continue;
    const isHigher = LAYERS.indexOf(imported.layer) < LAYERS.indexOf(importer.layer);
    const isCrossSlice =
      importer.layer === imported.layer &&
      importer.slice !== imported.slice &&
      imported.slice !== null;
    if ((isHigher || isCrossSlice) && !coveredForbiddenOccurrences.has(dependency)) {
      findings.push({ rule: 'project/forbidden-dependency', source, target });
    }

    if (
      imported.slice !== null &&
      imported.slice !== importer.slice &&
      target !== imported.slice + '/index.ts' &&
      target !== imported.slice + '/index.tsx'
    ) {
      findings.push({ rule: 'project/private-slice-import', source, target });
    }
  }
  return findings;
};

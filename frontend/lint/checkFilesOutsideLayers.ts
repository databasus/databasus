import { relative, sep } from 'node:path';

import { type ArchitectureFinding, LAYERS } from './checkProjectDependencies.ts';
import { getSourceFiles } from './frontendDependencies.ts';

const ALLOWED_ROOT_MODULES = ['main.tsx', 'vite-env.d.ts'];

// Reads the file list rather than the dependency edges, because a module with no imports and no
// importers has no edge.
export const checkFilesOutsideLayers = (root: string): ArchitectureFinding[] =>
  getSourceFiles(root)
    .map((file) => relative(root, file).split(sep).join('/'))
    .filter((source) => {
      const [topLevelEntry, ...nestedParts] = source.split('/');
      const isInLayer = nestedParts.length > 0 && LAYERS.includes(topLevelEntry);

      return !isInLayer && !ALLOWED_ROOT_MODULES.includes(source);
    })
    .map((source) => ({ rule: 'project/file-outside-layers', source }));

import { dirname, isAbsolute, relative, resolve, sep } from 'node:path';

import type { ArchitectureFinding } from './architectureFinding.ts';
import type { FrontendModuleReference } from './frontendModuleReferences.ts';

export const checkSourceAlias = (
  root: string,
  references: FrontendModuleReference[],
): ArchitectureFinding[] => {
  const isWithinSourceRoot = (path: string) => {
    const sourcePath = relative(root, path);

    return sourcePath !== '..' && !sourcePath.startsWith(`..${sep}`) && !isAbsolute(sourcePath);
  };

  return references
    .filter(
      (reference) =>
        /^\.{1,2}(?:\/|$)/.test(reference.specifier) &&
        isWithinSourceRoot(reference.source) &&
        isWithinSourceRoot(resolve(dirname(reference.source), reference.specifier)),
    )
    .map((reference) => ({
      rule: 'project/source-alias',
      source: relative(root, reference.source).split(sep).join('/'),
      target: reference.specifier,
      line: reference.line,
      column: reference.column,
    }));
};

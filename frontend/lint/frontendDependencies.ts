import { existsSync } from 'node:fs';
import { dirname, join, resolve, sep } from 'node:path';
import ts from 'typescript';

import type { FrontendModuleReference } from './frontendModuleReferences.ts';

export interface FrontendDependency extends FrontendModuleReference {
  target: string;
}

export const getFrontendDependencies = (
  root: string,
  references: FrontendModuleReference[],
): FrontendDependency[] => {
  const configPath = [
    join(dirname(root), 'tsconfig.app.json'),
    join(dirname(root), 'tsconfig.json'),
  ].find(existsSync);

  const options = configPath
    ? ts.parseJsonConfigFileContent(
        ts.readConfigFile(configPath, ts.sys.readFile).config,
        ts.sys,
        dirname(configPath),
      ).options
    : { moduleResolution: ts.ModuleResolutionKind.Bundler, paths: { '@/*': [join(root, '*')] } };
  const dependencies: FrontendDependency[] = [];

  for (const reference of references) {
    const target = ts.resolveModuleName(reference.specifier, reference.source, options, ts.sys)
      .resolvedModule?.resolvedFileName;

    if (target && resolve(target).startsWith(resolve(root) + sep))
      dependencies.push({ ...reference, target });
  }

  return dependencies;
};

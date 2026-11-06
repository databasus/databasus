import { readdirSync } from 'node:fs';
import { join } from 'node:path';

export const getSourceFiles = (directory: string): string[] =>
  readdirSync(directory, { withFileTypes: true }).flatMap((entry) => {
    const path = join(directory, entry.name);
    return entry.isDirectory() ? getSourceFiles(path) : /\.[cm]?[jt]sx?$/.test(path) ? [path] : [];
  });

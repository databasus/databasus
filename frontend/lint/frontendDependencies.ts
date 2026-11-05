import { existsSync, readFileSync, readdirSync } from 'node:fs';
import { dirname, join, resolve, sep } from 'node:path';
import ts from 'typescript';

export interface FrontendDependency {
  source: string;
  target: string;
  line: number;
  column: number;
  startLine: number;
  startColumn: number;
  endLine: number;
  endColumn: number;
}

export const getSourceFiles = (directory: string): string[] =>
  readdirSync(directory, { withFileTypes: true }).flatMap((entry) => {
    const path = join(directory, entry.name);
    return entry.isDirectory() ? getSourceFiles(path) : /\.[cm]?[jt]sx?$/.test(path) ? [path] : [];
  });

export const getFrontendDependencies = (root: string): FrontendDependency[] => {
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

  for (const source of getSourceFiles(root)) {
    const content = ts.createSourceFile(
      source,
      readFileSync(source, 'utf8'),
      ts.ScriptTarget.Latest,
      true,
    );

    const visit = (node: ts.Node) => {
      const moduleSpecifier =
        ts.isImportDeclaration(node) || ts.isExportDeclaration(node)
          ? node.moduleSpecifier
          : ts.isImportTypeNode(node) && ts.isLiteralTypeNode(node.argument)
            ? node.argument.literal
            : ts.isCallExpression(node) && node.expression.kind === ts.SyntaxKind.ImportKeyword
              ? node.arguments[0]
              : undefined;

      if (moduleSpecifier && ts.isStringLiteralLike(moduleSpecifier)) {
        const target = ts.resolveModuleName(moduleSpecifier.text, source, options, ts.sys)
          .resolvedModule?.resolvedFileName;
        if (target && resolve(target).startsWith(resolve(root) + sep)) {
          const { line, character } = content.getLineAndCharacterOfPosition(
            moduleSpecifier.getStart(content),
          );
          const start = content.getLineAndCharacterOfPosition(node.getStart(content));
          const end = content.getLineAndCharacterOfPosition(node.getEnd());

          dependencies.push({
            startLine: start.line + 1,
            startColumn: start.character + 1,
            endLine: end.line + 1,
            endColumn: end.character + 1,
            source,
            target,
            line: line + 1,
            column: character + 1,
          });
        }
      }

      ts.forEachChild(node, visit);
    };

    visit(content);
  }
  return dependencies;
};

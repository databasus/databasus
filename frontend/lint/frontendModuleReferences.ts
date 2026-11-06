import { readFileSync } from 'node:fs';
import ts from 'typescript';

import { getSourceFiles } from './sourceFiles.ts';

export interface FrontendModuleReference {
  source: string;
  specifier: string;
  line: number;
  column: number;
  startLine: number;
  startColumn: number;
  endLine: number;
  endColumn: number;
}

export const getFrontendModuleReferences = (root: string): FrontendModuleReference[] => {
  const references: FrontendModuleReference[] = [];

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
        const { line, character } = content.getLineAndCharacterOfPosition(
          moduleSpecifier.getStart(content),
        );
        const start = content.getLineAndCharacterOfPosition(node.getStart(content));
        const end = content.getLineAndCharacterOfPosition(node.getEnd());

        references.push({
          source,
          specifier: moduleSpecifier.text,
          line: line + 1,
          column: character + 1,
          startLine: start.line + 1,
          startColumn: start.character + 1,
          endLine: end.line + 1,
          endColumn: end.character + 1,
        });
      }

      ts.forEachChild(node, visit);
    };

    visit(content);
  }

  return references;
};

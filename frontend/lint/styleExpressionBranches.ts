import ts from 'typescript';

export const getStyleExpressionBranches = (
  node: ts.Node | undefined,
  checker: ts.TypeChecker,
  visited = new Set<ts.Node>(),
): string[][] => {
  if (!node || visited.has(node)) return [[]];
  visited.add(node);
  if (
    ts.isStringLiteralLike(node) ||
    ts.isJsxText(node) ||
    ts.isNumericLiteral(node) ||
    ts.isTemplateHead(node) ||
    ts.isTemplateMiddle(node) ||
    ts.isTemplateTail(node)
  )
    return [[node.text]];
  if (ts.isVoidExpression(node)) return [[]];
  if (
    ts.isJsxExpression(node) ||
    ts.isParenthesizedExpression(node) ||
    ts.isAsExpression(node) ||
    ts.isTypeAssertionExpression(node) ||
    ts.isNonNullExpression(node) ||
    ts.isSatisfiesExpression(node)
  )
    return getStyleExpressionBranches(node.expression, checker, visited);
  if (ts.isConditionalExpression(node))
    return [
      ...getStyleExpressionBranches(node.whenTrue, checker, new Set(visited)),
      ...getStyleExpressionBranches(node.whenFalse, checker, new Set(visited)),
    ];
  if (ts.isIdentifier(node) || ts.isPropertyAccessExpression(node)) {
    let symbol = ts.isShorthandPropertyAssignment(node.parent)
      ? checker.getShorthandAssignmentValueSymbol(node.parent)
      : checker.getSymbolAtLocation(node);
    if (symbol && symbol.flags & ts.SymbolFlags.Alias) symbol = checker.getAliasedSymbol(symbol);
    const initializers =
      symbol?.declarations?.flatMap((declaration) =>
        (ts.isVariableDeclaration(declaration) || ts.isPropertyAssignment(declaration)) &&
        declaration.initializer
          ? [declaration.initializer]
          : [],
      ) ?? [];
    return initializers.length
      ? initializers.flatMap((initializer) =>
          getStyleExpressionBranches(initializer, checker, new Set(visited)),
        )
      : [[]];
  }
  if (ts.isBinaryExpression(node)) {
    if (node.operatorToken.kind === ts.SyntaxKind.AmpersandAmpersandToken)
      return [...getStyleExpressionBranches(node.right, checker, new Set(visited)), []];
    if (
      [ts.SyntaxKind.BarBarToken, ts.SyntaxKind.QuestionQuestionToken].includes(
        node.operatorToken.kind,
      )
    )
      return [
        ...getStyleExpressionBranches(node.left, checker, new Set(visited)),
        ...getStyleExpressionBranches(node.right, checker, new Set(visited)),
      ];
  }
  let branches: string[][] = ts.isTemplateExpression(node) ? [[node.head.text]] : [[]];
  const children = ts.isTemplateExpression(node)
    ? node.templateSpans.flatMap((span) => [span.expression, span.literal])
    : node.getChildren();
  for (const child of children) {
    const alternatives = getStyleExpressionBranches(child, checker, new Set(visited));
    branches = branches.flatMap((branch) =>
      alternatives.map((alternative) => [...branch, ...alternative]),
    );
  }
  return branches;
};

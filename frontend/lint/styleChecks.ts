import colorNames from 'color-name';
import postcss from 'postcss';
import valueParser from 'postcss-value-parser';
import ts from 'typescript';

import {
  hasVisibleFocusUtility,
  hasVisibleFocusReplacement,
  hasVisibleFocusInEveryBranch,
  isFocusRemoval,
} from './focusVisibility.ts';
import { getStyleExpressionBranches } from './styleExpressionBranches.ts';

export interface StyleFinding {
  rule: 'direct-color' | 'invisible-focus';
  source: string;
  construct: string;
}

const COLOR_OWNERS = new Set([
  'shared/theme/themeTokens.ts',
  'pages/databases/wal-monitoring/lib/chartColors.ts',
]);
const colorProperty =
  /^(?:color|background(?:Color|Image)?|border(?:(?:Top|Right|Bottom|Left|Block|BlockStart|BlockEnd|Inline|InlineStart|InlineEnd)?(?:Color)?)?|fill|stroke|boxShadow|outline(?:Color)?|textShadow|stopColor|caretColor|accentColor)$/;
const paletteUtility =
  /^(?:text|bg|border(?:-[xytrblse])?|ring(?:-offset)?|outline|decoration|divide|placeholder|fill|stroke|shadow|accent|caret|from|via|to)-(?:black|white|(?:slate|gray|zinc|neutral|stone|red|orange|amber|yellow|lime|green|emerald|teal|cyan|sky|blue|indigo|violet|purple|fuchsia|pink|rose)-\d{2,3})(?:\/[^\s]+)?$/;
const colorFunctions = new Set([
  'rgb',
  'rgba',
  'hsl',
  'hsla',
  'hwb',
  'lab',
  'lch',
  'oklab',
  'oklch',
  'color',
]);

const hasDirectColor = (value: string) => {
  let hasColor = false;
  valueParser(value).walk((node) => {
    if (node.type === 'function' && colorFunctions.has(node.value.toLowerCase())) hasColor = true;
    if (node.type === 'function' && ['url', 'env'].includes(node.value)) return false;
    if (
      node.type === 'word' &&
      (/^#[\da-f]{3,8}$/i.test(node.value) || Object.hasOwn(colorNames, node.value.toLowerCase()))
    )
      hasColor = true;
  });
  return hasColor;
};

const getStrings = (
  node: ts.Node | undefined,
  checker: ts.TypeChecker,
  visited = new Set<ts.Node>(),
): string[] => {
  if (!node || visited.has(node)) return [];
  visited.add(node);
  if (ts.isStringLiteralLike(node) || ts.isJsxText(node) || ts.isNumericLiteral(node))
    return [node.text];
  if (ts.isConditionalExpression(node))
    return [
      ...getStrings(node.whenTrue, checker, new Set(visited)),
      ...getStrings(node.whenFalse, checker, new Set(visited)),
    ];
  if (ts.isTemplateExpression(node))
    return [
      node.head.text,
      ...node.templateSpans.flatMap((span) => [
        ...getStrings(span.expression, checker, visited),
        span.literal.text,
      ]),
    ];
  if (ts.isIdentifier(node) || ts.isPropertyAccessExpression(node)) {
    let symbol = ts.isShorthandPropertyAssignment(node.parent)
      ? checker.getShorthandAssignmentValueSymbol(node.parent)
      : checker.getSymbolAtLocation(node);
    if (symbol && symbol.flags & ts.SymbolFlags.Alias) symbol = checker.getAliasedSymbol(symbol);
    return (
      symbol?.declarations?.flatMap((declaration) =>
        ts.isVariableDeclaration(declaration) || ts.isPropertyAssignment(declaration)
          ? getStrings(declaration.initializer, checker, visited)
          : [],
      ) ?? []
    );
  }
  const strings: string[] = [];
  node.forEachChild((child) => {
    strings.push(...getStrings(child, checker, visited));
  });
  return strings;
};

type StyleProperty = ts.PropertyAssignment | ts.ShorthandPropertyAssignment;

const getStyleObjects = (
  node: ts.Node | undefined,
  checker: ts.TypeChecker,
  visited = new Set<ts.Node>(),
): StyleProperty[][] => {
  if (!node || visited.has(node)) return [];
  visited.add(node);
  if (ts.isIdentifier(node) || ts.isPropertyAccessExpression(node)) {
    let symbol = checker.getSymbolAtLocation(node);
    if (symbol && symbol.flags & ts.SymbolFlags.Alias) symbol = checker.getAliasedSymbol(symbol);
    return (
      symbol?.declarations?.flatMap((declaration) =>
        ts.isVariableDeclaration(declaration) || ts.isPropertyAssignment(declaration)
          ? getStyleObjects(declaration.initializer, checker, new Set(visited))
          : [],
      ) ?? []
    );
  }
  if (ts.isObjectLiteralExpression(node)) {
    const properties: StyleProperty[] = [];
    let objects: StyleProperty[][] = [[]];
    for (const property of node.properties) {
      if (ts.isPropertyAssignment(property) || ts.isShorthandPropertyAssignment(property)) {
        properties.push(property);
        objects = objects.map((object) => [...object, property]);
      } else if (ts.isSpreadAssignment(property)) {
        const spreads = getStyleObjects(property.expression, checker, new Set(visited));
        if (spreads.length)
          objects = objects.flatMap((object) => spreads.map((spread) => [...object, ...spread]));
      }
    }
    return [
      ...objects,
      ...properties.flatMap((property) =>
        ts.isPropertyAssignment(property)
          ? getStyleObjects(property.initializer, checker, new Set(visited))
          : [],
      ),
    ];
  }
  if (
    ts.isBinaryExpression(node) &&
    node.operatorToken.kind === ts.SyntaxKind.AmpersandAmpersandToken
  )
    return [...getStyleObjects(node.right, checker, new Set(visited)), []];
  if (
    ts.isConditionalExpression(node) ||
    (ts.isBinaryExpression(node) &&
      [ts.SyntaxKind.BarBarToken, ts.SyntaxKind.QuestionQuestionToken].includes(
        node.operatorToken.kind,
      ))
  ) {
    const alternatives = ts.isConditionalExpression(node)
      ? [node.whenTrue, node.whenFalse]
      : [node.left, node.right];
    return alternatives.flatMap((branch) => {
      const objects = getStyleObjects(branch, checker, new Set(visited));
      return objects.length ? objects : [[]];
    });
  }
  const objects: StyleProperty[][] = [];
  for (const child of node.getChildren())
    objects.push(...getStyleObjects(child, checker, new Set(visited)));
  return objects;
};

const getStylePropertyName = (property: StyleProperty): string =>
  property.name.getText().replace(/['"]/g, '');

const getStyleValue = (property: StyleProperty): ts.Node =>
  ts.isShorthandPropertyAssignment(property) ? property.name : property.initializer;

export const checkStyleSource = ({
  source,
  content,
  program,
  fileName = source,
}: {
  source: string;
  content: string;
  program?: ts.Program;
  fileName?: string;
}): StyleFinding[] => {
  const findings: StyleFinding[] = [];
  const add = (rule: StyleFinding['rule'], construct: string) =>
    findings.push({ rule, source, construct: construct.trim().replace(/\s+/g, ' ') });

  const checkUtilities = ({
    classes,
    construct,
    branches = [classes],
  }: {
    classes: string[];
    construct: string;
    branches?: string[][];
  }) => {
    const utilities = [
      ...classes,
      ...new Set(branches.flat().filter((utility) => !classes.includes(utility))),
    ];
    for (const utility of utilities) {
      const base = utility
        .split(/:(?![^[]*\])/)
        .at(-1)!
        .replace(/^!|!$/g, '');

      const arbitrary =
        base.match(
          /^(?:text|bg|border(?:-[xytrblse])?|ring(?:-offset)?|outline|decoration|divide|placeholder|fill|stroke|shadow|accent|caret|from|via|to)-\[(.+)\](?:\/[\d.]+)?$/,
        )?.[1] ?? base.match(/^\[(?:color|background|border-color):(.+)\]$/)?.[1];

      if (
        paletteUtility.test(base) ||
        (arbitrary && hasDirectColor(arbitrary.replaceAll('_', ' ')))
      )
        add('direct-color', `${construct}:${utility}`);

      if (
        /^(?:outline-none|outline-hidden|ring-0|outline-0)$/.test(base) &&
        !branches.every(
          (branch) =>
            !branch.includes(utility) ||
            hasVisibleFocusUtility({ classes: branch, removedUtility: utility }),
        )
      )
        add('invisible-focus', `${construct}:${utility}`);
    }
  };

  if (source.endsWith('.css')) {
    const sheet = postcss.parse(content, { from: source });
    const getConditions = (node: postcss.Node): string[] => {
      const conditions: string[] = [];
      for (let parent = node.parent; parent; parent = parent.parent)
        if (parent.type === 'atrule') {
          const condition = parent as postcss.AtRule;
          conditions.unshift(`${condition.name}:${condition.params}`);
        }
      return conditions;
    };

    sheet.walkAtRules('apply', (rule) =>
      checkUtilities({ classes: rule.params.split(/\s+/).filter(Boolean), construct: 'apply' }),
    );

    sheet.walkDecls((declaration) => {
      if (hasDirectColor(declaration.value))
        add('direct-color', `${declaration.prop}:${declaration.value}`);
      if (isFocusRemoval({ property: declaration.prop, value: declaration.value })) {
        const selector = declaration.parent?.type === 'rule' ? declaration.parent.selector : '';
        const baseSelector = selector.replace(/:focus(?:-visible)?/g, '');
        const conditions = getConditions(declaration);
        const baseDeclarations: { property: string; value: string; isImportant?: boolean }[] = [];
        const focusDeclarations: { property: string; value: string; isImportant?: boolean }[] = [];

        sheet.walkRules((rule) => {
          const candidateConditions = getConditions(rule);
          if (!candidateConditions.every((condition, index) => conditions[index] === condition))
            return;
          const declarations =
            rule.selector === baseSelector
              ? baseDeclarations
              : [`${baseSelector}:focus`, `${baseSelector}:focus-visible`].includes(rule.selector)
                ? focusDeclarations
                : undefined;

          if (declarations)
            rule.walkDecls((candidate) => {
              if (candidate.parent === rule)
                declarations.push({
                  property: candidate.prop,
                  value: candidate.value,
                  isImportant: candidate.important,
                });
            });
        });

        if (!hasVisibleFocusReplacement([...baseDeclarations, ...focusDeclarations]))
          add('invisible-focus', `${selector}{${declaration.prop}:${declaration.value}}`);
      }
    });

    return findings;
  }

  const compiled =
    program ??
    ts.createProgram(
      [source],
      { jsx: ts.JsxEmit.ReactJSX, allowJs: true, noResolve: true },
      {
        ...ts.createCompilerHost({}),
        getSourceFile: (fileName, languageVersion) =>
          fileName === source
            ? ts.createSourceFile(source, content, languageVersion, true, ts.ScriptKind.TSX)
            : undefined,
      },
    );
  const file = compiled.getSourceFile(fileName);
  if (!file) throw new Error(`Cannot parse ${source}`);
  const checker = compiled.getTypeChecker();
  const checkColor = (property: string, value: ts.Node | undefined) => {
    if (!COLOR_OWNERS.has(source))
      for (const text of getStrings(value, checker))
        if (hasDirectColor(text)) add('direct-color', `${property}:${text}`);
  };
  const mutations = new Map<
    ts.Node,
    {
      target: string;
      scope: ts.Node;
      property: string;
      values: (string | undefined)[];
      isImportant: boolean;
    }
  >();

  const collectMutations = (node: ts.Node) => {
    let target: ts.Node | undefined;
    let property: string | undefined;
    let value: ts.Node | undefined;
    let isImportant = false;

    if (
      ts.isBinaryExpression(node) &&
      node.operatorToken.kind === ts.SyntaxKind.EqualsToken &&
      (ts.isPropertyAccessExpression(node.left) || ts.isElementAccessExpression(node.left)) &&
      ts.isPropertyAccessExpression(node.left.expression) &&
      node.left.expression.name.text === 'style'
    ) {
      target = node.left.expression;
      property = ts.isPropertyAccessExpression(node.left)
        ? node.left.name.text
        : getStrings(node.left.argumentExpression, checker)[0];
      value = node.right;
    } else if (
      ts.isCallExpression(node) &&
      ts.isPropertyAccessExpression(node.expression) &&
      node.expression.name.text === 'setProperty' &&
      ts.isPropertyAccessExpression(node.expression.expression) &&
      node.expression.expression.name.text === 'style'
    ) {
      target = node.expression.expression;
      property = getStrings(node.arguments[0], checker)[0]?.replace(
        /-([a-z])/g,
        (_match, letter: string) => letter.toUpperCase(),
      );
      value = node.arguments[1];
      isImportant = getStrings(node.arguments[2], checker).includes('important');
    }

    if (target && property && value) {
      const values = getStyleExpressionBranches(value, checker).map((branch) =>
        branch.length ? branch.join('') : undefined,
      );

      let scope = node;
      while (
        scope.parent &&
        !ts.isBlock(scope) &&
        !ts.isSourceFile(scope) &&
        !ts.isIfStatement(scope.parent) &&
        !ts.isConditionalExpression(scope.parent) &&
        !ts.isIfStatement(scope) &&
        !ts.isConditionalExpression(scope) &&
        !ts.isCaseClause(scope) &&
        !ts.isIterationStatement(scope, false)
      )
        scope = scope.parent;

      mutations.set(node, { target: target.getText(file), scope, property, values, isImportant });
    }

    ts.forEachChild(node, collectMutations);
  };

  collectMutations(file);
  const visit = (node: ts.Node) => {
    const mutation = mutations.get(node);
    if (mutation) {
      if (colorProperty.test(mutation.property) && !COLOR_OWNERS.has(source))
        for (const value of mutation.values)
          if (value !== undefined && hasDirectColor(value))
            add('direct-color', `style.${mutation.property}:${value}`);

      const property = mutation.property.replace(/[A-Z]/g, (letter) => `-${letter.toLowerCase()}`);
      if (
        mutation.values.some((value) => value !== undefined && isFocusRemoval({ property, value }))
      ) {
        const assignments = [...mutations.values()]
          .filter(
            (candidate) =>
              candidate.target === mutation.target && candidate.scope === mutation.scope,
          )
          .map((candidate) => ({
            property: candidate.property.replace(/[A-Z]/g, (letter) => `-${letter.toLowerCase()}`),
            values: candidate.values,
            isImportant: candidate.isImportant,
          }));
        if (!hasVisibleFocusInEveryBranch(assignments))
          add('invisible-focus', `style.${mutation.property}:${mutation.values.join(' ')}`);
      }
    }

    if (ts.isJsxAttribute(node)) {
      const attribute = node.name.getText(file);

      if (
        (attribute === 'style' ||
          attribute === 'styles' ||
          attribute === 'vars' ||
          /Style$/.test(attribute)) &&
        node.initializer &&
        ts.isJsxExpression(node.initializer) &&
        node.initializer.expression
      ) {
        const objects = getStyleObjects(node.initializer.expression, checker);
        const properties = new Set(objects.flat());

        for (const property of properties) {
          const name = getStylePropertyName(property);
          if (colorProperty.test(name) || name.startsWith('--'))
            checkColor(
              `${ts.isObjectLiteralExpression(node.initializer.expression) ? '' : 'style.'}${name}`,
              getStyleValue(property),
            );
        }

        const failedProperties = new Map<StyleProperty, string[]>();
        for (const object of objects) {
          const properties = object.filter(
            (property, index) =>
              !object
                .slice(index + 1)
                .some(
                  (laterProperty) =>
                    getStylePropertyName(laterProperty) === getStylePropertyName(property),
                ),
          );
          const assignments = properties.map((property) => ({
            property: getStylePropertyName(property).replace(
              /[A-Z]/g,
              (letter) => `-${letter.toLowerCase()}`,
            ),
            values: getStyleExpressionBranches(getStyleValue(property), checker).map((branch) =>
              branch.length ? branch.join('') : undefined,
            ),
          }));
          if (hasVisibleFocusInEveryBranch(assignments)) continue;

          properties.forEach((property, index) => {
            const assignment = assignments[index];
            const removals = assignment.values.filter(
              (value): value is string =>
                value !== undefined && isFocusRemoval({ property: assignment.property, value }),
            );
            if (removals.length) failedProperties.set(property, removals);
          });
        }

        for (const [property, removals] of failedProperties)
          add('invisible-focus', `${getStylePropertyName(property)}:${removals.join(' ')}`);
      }

      if (colorProperty.test(attribute) || attribute === 'c' || attribute === 'bg')
        checkColor(attribute, node.initializer);
      if (attribute === 'className' || attribute === 'classNames') {
        const classes = getStrings(node.initializer, checker)
          .flatMap((value) => value.split(/\s+/))
          .filter(Boolean);
        const branches = getStyleExpressionBranches(node.initializer, checker).map((branch) =>
          branch.join('').split(/\s+/).filter(Boolean),
        );
        checkUtilities({ classes, construct: 'class', branches });
      }
    }

    ts.forEachChild(node, visit);
  };
  visit(file);
  return findings;
};

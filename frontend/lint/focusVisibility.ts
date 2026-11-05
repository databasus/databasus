import valueParser from 'postcss-value-parser';

const hasTransparentPaint = (value: string): boolean => {
  let isTransparent = /\btransparent\b|#[\da-f]{6}00\b|#[\da-f]{3}0\b/i.test(value);
  valueParser(value).walk((node) => {
    if (
      node.type !== 'function' ||
      !/^(?:rgba?|hsla?|hwb|lab|lch|oklab|oklch|color)$/.test(node.value)
    )
      return;
    const content = valueParser.stringify(node.nodes);
    const alpha = content.includes('/')
      ? content.split('/').at(-1)
      : /^(rgba|hsla)$/.test(node.value)
        ? content.split(',')[3]
        : undefined;
    if (alpha !== undefined && parseFloat(alpha) === 0) isTransparent = true;
  });
  return isTransparent;
};

export const isVisibleFocusStyle = ({
  property,
  value,
}: {
  property: string;
  value: string;
}): boolean => {
  if (/\b(?:none|hidden)\b/.test(value) || hasTransparentPaint(value)) return false;
  const dimensions = valueParser(value)
    .nodes.filter((node) => node.type === 'word' && /^-?[\d.]+(?:px|rem|em)?$/.test(node.value))
    .map((node) => parseFloat(node.value));
  if (property === 'outline')
    return (
      dimensions.some((dimension) => dimension > 0) &&
      /\b(?:solid|dashed|dotted|double|groove|ridge|inset|outset)\b/.test(value)
    );
  return (
    dimensions.length >= 2 &&
    (dimensions.slice(0, 2).some((dimension) => dimension !== 0) ||
      dimensions.slice(2).some((dimension) => dimension > 0))
  );
};

interface FocusDeclaration {
  property: string;
  value: string;
  isImportant?: boolean;
}

export const isFocusRemoval = ({ property, value }: FocusDeclaration): boolean => {
  if (property === 'outline-width') return /^0(?:[a-z%]+)?$/i.test(value.trim());
  if (property === 'outline-style') return /^(?:none|hidden|initial|unset)$/.test(value.trim());
  if (property === 'outline-color') return hasTransparentPaint(value);
  return ['outline', 'box-shadow'].includes(property) && !isVisibleFocusStyle({ property, value });
};

export const hasVisibleFocusReplacement = (declarations: FocusDeclaration[]): boolean => {
  let width = 'medium';
  let style = 'none';
  let color = 'currentColor';
  let shadow = 'none';

  for (const declaration of declarations.toSorted(
    (left, right) => Number(!!left.isImportant) - Number(!!right.isImportant),
  )) {
    if (declaration.property === 'outline') {
      const words = valueParser(declaration.value)
        .nodes.filter((node) => node.type === 'word')
        .map((node) => node.value);
      width =
        words.find((word) => /^(?:[\d.]+(?:px|em|rem)?|thin|medium|thick)$/.test(word)) ?? 'medium';
      style =
        words.find((word) =>
          /^(?:none|hidden|auto|solid|dashed|dotted|double|groove|ridge|inset|outset)$/.test(word),
        ) ?? 'none';
      color = hasTransparentPaint(declaration.value) ? 'transparent' : 'currentColor';
    } else if (declaration.property === 'outline-width') width = declaration.value;
    else if (declaration.property === 'outline-style') style = declaration.value;
    else if (declaration.property === 'outline-color') color = declaration.value;
    else if (declaration.property === 'box-shadow') shadow = declaration.value;
  }

  const hasOutlineWidth = /^(?:thin|medium|thick)$/.test(width) || parseFloat(width) > 0;
  const hasOutlineStyle = /^(?:auto|solid|dashed|dotted|double|groove|ridge|inset|outset)$/.test(
    style,
  );

  return (
    (hasOutlineWidth && hasOutlineStyle && !hasTransparentPaint(color)) ||
    isVisibleFocusStyle({ property: 'box-shadow', value: shadow })
  );
};

export const hasVisibleFocusInEveryBranch = (
  assignments: { property: string; values: (string | undefined)[]; isImportant?: boolean }[],
): boolean => {
  let branches: FocusDeclaration[][] = [[]];
  for (const { property, values, isImportant } of assignments) {
    if (!/^(?:outline(?:-width|-style|-color)?|box-shadow)$/.test(property) || !values.length)
      continue;
    branches = branches.flatMap((branch) =>
      values.map((value) =>
        value === undefined ? branch : [...branch, { property, value, isImportant }],
      ),
    );
  }
  return branches.every(hasVisibleFocusReplacement);
};

const getUtility = (value: string): string =>
  value
    .split(/:(?![^[]*\])/)
    .at(-1)!
    .replace(/^!|!$/g, '');

const isImportantUtility = (utility: string): boolean =>
  /^(?:!.*|.*!)$/.test(utility.split(/:(?![^[]*\])/).at(-1)!);

const hasOutlineStyle = (utility: string): boolean => {
  const base = getUtility(utility);
  if (/^outline-(?:solid|dashed|dotted|double)$/.test(base)) return true;
  const arbitrary = base.match(/^outline-\[(.+)\]$/)?.[1];
  return (
    !!arbitrary &&
    isVisibleFocusStyle({ property: 'outline', value: arbitrary.replaceAll('_', ' ') })
  );
};

export const hasVisibleFocusUtility = ({
  classes,
  removedUtility,
}: {
  classes: string[];
  removedUtility: string;
}): boolean => {
  const removedConditions = removedUtility
    .split(/:(?![^[]*\])/)
    .slice(0, -1)
    .filter((condition) => !['focus', 'focus-visible'].includes(condition));

  const applicable = classes.filter((utility) =>
    utility
      .split(/:(?![^[]*\])/)
      .slice(0, -1)
      .filter((condition) => !['focus', 'focus-visible'].includes(condition))
      .every((condition) => removedConditions.includes(condition)),
  );

  const importantRemovals = applicable.filter(
    (utility) =>
      isImportantUtility(utility) && /^outline-(?:none|hidden)$/.test(getUtility(utility)),
  );

  for (const state of ['focus:', 'focus-visible:']) {
    const stateClasses = applicable.filter((utility) => utility.includes(state));
    const utilities = stateClasses.map(getUtility);

    const hasImportantOutlineWidth = stateClasses.some(
      (utility) =>
        isImportantUtility(utility) &&
        /^outline-(?:\d+|\[(?:\d+(?:\.\d+)?)(?:px|rem|em)\])$/.test(getUtility(utility)),
    );
    const hasExplicitOutlineStyle = stateClasses.some(hasOutlineStyle);
    const hasEffectiveOutlineStyle =
      (!hasImportantOutlineWidth ||
        !applicable.some((utility) => /^outline-(?:none|hidden)$/.test(getUtility(utility))) ||
        hasExplicitOutlineStyle) &&
      importantRemovals.every(
        (removal) =>
          !removal.includes(state) &&
          stateClasses.some((utility) => isImportantUtility(utility) && hasOutlineStyle(utility)),
      );

    for (const kind of ['outline', 'ring']) {
      if (kind === 'outline' && !hasEffectiveOutlineStyle) continue;

      const related = utilities.filter(
        (utility) => utility.startsWith(`${kind}-`) || utility === kind,
      );
      if (
        related.some(
          (utility) =>
            hasTransparentPaint((utility.match(/\[(.+)\]/)?.[1] ?? utility).replaceAll('_', ' ')) ||
            new RegExp(`^${kind}-(?:none|hidden|0)$`).test(utility) ||
            /\/0(?:\D|$)/.test(utility),
        )
      )
        continue;

      if (
        related.some(
          (utility) =>
            new RegExp(`^${kind}-(?:[1-9]\\d*|\\[(?:[1-9][\\d.]*|0\\.\\d+)(?:px|rem|em)\\])$`).test(
              utility,
            ) ||
            (kind === 'ring' && utility === 'ring'),
        )
      )
        return true;

      if (
        related.some((utility) => {
          const arbitrary = utility.match(/^outline-\[(.+)\]$/)?.[1];
          return (
            arbitrary &&
            isVisibleFocusStyle({ property: 'outline', value: arbitrary.replaceAll('_', ' ') })
          );
        })
      )
        return true;
    }
  }
  return false;
};

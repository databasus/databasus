import i18next from 'eslint-plugin-i18next';
import i18nextDefaults from 'eslint-plugin-i18next/lib/options/defaults.js';

// Options for i18next/no-literal-string. `all` mode checks every string literal and template, not
// only JSX text, because copy also lives in attributes, option arrays, message.* calls and .ts
// files. Structural strings are excluded by recurring class; a one-off technical string takes an
// inline suppression with a reason instead of a new entry here.
const i18nextLiteralStringOptions = {
  mode: 'all',
  'should-validate-template': true,
  'jsx-components': {
    exclude: ['Trans'],
  },
  // Props that never carry copy. An exclude-list rather than an include-list: a copy prop missing
  // from an include-list would go unchecked unnoticed, while a non-copy prop missing here shows up
  // as a lint error someone fixes.
  'jsx-attributes': {
    exclude: [
      ...i18nextDefaults['jsx-attributes'].exclude,
      '\\w*ClassName',
      'data-[\\w-]+',
      'accept',
      'align',
      'autoComplete',
      'color',
      'controlId',
      'htmlFor',
      'regionId',
      'aria-controls',
      'aria-describedby',
      'dataIndex',
      'direction',
      'format',
      'href',
      'htmlType',
      'i18nKey',
      'inputMode',
      'justify',
      'labelAlign',
      'lang',
      'layout',
      'mode',
      'name',
      'optionFilterProp',
      'path',
      'picker',
      'placement',
      'rel',
      'role',
      'rowKey',
      'shape',
      'size',
      'src',
      'status',
      'target',
      'theme',
      'trigger',
      'value',
      'variant',
    ],
  },
  words: {
    exclude: [
      // digits, punctuation and whitespace only, e.g. "0 2 * * *" or "********"
      '[\\s0-9!-/:-@[-`{-~]+',
      ...i18nextDefaults.words.exclude.filter((pattern) => pattern !== '[0-9!-/:-@[-`{-~]+'),
      // URLs and connection strings
      '(https?|wss?|s?ftp|s3|mongodb(\\+srv)?|postgres(ql)?|mysql|mariadb|jdbc:\\w+)://\\S*',
      // absolute and relative paths
      '\\.{0,2}/\\S*',
      // CSS values
      '#[0-9a-fA-F]{3,8}',
      '-?\\d+(\\.\\d+)?(px|rem|em|%|vh|vw|ms|s|fr)',
      // e-mail addresses used as examples
      '[\\w.+-]+@[\\w-]+(\\.[\\w-]+)+',
      // BCP 47 language tags such as en-US
      '[a-z]{2}-[A-Z]{2}',
      // Without type information a literal typed as a union of string literals is no longer exempt,
      // so the two shapes those unions take are excluded by form: translation keys such as
      // 'workspaces.roles.viewer', and single lower-case tokens such as 'signIn', 'db-settings' or
      // 'dark'. Copy starts with a capital letter or holds a space, so it is still caught.
      '[a-z][a-zA-Z0-9]*(\\.[a-zA-Z0-9_]+)+',
      '[a-z0-9][a-zA-Z0-9]*([-_][a-zA-Z0-9]+)*',
    ],
  },
  callees: {
    exclude: [
      ...i18nextDefaults.callees.exclude,
      'console\\.\\w+',
      'addHeader',
      'setMethod',
      'setCredentials',
      'document\\.\\w+',
      '(local|session)Storage\\.\\w+',
      'URLSearchParams',
      'searchParams\\.\\w+',
      // Map, Set and URLSearchParams lookups
      'get(All)?',
      'has',
      'dayjs(\\.\\w+)?',
      'format',
      'split',
      'join',
      'replace(All)?',
      'match(All)?',
      'test',
      'matchMedia',
      'querySelector(All)?',
      '(set|get|remove)Attribute',
      'classList\\.\\w+',
      // Error text never reaches the user: translateApiError shows only an ApiError's backend
      // message and a translated message otherwise.
      'Error',
      'TypeError',
    ],
  },
  'object-properties': {
    exclude: [
      ...i18nextDefaults['object-properties'].exclude,
      'align',
      'anchor',
      'path',
      // CSS properties set on DOM elements
      'border\\w*',
      'boxShadow',
      'cursor',
      'display',
      'fontWeight',
      'overflow\\w*',
      'position',
      'textAlign',
      'transition',
      'whiteSpace',
      'cache',
      'className',
      'color',
      'credentials',
      'dataIndex',
      'defaultSortOrder',
      'fixed',
      'format',
      'headers',
      'icon',
      'key',
      'method',
      'mode',
      'okType',
      'placement',
      'size',
      'type',
      'value',
      'width',
    ],
  },
};

// The plugin's default patterns include RegExp objects, which do not survive oxlint's JSON
// configuration, so the options are injected here rather than set in the linter config.
const withOptions = (context) =>
  new Proxy(context, {
    get(target, property) {
      return property === 'options'
        ? [i18nextLiteralStringOptions]
        : Reflect.get(target, property, target);
    },
  });

// eslint-plugin-i18next exempts every literal inside a SCREAMING_SNAKE variable and every default
// parameter value, and no option turns that off. Module-level label tables are exactly where copy
// hides, so both node types are routed through the handler the plugin uses for an ordinary
// lower-case variable, which checks its literals.
const pluginNoLiteralString = i18next.rules['no-literal-string'];
const i18nextPlugin = {
  ...i18next,
  rules: {
    ...i18next.rules,
    'no-literal-string': {
      ...pluginNoLiteralString,
      create(context) {
        const visitor = pluginNoLiteralString.create(withOptions(context));
        const checkLikeLowerCaseVariable = () =>
          visitor.VariableDeclarator({ id: { name: 'checked' } });

        return {
          ...visitor,
          VariableDeclarator: checkLikeLowerCaseVariable,
          AssignmentPattern: checkLikeLowerCaseVariable,
        };
      },
    },
  },
};

export default i18nextPlugin;

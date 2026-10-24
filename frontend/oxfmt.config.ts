import { defineConfig } from 'oxfmt';

export default defineConfig({
  semi: true,
  singleQuote: true,
  trailingComma: 'all',
  printWidth: 100,
  tabWidth: 2,
  arrowParens: 'always',
  sortImports: {
    groups: [['builtin', 'external'], 'internal', ['parent', 'sibling', 'index'], 'unknown'],
  },
  sortTailwindcss: {
    stylesheet: './src/index.css',
  },
  ignorePatterns: ['dist', 'pnpm-lock.yaml', 'src/shared/api/schema.gen.ts'],
});

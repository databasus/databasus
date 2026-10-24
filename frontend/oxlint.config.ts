import { defineConfig } from 'oxlint';

export default defineConfig({
  plugins: ['react', 'typescript'],
  jsPlugins: [{ name: 'i18next', specifier: './lint/i18next.js' }],
  ignorePatterns: ['dist', 'src/shared/api/schema.gen.ts'],
  rules: {
    'react/rules-of-hooks': 'error',
    'react/only-export-components': ['warn', { allowConstantExport: true }],
    'react-hooks/exhaustive-deps': 'off',
    'react/immutability': 'off',
    'react/purity': 'off',
    'react/refs': 'off',
    'react/set-state-in-effect': 'off',
  },
  overrides: [
    {
      files: ['src/**/*.{ts,tsx}'],
      rules: { 'i18next/no-literal-string': 'error' },
    },
    {
      files: ['**/*.test.{ts,tsx}', '**/api/**', 'src/shared/i18n/locales/**'],
      rules: { 'i18next/no-literal-string': 'off' },
    },
  ],
});

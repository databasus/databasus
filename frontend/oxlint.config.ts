import { defineConfig } from 'oxlint';

export default defineConfig({
  plugins: ['react', 'typescript', 'jsx-a11y'],
  jsPlugins: [{ name: 'i18next', specifier: './lint/i18next.js' }],
  ignorePatterns: ['dist', 'src/shared/api/schema.gen.ts'],
  rules: {
    'react/rules-of-hooks': 'error',
    'react/only-export-components': ['error', { allowConstantExport: true }],
    'react-hooks/exhaustive-deps': 'error',
    'react/immutability': 'error',
    'react/purity': 'error',
    'react/refs': 'error',
    // Most findings are a pending flag or a reset at the start of a load. Removing them means
    // rewriting how nearly every screen loads, which risks its tested late-answer guarantees for no
    // visible gain.
    'react/set-state-in-effect': 'off',
    'jsx-a11y/click-events-have-key-events': 'error',
    'jsx-a11y/no-static-element-interactions': 'error',
    'jsx-a11y/alt-text': 'error',
    'jsx-a11y/anchor-is-valid': 'error',
    // Cannot see translated children: every finding is a link or button named by a t() call.
    'jsx-a11y/control-has-associated-label': 'off',
    'jsx-a11y/anchor-has-content': 'off',
    // The findings are `role` on div elements where the native tag would bring unwanted layout.
    'jsx-a11y/prefer-tag-over-role': 'off',
    // Tooltip triggers take focus so a keyboard user can reach what the tooltip says.
    'jsx-a11y/no-noninteractive-tabindex': 'off',
    // Dialogs with a single field focus it on purpose when the user opens them.
    'jsx-a11y/no-autofocus': 'off',
  },
  overrides: [
    {
      files: ['src/**/*.{ts,tsx}'],
      rules: { 'i18next/no-literal-string': 'error' },
    },
    {
      files: [
        '**/*.test.{ts,tsx}',
        '**/*.fixture.{ts,tsx}',
        '**/api/**',
        'src/shared/i18n/locales/**',
      ],
      rules: { 'i18next/no-literal-string': 'off' },
    },
  ],
});

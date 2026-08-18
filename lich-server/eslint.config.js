import neostandard from 'neostandard';
import prettierConfig from 'eslint-config-prettier';

export default [
  ...neostandard({
    ts: true,
    semi: true,
  }),
  {
    ignores: ['dist/**', 'data/**', 'coverage/**', 'node_modules/**', '*.config.js'],
  },
  {
    rules: {
      camelcase: 'off',
      '@typescript-eslint/no-explicit-any': 'off',
      '@typescript-eslint/no-unused-vars': ['warn', { argsIgnorePattern: '^_' }],
    },
  },
  prettierConfig,
];

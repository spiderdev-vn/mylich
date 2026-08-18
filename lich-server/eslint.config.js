import neostandard from 'neostandard';

export default [
  ...neostandard({
    ts: true,
    semi: true,
  }),
  {
    ignores: [
      'dist/**',
      'data/**',
      'coverage/**',
      'node_modules/**',
      '*.config.js',
    ],
  },
  {
    rules: {
      'camelcase': 'off',
      '@typescript-eslint/no-explicit-any': 'off',
      '@typescript-eslint/no-unused-vars': ['warn', { argsIgnorePattern: '^_' }],
      '@stylistic/semi': ['error', 'always'],
      '@stylistic/space-before-function-paren': ['error', {
        anonymous: 'always',
        named: 'never',
        asyncArrow: 'always',
      }],
    },
  },
];

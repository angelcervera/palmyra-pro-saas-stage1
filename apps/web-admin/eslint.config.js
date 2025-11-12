import js from '@eslint/js'
import globals from 'globals'
import reactHooks from 'eslint-plugin-react-hooks'
import reactRefresh from 'eslint-plugin-react-refresh'
import tseslint from 'typescript-eslint'
import { defineConfig, globalIgnores } from 'eslint/config'

const tsRules = Object.assign({}, ...tseslint.configs.recommended.map((cfg) => cfg.rules ?? {}))
const reactHooksRules = reactHooks.configs['recommended-latest']?.rules ?? {}
const reactRefreshRules = reactRefresh.configs.vite?.rules ?? {}

export default defineConfig([
  globalIgnores(['dist']),
  js.configs.recommended,
  {
    files: ['**/*.{ts,tsx}'],
    languageOptions: {
      parser: tseslint.parser,
      parserOptions: {
        project: ['./tsconfig.node.json', './tsconfig.app.json'],
        tsconfigRootDir: import.meta.dirname,
      },
      ecmaVersion: 2020,
      sourceType: 'module',
      globals: globals.browser,
    },
    plugins: {
      '@typescript-eslint': tseslint.plugin,
      'react-hooks': reactHooks,
      'react-refresh': reactRefresh,
    },
    rules: {
      ...tsRules,
      ...reactHooksRules,
      ...reactRefreshRules,
      '@typescript-eslint/no-explicit-any': 'off',
      'react-refresh/only-export-components': 'off',
      'react-hooks/incompatible-library': 'off',
      'react-hooks/purity': 'off',
    },
  },
])

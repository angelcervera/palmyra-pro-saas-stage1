import { defineConfig, globalIgnores } from "eslint/config";
import reactHooks from "eslint-plugin-react-hooks";

export default defineConfig([
	globalIgnores(["dist"]),
	{
		files: ["**/*.tsx"],
		languageOptions: {
			ecmaVersion: 2020,
			sourceType: "module",
			parserOptions: {
				ecmaFeatures: { jsx: true },
			},
		},
		plugins: {
			"react-hooks": reactHooks,
		},
		rules: {
			"react-hooks/rules-of-hooks": "error",
			"react-hooks/exhaustive-deps": "warn",
		},
	},
]);

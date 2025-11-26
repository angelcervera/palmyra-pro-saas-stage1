import { fileURLToPath } from "node:url";
import { defineConfig } from "vite";

export default defineConfig({
	plugins: [],
	worker: {
		format: "es",
	},
	assetsInclude: ["**/*.wasm"],
});

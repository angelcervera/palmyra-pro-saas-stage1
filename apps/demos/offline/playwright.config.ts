import { defineConfig } from "@playwright/test";

export default defineConfig({
	testDir: "./tests",
	timeout: 60_000,
	expect: {
		timeout: 5_000,
	},
	use: {
		baseURL: "http://localhost:4173",
		headless: true,
	},
	webServer: {
		command: "pnpm dev --host --port 4173",
		port: 4173,
		reuseExistingServer: !process.env.CI,
		timeout: 120_000,
	},
});

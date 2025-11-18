import "@sqlite.org/sqlite-wasm";

// The upstream package exports sqlite3Worker1Promiser at runtime but omits it
// from its .d.ts. We augment the module here so TypeScript understands the
// named export when compiling our offline provider.
declare module "@sqlite.org/sqlite-wasm" {
	type WorkerRequestHandler = (
		type: string,
		args?: Record<string, unknown>,
	) => Promise<unknown>;

	type WorkerConfig = Record<string, unknown> | undefined;

	export interface Sqlite3Worker1Promiser {
		(config?: WorkerConfig): WorkerRequestHandler;
		defaultConfig?: Record<string, unknown>;
		v2(config?: WorkerConfig): Promise<WorkerRequestHandler>;
	}

	export const sqlite3Worker1Promiser: Sqlite3Worker1Promiser;
}

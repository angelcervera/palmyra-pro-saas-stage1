import { readFile } from "node:fs/promises";
import { fileURLToPath } from "node:url";

type WorkerMessage = Record<string, unknown>;

export type NodeWorkerPromiser = (
	type: string,
	args?: WorkerMessage,
) => Promise<WorkerMessage>;

interface ExecArgs extends WorkerMessage {
	sql: string;
	bind?: ReadonlyArray<unknown> | Record<string, unknown>;
	rowMode?: "array" | "object";
	resultRows?: unknown[];
	countChanges?: number;
}

function patchFetchForFileUris() {
	const originalFetch = globalThis.fetch;
	globalThis.fetch = async (input: RequestInfo | URL, init?: RequestInit) => {
		const toResponse = async (target: string) => {
			const filePath = fileURLToPath(target);
			const data = await readFile(filePath);
			const headers: Record<string, string> = {};
			if (target.endsWith(".wasm")) {
				headers["Content-Type"] = "application/wasm";
			}
			return new Response(data, { status: 200, headers });
		};
		if (typeof input === "string" && input.startsWith("file://")) {
			return toResponse(input);
		}
		if (input instanceof URL && input.protocol === "file:") {
			return toResponse(input.href);
		}
		return originalFetch(input, init);
	};
	return () => {
		globalThis.fetch = originalFetch;
	};
}

export async function createNodeSqlitePromiser(): Promise<NodeWorkerPromiser> {
	const restoreFetch = patchFetchForFileUris();
	try {
		const moduleUrl = new URL(
			"../../../node_modules/@sqlite.org/sqlite-wasm/sqlite-wasm/jswasm/sqlite3.mjs",
			import.meta.url,
		);
		const { default: sqlite3InitModule } = await import(moduleUrl.href);
		const sqlite3 = await sqlite3InitModule({
			print: () => undefined,
			printErr: (...parts: unknown[]) => {
				console.error("[node-sqlite]", ...parts);
			},
		});
		const db = new sqlite3.oo1.DB();
		let opened = false;
		return async function nodePromiser(type: string, args: WorkerMessage = {}) {
			switch (type) {
				case "open": {
					if (!opened) {
						opened = true;
					}
					return { dbId: 1, result: { dbId: 1 } };
				}
				case "close": {
					opened = false;
					return {};
				}
				case "exec": {
					const execArgs = args as ExecArgs;
					const resultRows = Array.isArray(execArgs.resultRows)
						? execArgs.resultRows
						: [];
					db.exec({
						sql: execArgs.sql,
						bind: execArgs.bind,
						rowMode: execArgs.rowMode,
						resultRows,
					});
					const changeCount = execArgs.countChanges
						? db.changes(false, false)
						: undefined;
					return { result: { resultRows, changeCount } };
				}
				default:
					throw new Error(`Unsupported worker message: ${type}`);
			}
		};
	} finally {
		restoreFetch();
	}
}

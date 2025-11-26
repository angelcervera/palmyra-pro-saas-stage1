import sqlite3InitModule from "@sqlite.org/sqlite-wasm";

// NOTE: Vite handles ?url imports and serves the WASM with application/wasm.
// eslint-disable-next-line @typescript-eslint/ban-ts-comment
// @ts-ignore
import wasmUrl from "./sqlite3.wasm?url";

const logPrefix = "[sqlite-wasm-worker]";
const print = (...parts: unknown[]) => console.log(logPrefix, ...parts);
const printErr = (...parts: unknown[]) => console.error(logPrefix, ...parts);

sqlite3InitModule({
	print,
	printErr,
	locateFile: () => wasmUrl,
})
	.then(async (sqlite3) => {
		try {
			if (typeof sqlite3.installOpfsSAHPoolVfs === "function") {
				await sqlite3.installOpfsSAHPoolVfs({ name: "opfs-sahpool" });
				print("opfs-sahpool VFS ready");
			} else {
				print("opfs-sahpool not available; continuing without it");
			}
		} catch (error) {
			printErr("Failed to install SAH VFS", error);
		}
		sqlite3.initWorker1API();
	})
	.catch((error) => {
		printErr("Failed to bootstrap sqlite3 worker", error);
	});

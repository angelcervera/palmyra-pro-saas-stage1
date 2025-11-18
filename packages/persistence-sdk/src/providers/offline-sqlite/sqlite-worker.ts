import sqlite3InitModule from "@sqlite.org/sqlite-wasm";

const logPrefix = "[@zengate/sqlite-worker]";

const print = (...parts: unknown[]) => {
	// eslint-disable-next-line no-console
	console.log(logPrefix, ...parts);
};

const printErr = (...parts: unknown[]) => {
	// eslint-disable-next-line no-console
	console.error(logPrefix, ...parts);
};

sqlite3InitModule({
	print,
	printErr,
})
	.then(async (sqlite3) => {
		try {
			if (typeof sqlite3.installOpfsSAHPoolVfs === "function") {
				await sqlite3.installOpfsSAHPoolVfs({
					name: "opfs-sahpool",
					initialCapacity: 8,
				});
				print("opfs-sahpool VFS ready");
			} else {
				print("installOpfsSAHPoolVfs not available, continuing without it");
			}
		} catch (error) {
			printErr("Failed to install opfs-sahpool VFS", error);
		}

		sqlite3.initWorker1API();
	})
	.catch((error) => {
		printErr("Failed to bootstrap sqlite3 worker", error);
	});

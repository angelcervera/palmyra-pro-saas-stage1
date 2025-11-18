import sqlite3InitModule from "@sqlite.org/sqlite-wasm";

const logPrefix = "[@zengate/sqlite-worker]";

// helper to avoid eslint no-console errors in this module.
const print = (...parts: unknown[]) => {
	// eslint-disable-next-line no-console
	console.log(logPrefix, ...parts);
};

print("worker booting");

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

// Minimal Worker1 promiser demo: open DB, insert 100 rows, read them back, close.
// The worker bundles sqlite3.wasm via locateFile and runs in module mode.

// eslint-disable-next-line @typescript-eslint/ban-ts-comment
// @ts-ignore - package lacks types for sqlite3Worker1Promiser
import { sqlite3Worker1Promiser } from "@sqlite.org/sqlite-wasm";

const runBtn = document.getElementById("run") as HTMLButtonElement;
const statusEl = document.getElementById("status") as HTMLParagraphElement;
const outputEl = document.getElementById("output") as HTMLPreElement;

function setStatus(text: string) {
	statusEl.textContent = text;
}

function setOutput(data: unknown) {
	outputEl.textContent = typeof data === "string" ? data : JSON.stringify(data, null, 2);
}

async function runTest() {
	setStatus("Starting…");
	setOutput("");
	try {
		const promiser = await sqlite3Worker1Promiser({
			worker: () => new Worker(new URL("./sqlite-worker.ts", import.meta.url), { type: "module" }),
			// Provide a default db name; OPFS VFS will store it locally.
			filename: "file:wasm-demo.db?vfs=opfs-sahpool",
		});

		setStatus("Opening database…");
		await promiser("open", { filename: "file:wasm-demo.db?vfs=opfs-sahpool" });

		setStatus("Creating table…");
		await promiser("exec", { sql: "CREATE TABLE IF NOT EXISTS items(id INTEGER PRIMARY KEY, name TEXT); DELETE FROM items;" });

		setStatus("Inserting 100 rows…");
		for (let i = 1; i <= 100; i += 1) {
			await promiser("exec", {
				sql: "INSERT INTO items(name) VALUES (?);",
				bind: [`item-${i}`],
			});
		}

		setStatus("Querying rows…");
		const res = await promiser("exec", {
			sql: "SELECT id, name FROM items ORDER BY id;",
			resultRows: [],
			rowMode: "object",
		});
		setOutput(res.result?.resultRows ?? res);

		setStatus("Closing database…");
		await promiser("close", {});
		setStatus("Done ✅");
	} catch (error) {
		setStatus("Error");
		setOutput(error instanceof Error ? error.message : String(error));
		console.error(error);
	}
}

runBtn.addEventListener("click", () => {
	void runTest();
});

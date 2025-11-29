import type { PersistenceClient } from "@zengateglobal/persistence-sdk";

import { pushToast } from "../../components/toast";
import { defaultClientPromise } from "./client";

/**
 * Helper to run operations with the shared client and surface user-facing errors.
 */
export async function runWithClient<T>(
	opLabel: string,
	fn: (c: PersistenceClient) => Promise<T>,
): Promise<T> {
	try {
		const client = await defaultClientPromise;
		return await fn(client);
	} catch (error) {
		const message = `${opLabel} failed: ${describeError(error)}`;
		pushToast({ kind: "error", title: opLabel, description: message });
		throw new Error(message);
	}
}

function describeError(error: unknown): string {
	if (error instanceof Error) return error.message;
	if (typeof error === "string") return error;
	try {
		return JSON.stringify(error);
	} catch {
		return "Unknown error";
	}
}

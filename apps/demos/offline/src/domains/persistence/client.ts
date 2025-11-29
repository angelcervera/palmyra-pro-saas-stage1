import {
	createOfflineDexieProvider,
	PersistenceClient,
	type Schema,
} from "@zengateglobal/persistence-sdk";

import { pushToast } from "../../components/toast";

export const OFFLINE_ENV_KEY = "demo";
export const OFFLINE_TENANT_ID = "demo-tenant";
export const OFFLINE_APP_NAME = "offline-demo";

/**
 * Build a shared PersistenceClient promise for the offline demo.
 * Uses the SDK’s offline Dexie provider so everything stays in the browser.
 */
export function buildClientPromise(
	schemas: Schema[],
): Promise<PersistenceClient> {
	return createOfflineDexieProvider({
		envKey: OFFLINE_ENV_KEY,
		tenantId: OFFLINE_TENANT_ID,
		appName: OFFLINE_APP_NAME,
		schemas,
	}).then((provider) => new PersistenceClient([provider]));
}

const clientCache = new Map<string, Promise<PersistenceClient>>();

/**
 * Get (and memoize) a client promise for the provided schema set.
 * Keyed by the sorted table names to keep it simple for the demo.
 */
export function getClientPromise(schemas: Schema[]): Promise<PersistenceClient> {
	const key = schemas.map((s) => s.tableName).sort().join(",");
	if (clientCache.has(key)) {
		return clientCache.get(key)!;
	}
	const promise = buildClientPromise(schemas);
	clientCache.set(key, promise);
	return promise;
}

// Default shared client for domains that don't care about schemas (rare for the demo).
export const clientPromise = getClientPromise([]);

/**
 * Helper to run operations with the shared client and surface user-facing errors.
 */
export async function runWithClient<T>(
	clientPromise: Promise<PersistenceClient>,
	opLabel: string,
	fn: (c: PersistenceClient) => Promise<T>,
): Promise<T> {
	try {
		const client = await clientPromise;
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

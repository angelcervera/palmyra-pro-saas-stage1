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
export async function buildClientPromise(): Promise<PersistenceClient> {
	const provider = await createOfflineDexieProvider({
		envKey: OFFLINE_ENV_KEY,
		tenantId: OFFLINE_TENANT_ID,
		appName: OFFLINE_APP_NAME,
		schemas: [],
	});
	return new PersistenceClient([provider]);
}

const clientCache = new Map<string, Promise<PersistenceClient>>();

/**
 * Get (and memoize) a client promise keyed by a single default entry.
 */
export function getClientPromise(): Promise<PersistenceClient> {
	if (clientCache.has("default")) {
		return clientCache.get("default")!;
	}
	const promise = buildClientPromise();
	clientCache.set("default", promise);
	return promise;
}

// Default shared client for the demo; schemas can be injected via setDefaultSchemas.
const defaultClientPromise = getClientPromise();
let schemasInitialized = false;

export async function setDefaultSchemas(schemas: Schema[]): Promise<void> {
	if (schemasInitialized || schemas.length === 0) return;
	const client = await defaultClientPromise;
	await client.setMetadata(schemas);
	schemasInitialized = true;
}

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

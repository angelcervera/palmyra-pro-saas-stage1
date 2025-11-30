import {
	createOfflineDexieProvider,
	PersistenceClient,
} from "@zengateglobal/persistence-sdk";

export const OFFLINE_ENV_KEY = "demo";
export const OFFLINE_TENANT_ID = "demo-tenant";
export const OFFLINE_APP_NAME = "offline-demo";

/**
 * Build a shared PersistenceClient promise for the offline demo.
 * Uses the SDK’s offline Dexie provider so everything stays in the browser.
 * Schemas are handled by the SDK, so we pass an empty array.
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

// Single shared client for the demo.
export const defaultClientPromise = buildClientPromise();

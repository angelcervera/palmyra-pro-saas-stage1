import {
	createOfflineDexieProvider,
	createOnlineApiSdkProvider,
	PersistenceClient,
} from "@zengateglobal/persistence-sdk";

export const OFFLINE_ENV_KEY = "demo";
export const OFFLINE_TENANT_ID = "demo-tenant";
export const OFFLINE_APP_NAME = "offline-demo";
const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? "/api/v1";
const API_TOKEN = import.meta.env.VITE_API_TOKEN;

/**
 * Build a shared PersistenceClient promise for the offline demo.
 * Uses the SDK’s offline Dexie provider so everything stays in the browser.
 * Schemas are handled by the SDK, so we pass an empty array.
 */
export async function buildClientPromise(): Promise<PersistenceClient> {
	const offlineProvider = await createOfflineDexieProvider({
		envKey: OFFLINE_ENV_KEY,
		tenantId: OFFLINE_TENANT_ID,
		appName: OFFLINE_APP_NAME,
		schemas: [],
	});
	const onlineProvider = createOnlineApiSdkProvider({
		baseUrl: API_BASE_URL,
		getToken: () => API_TOKEN, // FIXME: Use a real auth provider.
	});

	return new PersistenceClient([offlineProvider, onlineProvider]);
}

// Single shared client for the demo.
export const defaultClientPromise = buildClientPromise();

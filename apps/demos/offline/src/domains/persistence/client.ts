import {
	createOfflineDexieProvider,
	PersistenceClient,
	type Schema,
} from "@zengateglobal/persistence-sdk";

import { pushToast } from "../../components/toast";

export const OFFLINE_ENV_KEY = "demo";
export const OFFLINE_TENANT_ID = "demo-tenant";
export const OFFLINE_APP_NAME = "offline-demo";

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

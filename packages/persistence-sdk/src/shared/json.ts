type JsonPrimitive = string | number | boolean | null;
export type JsonValue =
	| JsonPrimitive
	| JsonValue[]
	| {
			[key: string]: JsonValue;
	  };

export type JsonObject = Record<string, JsonValue>;

const isPlainObject = (value: unknown): value is Record<string, unknown> => {
	if (value === null || typeof value !== "object") {
		return false;
	}
	const proto = Object.getPrototypeOf(value);
	return proto === Object.prototype || proto === null;
};

export function toWireJson(value: unknown, path: string[] = []): JsonValue {
	if (value === null || value === undefined) {
		return null;
	}

	const type = typeof value;
	if (type === "string" || type === "number" || type === "boolean") {
		return value as JsonValue;
	}

	if (value instanceof Date) {
		return value.toISOString();
	}

	if (Array.isArray(value)) {
		return value.map((entry, idx) => toWireJson(entry, [...path, String(idx)]));
	}

	if (isPlainObject(value)) {
		const result: JsonObject = {};
		for (const [key, entry] of Object.entries(value)) {
			result[key] = toWireJson(entry, [...path, key]);
		}
		return result;
	}

	throw new Error(
		`Unsupported value at "${path.join(".") || "<root>"}": ${
			value instanceof Object ? value.constructor.name : typeof value
		}`,
	);
}

export const fromWireJson = <TPayload = unknown>(
	value: JsonValue,
): TPayload => {
	return value as unknown as TPayload;
};

export function toJsonObject(value: unknown): JsonObject {
	const jsonValue = toWireJson(value);
	if (
		jsonValue === null ||
		typeof jsonValue !== "object" ||
		Array.isArray(jsonValue)
	) {
		throw new Error("Value must serialize to a JSON object");
	}
	return jsonValue as JsonObject;
}

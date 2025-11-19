export type ProblemDetails = {
	type?: string;
	title?: string;
	status?: number;
	detail?: string;
	instance?: string;
	errors?: Record<string, unknown>;
};

export type ApiError = {
	status: number;
	problem?: ProblemDetails;
};

export type FetchClientOptions = {
	baseUrl: string;
	getToken?: () => string | undefined;
};

function resolveBaseUrl(baseUrl: string): string {
	if (/^https?:\/\//i.test(baseUrl)) {
		return baseUrl;
	}
	if (typeof window !== "undefined" && window.location) {
		return new URL(baseUrl, window.location.origin).toString();
	}
	throw new Error(
		"createFetchClient: absolute baseUrl required when window.location is unavailable",
	);
}

export function createFetchClient(opts: FetchClientOptions) {
	const { baseUrl, getToken } = opts;
	const resolvedBaseUrl = resolveBaseUrl(baseUrl);
	return async function request(input: string | URL, init: RequestInit = {}) {
		const raw = typeof input === "string" ? input : input.toString();
		const base = resolvedBaseUrl.endsWith("/")
			? resolvedBaseUrl
			: `${resolvedBaseUrl}/`;
		const isAbsolute = /^[a-z][a-z\d+\-.]*:/i.test(raw) || raw.startsWith("//");
		const normalized = isAbsolute ? raw : raw.replace(/^\/+/, "");
		const url = new URL(normalized, base);
		const headers = new Headers(init.headers);
		const token = getToken?.();
		if (token) headers.set("Authorization", `Bearer ${token}`);
		if (!headers.has("Accept")) {
			headers.set("Accept", "application/json");
		}
		const res = await fetch(url, { ...init, headers });
		if (!res.ok) {
			let problem: ProblemDetails | undefined;
			try {
				const data = await res.clone().json();
				if (data && typeof data === "object") {
					problem = {
						type: typeof data.type === "string" ? data.type : undefined,
						title: typeof data.title === "string" ? data.title : undefined,
						status: typeof data.status === "number" ? data.status : undefined,
						detail: typeof data.detail === "string" ? data.detail : undefined,
						instance:
							typeof data.instance === "string" ? data.instance : undefined,
						errors:
							typeof data.errors === "object"
								? (data.errors as Record<string, unknown>)
								: undefined,
					};
				}
			} catch {}
			const err: ApiError = { status: res.status, problem };
			throw err;
		}
		const contentType = res.headers.get("content-type") || "";
		if (contentType.includes("application/json")) {
			return res.json();
		}
		return res.text();
	};
}

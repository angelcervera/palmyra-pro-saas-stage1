type ProblemDetailsLike = {
	title?: string;
	detail?: string;
	status?: number;
};

export function wrapProviderError(message: string, error: unknown): Error {
	if (error instanceof Error) {
		return new Error(`${message}: ${error.message}`, { cause: error });
	}
	if (typeof error === "string") {
		return new Error(`${message}: ${error}`);
	}
	if (error && typeof error === "object") {
		const typed = error as ProblemDetailsLike;
		const parts = [message];
		if (typed.title) {
			parts.push(typed.title);
		}
		if (typed.detail) {
			parts.push(typed.detail);
		} else if (typed.status) {
			parts.push(`status ${typed.status}`);
		}
		return new Error(parts.join(": "));
	}
	return new Error(message);
}

export function describeProviderError(error: unknown): string {
	if (error instanceof Error) {
		return error.message;
	}
	if (typeof error === "string") {
		return error;
	}
	if (error && typeof error === "object") {
		const typed = error as ProblemDetailsLike;
		const parts: string[] = [];
		if (typed.title) {
			parts.push(typed.title);
		}
		if (typed.detail) {
			parts.push(typed.detail);
		} else if (typed.status) {
			parts.push(`status ${typed.status}`);
		}
		if (parts.length > 0) {
			return parts.join(": ");
		}
		try {
			return JSON.stringify(error);
		} catch {
			return "Unknown API error";
		}
	}
	return "Unknown API error";
}

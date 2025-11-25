import type { PropsWithChildren } from "react";
import { createContext, useContext, useMemo, useState } from "react";

type AuthState = {
	token?: string;
};

type AuthContextValue = AuthState & {
	signIn: (token: string) => void;
	signOut: () => void;
};

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: PropsWithChildren) {
	const [token, setToken] = useState<string | undefined>(
		() => sessionStorage.getItem("jwt") ?? undefined,
	);

	const value = useMemo<AuthContextValue>(
		() => ({
			token,
			signIn: (nextToken) => {
				setToken(nextToken);
				sessionStorage.setItem("jwt", nextToken);
			},
			signOut: () => {
				setToken(undefined);
				sessionStorage.removeItem("jwt");
			},
		}),
		[token],
	);

	return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth() {
	const ctx = useContext(AuthContext);
	if (!ctx) throw new Error("useAuth must be used within <AuthProvider>");
	return ctx;
}

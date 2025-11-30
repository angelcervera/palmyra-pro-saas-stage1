import { ThemeProvider as NextThemes } from "next-themes";
import type { PropsWithChildren } from "react";

export function ThemeProvider({ children }: PropsWithChildren) {
	return (
		<NextThemes
			attribute="class"
			defaultTheme="system"
			enableSystem
			storageKey="web-platform-admin-theme"
		>
			{children}
		</NextThemes>
	);
}

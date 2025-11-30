import { MoonIcon, SunIcon } from "lucide-react";
import { useTheme } from "next-themes";
import { Button } from "@/components/ui/button";

export function ThemeToggle() {
	const { theme, setTheme } = useTheme();
	const isDark = theme === "dark";

	return (
		<Button
			variant="ghost"
			size="icon"
			aria-label="Toggle theme"
			onClick={() => setTheme(isDark ? "light" : "dark")}
			className="size-8"
		>
			{isDark ? (
				<MoonIcon className="size-4" />
			) : (
				<SunIcon className="size-4" />
			)}
		</Button>
	);
}

import * as React from "react";

import { cn } from "@/lib/utils";

export type TextareaProps = React.TextareaHTMLAttributes<HTMLTextAreaElement>;

const Textarea = React.forwardRef<HTMLTextAreaElement, TextareaProps>(
	({ className, ...props }, ref) => {
		return (
			<textarea
				ref={ref}
				className={cn(
					"flex min-h-[80px] w-full rounded-md border border-input bg-background px-3 py-2 text-sm shadow-sm transition-[color,box-shadow] placeholder:text-muted-foreground outline-none disabled:cursor-not-allowed disabled:opacity-50",
					// Focus + error states align with `Input`
					"focus-visible:border-ring focus-visible:ring-ring/50 focus-visible:ring-[3px]",
					"aria-invalid:ring-destructive/20 dark:aria-invalid:ring-destructive/40 aria-invalid:border-destructive",
					className,
				)}
				{...props}
			/>
		);
	},
);
Textarea.displayName = "Textarea";

export { Textarea };

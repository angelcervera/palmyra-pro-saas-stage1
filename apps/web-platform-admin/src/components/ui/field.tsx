import * as React from "react";
import { cn } from "@/lib/utils";

const Fieldset = React.forwardRef<
	HTMLFieldSetElement,
	React.HTMLAttributes<HTMLFieldSetElement>
>(({ className, ...props }, ref) => (
	<fieldset
		ref={ref}
		className={cn("grid gap-6 rounded-lg border border-border p-6", className)}
		{...props}
	/>
));
Fieldset.displayName = "Fieldset";

const FieldsetLegend = React.forwardRef<
	HTMLLegendElement,
	React.HTMLAttributes<HTMLLegendElement>
>(({ className, ...props }, ref) => (
	<legend
		ref={ref}
		className={cn(
			"text-base font-semibold leading-none tracking-tight",
			className,
		)}
		{...props}
	/>
));
FieldsetLegend.displayName = "FieldsetLegend";

const FieldsetDescription = React.forwardRef<
	HTMLParagraphElement,
	React.HTMLAttributes<HTMLParagraphElement>
>(({ className, ...props }, ref) => (
	<p
		ref={ref}
		className={cn("text-sm text-muted-foreground", className)}
		{...props}
	/>
));
FieldsetDescription.displayName = "FieldsetDescription";

const FieldGroup = React.forwardRef<
	HTMLDivElement,
	React.HTMLAttributes<HTMLDivElement>
>(({ className, ...props }, ref) => (
	<div
		ref={ref}
		className={cn("grid gap-4 sm:grid-cols-2", className)}
		{...props}
	/>
));
FieldGroup.displayName = "FieldGroup";

export { Fieldset, FieldsetLegend, FieldsetDescription, FieldGroup };

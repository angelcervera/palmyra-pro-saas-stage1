import type { CSSProperties } from "react";
import { Outlet } from "react-router-dom";

import { AppSidebar } from "@/components/app-sidebar";
import { SiteHeader } from "@/components/site-header";
import { SidebarInset, SidebarProvider } from "@/components/ui/sidebar";

const layoutStyle = {
	"--sidebar-width": "calc(var(--spacing) * 72)",
	"--header-height": "calc(var(--spacing) * 12)",
} as CSSProperties;

export function AdminLayout() {
	return (
		<SidebarProvider style={layoutStyle}>
			<AppSidebar variant="inset" />
			<SidebarInset>
				<SiteHeader />
				<div className="flex flex-1 flex-col">
					<Outlet />
				</div>
			</SidebarInset>
		</SidebarProvider>
	);
}

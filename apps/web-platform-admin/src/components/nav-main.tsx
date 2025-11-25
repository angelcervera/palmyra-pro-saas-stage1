import type { Icon } from "@tabler/icons-react";
import { NavLink } from "react-router-dom";

import {
	SidebarGroup,
	SidebarGroupContent,
	SidebarMenu,
	SidebarMenuButton,
	SidebarMenuItem,
} from "@/components/ui/sidebar";

export function NavMain({
	items,
}: {
	items: {
		title: string;
		url: string;
		icon?: Icon;
		exact?: boolean;
	}[];
}) {
	return (
		<SidebarGroup>
			<SidebarGroupContent>
				<SidebarMenu>
					{items.map((item) => (
						<SidebarMenuItem key={item.title}>
							<SidebarMenuButton asChild tooltip={item.title}>
								<NavLink
									to={item.url}
									className={({ isActive }) =>
										isActive ? "data-[active=true]" : undefined
									}
									end={item.exact ?? false}
								>
									{item.icon && <item.icon />}
									<span>{item.title}</span>
								</NavLink>
							</SidebarMenuButton>
						</SidebarMenuItem>
					))}
				</SidebarMenu>
			</SidebarGroupContent>
		</SidebarGroup>
	);
}

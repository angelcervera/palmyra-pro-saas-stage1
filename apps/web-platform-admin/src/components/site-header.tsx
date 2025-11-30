import { useLocation } from "react-router-dom";
import { ThemeToggle } from "@/components/theme-toggle";
import {
	Breadcrumb,
	BreadcrumbItem,
	BreadcrumbLink,
	BreadcrumbList,
	BreadcrumbPage,
	BreadcrumbSeparator,
} from "@/components/ui/breadcrumb";
import { Separator } from "@/components/ui/separator";
import { SidebarTrigger } from "@/components/ui/sidebar";

function useRouteInfo() {
	const { pathname } = useLocation();
	const segments = pathname.split("/").filter(Boolean);

	const map: Record<string, string> = {
		"": "Dashboard",
		"schema-categories": "Schema Categories",
		users: "Users",
	};

	const first = segments[0] ?? "";
	const title = map[first] ?? map[""];
	const lastLabel =
		segments.length > 1
			? decodeURIComponent(segments[segments.length - 1])
			: undefined;
	return { title, segments, lastLabel, pathname };
}

export function SiteHeader() {
	const { title, segments, lastLabel } = useRouteInfo();
	const isRoot = segments.length === 0;

	return (
		<header className="flex h-(--header-height) shrink-0 items-center gap-2 border-b transition-[width,height] ease-linear group-has-data-[collapsible=icon]/sidebar-wrapper:h-(--header-height)">
			<div className="flex w-full items-center gap-1 px-4 lg:gap-2 lg:px-6">
				<SidebarTrigger className="-ml-1" />
				<Separator
					orientation="vertical"
					className="mx-2 data-[orientation=vertical]:h-4"
				/>
				<div className="flex min-w-0 flex-col">
					<h1 className="text-base font-medium truncate">{title}</h1>
					{!isRoot && (
						<Breadcrumb>
							<BreadcrumbList>
								<BreadcrumbItem>
									<BreadcrumbLink href="/">Dashboard</BreadcrumbLink>
								</BreadcrumbItem>
								<BreadcrumbSeparator />
								<BreadcrumbItem>
									{segments.length === 1 ? (
										<BreadcrumbPage>{title}</BreadcrumbPage>
									) : (
										<BreadcrumbLink href={`/${segments[0]}`}>
											{title}
										</BreadcrumbLink>
									)}
								</BreadcrumbItem>
								{segments.length > 1 && (
									<>
										<BreadcrumbSeparator />
										<BreadcrumbItem>
											<BreadcrumbPage>{lastLabel}</BreadcrumbPage>
										</BreadcrumbItem>
									</>
								)}
							</BreadcrumbList>
						</Breadcrumb>
					)}
				</div>
				<div className="ml-auto flex items-center gap-2">
					<ThemeToggle />
				</div>
			</div>
		</header>
	);
}

export default function NotFoundPage() {
	return (
		<div className="flex h-[60vh] w-full items-center justify-center p-6 text-center">
			<div>
				<h1 className="text-2xl font-semibold">Page not found</h1>
				<p className="text-muted-foreground mt-2">
					The page you’re looking for doesn’t exist.
				</p>
				<a
					href="/"
					className="mt-4 inline-flex rounded-md bg-primary px-3 py-1.5 text-sm font-medium text-primary-foreground hover:bg-primary/90"
				>
					Go to dashboard
				</a>
			</div>
		</div>
	);
}

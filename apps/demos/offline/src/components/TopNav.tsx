import { Link } from "react-router-dom";

type Props = {
	active?: "sync" | "persons";
};

export function TopNav({ active }: Props) {
	return (
		<div className="app-shell" style={{ paddingTop: 16, paddingBottom: 0 }}>
			<div className="toolbar" style={{ gap: 12, alignItems: "center" }}>
				<Link className="link" to="/sync" aria-current={active === "sync"}>
					Sync
				</Link>
				<span aria-hidden="true">|</span>
				<Link
					className="link"
					to="/persons"
					aria-current={active === "persons"}
				>
					Persons
				</Link>
			</div>
		</div>
	);
}

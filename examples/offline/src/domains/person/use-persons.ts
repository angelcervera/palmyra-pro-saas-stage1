import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
	createPerson,
	deletePerson,
	getPerson,
	listPersons,
	type Person,
	type PersonRecord,
	updatePerson,
} from "./persistence";

const PERSON_LIST_KEY = ["person-demo", "list"] as const;

export function usePersonList(
	params: { page: number; pageSize: number } | null,
) {
	return useQuery({
		enabled: Boolean(params),
		queryKey: params ? [...PERSON_LIST_KEY, params] : PERSON_LIST_KEY,
		queryFn: () => {
			if (!params) throw new Error("params not set");
			return listPersons(params);
		},
		staleTime: 5_000,
	});
}

export function usePerson(entityId?: string) {
	return useQuery({
		enabled: Boolean(entityId),
		queryKey: ["person-demo", "detail", entityId],
		queryFn: () => {
			if (!entityId) throw new Error("Missing entity id");
			return getPerson(entityId);
		},
		staleTime: 5_000,
	});
}

export function useCreatePerson() {
	const qc = useQueryClient();
	return useMutation({
		mutationFn: (input: Person) => createPerson(input),
		onSuccess: () => qc.invalidateQueries({ queryKey: PERSON_LIST_KEY }),
	});
}

export function useUpdatePerson(entityId: string) {
	const qc = useQueryClient();
	return useMutation({
		mutationFn: (input: Person) => updatePerson(entityId, input),
		onSuccess: () => {
			qc.invalidateQueries({ queryKey: PERSON_LIST_KEY });
			qc.invalidateQueries({ queryKey: ["person-demo", "detail", entityId] });
		},
	});
}

export function useDeletePerson() {
	const qc = useQueryClient();
	return useMutation({
		mutationFn: (entityId: string) => deletePerson(entityId),
		onSuccess: () => qc.invalidateQueries({ queryKey: PERSON_LIST_KEY }),
	});
}

export type { Person, PersonRecord };

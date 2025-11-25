import { zodResolver } from "@hookform/resolvers/zod";
import { useEffect } from "react";
import { useForm } from "react-hook-form";
import { useNavigate } from "react-router-dom";
import { z } from "zod";

import type { Person } from "../persistence";

const schema = z.object({
	name: z.string().min(1, "Name is required"),
	surname: z.string().min(1, "Surname is required"),
	age: z.number().int().min(0).max(150),
	dob: z.string().min(1, "Date of birth is required"),
	phoneNumber: z
		.string()
		.regex(/^[+]?[1-9]\d{7,14}$/i, "Use an E.164 style phone number"),
	photo: z.string().url("Photo must be a valid URL"),
});

export type PersonFormValues = z.infer<typeof schema>;

type Props = {
	defaultValues?: Person;
	onSubmit(values: PersonFormValues): Promise<void> | void;
	submitLabel: string;
	cancelTo: string;
	isSubmitting?: boolean;
};

export function PersonForm({
	defaultValues,
	onSubmit,
	submitLabel,
	cancelTo,
	isSubmitting,
}: Props) {
	const navigate = useNavigate();
	const form = useForm<PersonFormValues>({
		resolver: zodResolver(schema),
		defaultValues: defaultValues ?? {
			name: "",
			surname: "",
			age: 0,
			dob: "",
			phoneNumber: "+",
			photo: "",
		},
	});

	useEffect(() => {
		if (defaultValues) {
			form.reset(defaultValues);
		}
	}, [defaultValues, form]);

	const { register, handleSubmit, formState } = form;

	return (
		<div className="card">
			<h2 style={{ marginTop: 0 }}>Person form</h2>
			<form
				onSubmit={handleSubmit(async (values) => {
					await onSubmit(values);
				})}
				className="form-grid"
			>
				<div>
					<label className="label" htmlFor="name">
						Name
					</label>
					<input
						id="name"
						className="input"
						{...register("name")}
						placeholder="Ada"
					/>
					{formState.errors.name && (
						<div className="error">{formState.errors.name.message}</div>
					)}
				</div>
				<div>
					<label className="label" htmlFor="surname">
						Surname
					</label>
					<input
						id="surname"
						className="input"
						{...register("surname")}
						placeholder="Lovelace"
					/>
					{formState.errors.surname && (
						<div className="error">{formState.errors.surname.message}</div>
					)}
				</div>
				<div>
					<label className="label" htmlFor="age">
						Age
					</label>
					<input
						id="age"
						className="input"
						type="number"
						min={0}
						max={150}
						{...register("age", { valueAsNumber: true })}
					/>
					{formState.errors.age && (
						<div className="error">{formState.errors.age.message}</div>
					)}
				</div>
				<div>
					<label className="label" htmlFor="dob">
						Date of birth
					</label>
					<input id="dob" className="input" type="date" {...register("dob")} />
					{formState.errors.dob && (
						<div className="error">{formState.errors.dob.message}</div>
					)}
				</div>
				<div>
					<label className="label" htmlFor="phone">
						Phone
					</label>
					<input
						id="phone"
						className="input"
						placeholder="+447000000000"
						{...register("phoneNumber")}
					/>
					{formState.errors.phoneNumber && (
						<div className="error">{formState.errors.phoneNumber.message}</div>
					)}
				</div>
				<div>
					<label className="label" htmlFor="photo">
						Photo URL
					</label>
					<input
						id="photo"
						className="input"
						placeholder="https://..."
						{...register("photo")}
					/>
					{formState.errors.photo && (
						<div className="error">{formState.errors.photo.message}</div>
					)}
				</div>
				<div className="actions" style={{ gridColumn: "1 / -1" }}>
					<button
						type="button"
						className="btn"
						onClick={() => navigate(cancelTo)}
					>
						Cancel
					</button>
					<button type="submit" className="btn primary" disabled={isSubmitting}>
						{isSubmitting ? "Saving..." : submitLabel}
					</button>
				</div>
			</form>
		</div>
	);
}

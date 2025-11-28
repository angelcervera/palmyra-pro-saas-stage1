import { expect, test } from "@playwright/test";

const DB_NAME = "demo-demo-tenant-offline-demo";

const samplePerson = {
	name: "Ada",
	surname: "Lovelace",
	age: "36",
	dob: "1815-12-10",
	phone: "+447000000000",
	photo: "https://example.com/ada.png",
};

test.describe("offline person demo", () => {
	test.beforeEach(async ({ context }) => {
		// Ensure a clean IndexedDB for every test run.
		await context.addInitScript(
			({ name }) => {
				indexedDB.deleteDatabase(name);
			},
			{ name: DB_NAME },
		);
	});

	test("create, edit and delete a person while offline capable", async ({
		page,
		context,
	}) => {
		await page.goto("/");

		// Create
		await page.getByRole("link", { name: "New person" }).click();
		await page.fill("#name", samplePerson.name);
		await page.fill("#surname", samplePerson.surname);
		await page.fill("#age", samplePerson.age);
		await page.fill("#dob", samplePerson.dob);
		await page.fill("#phone", samplePerson.phone);
		await page.fill("#photo", samplePerson.photo);
		await page.getByRole("button", { name: "Save person" }).click();

		await expect(page.getByText(samplePerson.name)).toBeVisible();
		await expect(page.getByText(samplePerson.surname)).toBeVisible();

		// Go offline and ensure data is still rendered from Dexie.
		await context.setOffline(true);
		await page.reload();
		await expect(page.getByText(samplePerson.name)).toBeVisible();

		// Edit while offline
		await page.getByRole("link", { name: "Edit" }).first().click();
		await page.fill("#surname", "Byron");
		await page.getByRole("button", { name: "Save changes" }).click();
		await expect(page.getByText("Byron")).toBeVisible();

		// Delete (dialog confirm)
		const dialogPromise = page.waitForEvent("dialog");
		await page.getByRole("button", { name: "Delete" }).click();
		const dialog = await dialogPromise;
		await dialog.accept();

		await expect(page.getByText("No persons yet")).toBeVisible();
	});
});

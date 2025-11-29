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

test.describe("person demo (online flow)", () => {
	test.beforeEach(async ({ page }) => {
		// Ensure a clean IndexedDB for every test run (once, before first load).
		await page.goto("/");
		await page.evaluate((name) => indexedDB.deleteDatabase(name), DB_NAME);
	});

	test("create, edit and delete a person (online)", async ({ page }) => {
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

		// Reload to ensure data persists across refresh.
		await page.reload();
		await expect(page.getByText(samplePerson.name)).toBeVisible();

		// Edit while offline
		await page.getByRole("link", { name: "Edit" }).first().click();
		await page.fill("#surname", "Byron");
		await page.getByRole("button", { name: "Save changes" }).click();
		await expect(page.getByText("Byron")).toBeVisible();

		// Delete via confirmation modal
		await page.getByRole("button", { name: "Delete" }).click();
		await page.getByRole("button", { name: "Yes, delete" }).click();

		await expect(page.getByText("No persons yet")).toBeVisible();
	});
});

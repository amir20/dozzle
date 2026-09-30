import { test, expect, devices } from "@playwright/test";

test.use({ ...devices["Pixel 5"] });

test.describe("mobile stats", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("http://dozzle:8080/");
  });

  test("shows CPU and memory stats for containers on the dashboard", async ({ page }) => {
    const row = page.locator("tr").filter({ has: page.getByTitle("dozzle_e2e_dozzle", { exact: true }) });
    await expect(row.getByText(/^\d+%$/)).toBeVisible();
    await expect(row.getByText(/^[\d.]+ ?[KMGT]?B$/)).toBeVisible();
  });

  test("opens a log view as a pushed screen: no tab bar, header pinned to the top, back goes home", async ({
    page,
  }) => {
    await expect(page.getByTestId("browse")).toBeVisible();

    const row = page.locator("tr").filter({ has: page.getByTitle("dozzle_e2e_dozzle", { exact: true }) });
    await row.getByTitle("dozzle_e2e_dozzle", { exact: true }).click();
    await expect(page).toHaveURL(/\/container\//);

    // The tab bar steps aside so the bottom of the stream stays readable.
    await expect(page.getByTestId("browse")).toBeHidden();

    const header = page.getByTestId("scrollable-header");
    await expect.poll(async () => (await header.boundingBox())?.y).toBeLessThanOrEqual(1);

    // ...and stays there once the logs scroll under it.
    await page.mouse.wheel(0, 600);
    await expect.poll(async () => (await header.boundingBox())?.y).toBeLessThanOrEqual(1);

    await page.getByRole("button", { name: "Back" }).click();
    await expect(page).toHaveURL(/\/$/);
    await expect(page.getByTestId("browse")).toBeVisible();
  });
});

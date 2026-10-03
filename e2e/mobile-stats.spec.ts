import { test, expect, devices } from "@playwright/test";

test.use({ ...devices["Pixel 5"] });

test.describe("mobile stats", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("http://dozzle:8080/");
  });

  test("shows CPU and memory stats for containers on the dashboard", async ({ page }) => {
    const row = page.locator("tr").filter({ has: page.getByTitle("dozzle_e2e_dozzle", { exact: true }) });
    await expect(row.getByText(/^\d+%$/)).toBeVisible();
    // Short units on a phone ("420M"), full ones elsewhere ("419.98 MB").
    await expect(row.getByText(/^[\d.]+ ?(Bytes|[KMGTB]B?)$/)).toBeVisible();
  });
});

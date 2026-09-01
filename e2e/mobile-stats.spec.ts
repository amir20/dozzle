import { test, expect, devices } from "@playwright/test";

test.use({ ...devices["Pixel 5"] });

test.describe("mobile stats", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("http://dozzle:8080/");
  });

  test("shows CPU and memory stats for containers on the dashboard", async ({ page }) => {
    const row = page.locator("tr").filter({ has: page.getByTitle("dozzle") });
    await expect(row.getByText(/^\d+%$/)).toBeVisible();
    await expect(row.getByText(/^[\d.]+ ?[KMGT]?B$/)).toBeVisible();
  });

  test("locks the container stats header directly under the mobile nav, with no gap or overlap", async ({ page }) => {
    const row = page.locator("tr").filter({ has: page.getByTitle("dozzle") });
    await row.getByTitle("dozzle").click();
    await expect(page).toHaveURL(/\/container\//);

    const nav = await page.getByTestId("navigation").boundingBox();
    const header = await page.getByTestId("scrollable-header").boundingBox();

    expect(nav).not.toBeNull();
    expect(header).not.toBeNull();
    // The stats header must sit flush against the bottom of the fixed nav bar:
    // any gap exposes empty space, any overlap clips the CPU/MEM chips underneath the nav.
    expect(Math.abs(header!.y - (nav!.y + nav!.height))).toBeLessThanOrEqual(1);

    // ...and stay there once the logs scroll under it.
    await page.mouse.wheel(0, 600);
    await expect
      .poll(async () => {
        const stuckNav = await page.getByTestId("navigation").boundingBox();
        const stuckHeader = await page.getByTestId("scrollable-header").boundingBox();
        return Math.abs(stuckHeader!.y - (stuckNav!.y + stuckNav!.height));
      })
      .toBeLessThanOrEqual(1);
  });
});

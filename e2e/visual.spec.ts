import { test, expect } from "@playwright/test";

test.beforeEach(async ({ page }) => {
  await page.goto("http://dozzle:8080/");
});

test.describe("default", () => {
  test("homepage", async ({ page }) => {
    await expect(page).toHaveScreenshot({
      mask: [page.locator("time"), page.locator("[data-ci-skip]")],
    });
  });
});

test.describe("dark", () => {
  test.use({ colorScheme: "dark" });
  test("homepage", async ({ page }) => {
    await expect(page).toHaveScreenshot({
      mask: [page.locator("time"), page.locator("[data-ci-skip]")],
    });
  });
});

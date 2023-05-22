import { test, expect } from "@playwright/test";

test.beforeEach(async ({ page }) => {
  await page.goto("http://dozzle:8080/");
});

test.describe("default", () => {
  test("homepage", async ({ page }) => {
    await page.addStyleTag({ content: `[data-ci-skip] { visibility: hidden; }` });
    await expect(page).toHaveScreenshot({});
  });
});

test.describe("dark", () => {
  test.use({ colorScheme: "dark" });
  test("homepage", async ({ page }) => {
    await page.addStyleTag({ content: `[data-ci-skip] { visibility: hidden; }` });
    await expect(page).toHaveScreenshot({});
  });
});

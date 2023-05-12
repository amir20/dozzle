import { test, expect } from "@playwright/test";

test.beforeEach(async ({ page }) => {
  await page.goto("http://dozzle:8080/");
});

test("has right title", async ({ page }) => {
  await expect(page).toHaveTitle(/.* - Dozzle/);
});

test("click on settings button", async ({ page }) => {
  await page.getByRole("link", { name: "Settings" }).click();
  await expect(page.getByRole("heading", { name: "About" })).toBeVisible();
});

import { test, expect } from "@playwright/test";

test("simple authentication", async ({ page }) => {
  await page.goto("http://simple-auth:8080/");
  await page.locator('input[name="username"]').fill("admin");
  await page.locator('input[name="password"]').fill("password");
  await page.locator('button[type="submit"]').click();
  await expect(page.getByTestId("settings")).toBeVisible();
});

import { test, expect } from "@playwright/test";

test.beforeEach(async ({ page }) => {
  await page.goto("http://custom_base:8080/foobarbase");
});

test("has right title", async ({ page }) => {
  await expect(page).toHaveTitle(/.* - Dozzle/);
});

test("url should have custom base", async ({ page }) => {
  await expect(page).toHaveURL(/foobarbase/);
});

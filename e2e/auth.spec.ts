import { test, expect } from "@playwright/test";

test("authentication", async ({ page }) => {
  await page.goto("http://auth:8080/");
  await page.locator('input[name="username"]').fill("foo");
  await page.locator('input[name="password"]').fill("bar");
  await page.locator('button[type="submit"]').click();
  await expect(page.getByTestId("containers")).toHaveText("Containers");
});

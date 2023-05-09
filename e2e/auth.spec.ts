import { test, expect } from "@playwright/test";

test("authentication", async ({ page }) => {
  await page.goto("http://localhost:9090/");
  await page.locator('input[name="username"]').fill("foo");
  await page.locator('input[name="password"]').fill("bar");
  await page.getByRole("button", { name: "Login" }).click();
  await expect(page.locator("p.menu-label")).toHaveText("Containers");
});

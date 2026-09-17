import { test, expect } from "@playwright/test";

test.beforeEach(async ({ page }) => {
  await page.goto("http://dozzle:8080/");
});

test("has right title", async ({ page }) => {
  await expect(page).toHaveTitle(/.* - Dozzle/);
});

test("has dashboard text", async ({ page }) => {
  await expect(page.getByText("container name")).toBeVisible();
});

test("click on settings button", async ({ page }) => {
  await page.getByTestId("settings").click();
  await expect(page.getByRole("heading", { name: "Logs" })).toBeVisible();
});

test("shortcut for fuzzy search", async ({ page }) => {
  await page.locator("body").press("Control+k");
  await expect(page.locator(".modal").getByPlaceholder("Search containers…")).toBeVisible();
});

test("route by name", async ({ page }) => {
  await page.goto("http://dozzle:8080/show?name=dozzle_e2e_dozzle");
  await expect(page).toHaveURL(/\/container/);
});

test("route by names skips unknown names", async ({ page }) => {
  await page.goto("http://dozzle:8080/show?name=dozzle_e2e_dozzle,does-not-exist");
  await expect(page).toHaveURL(/\/container/);
});

test.describe("es locale", () => {
  test.use({ locale: "es" });

  test("translated text", async ({ page }) => {
    await expect(page.getByTestId("search")).toContainText("Buscar");
  });
});

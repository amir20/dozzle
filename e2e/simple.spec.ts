import { test, expect } from "@playwright/test";

const BASE = "http://simple-auth:8080";

async function login(page: import("@playwright/test").Page) {
  await page.locator('input[name="username"]').fill("admin");
  await page.locator('input[name="password"]').fill("password");
  await page.locator('button[type="submit"]').click();
}

test("simple authentication", async ({ page }) => {
  await page.goto(BASE + "/");
  await login(page);
  await expect(page.getByTestId("settings")).toBeVisible();
});

test("unauthenticated request redirects to the login page", async ({ page }) => {
  await page.goto(BASE + "/");
  await expect(page).toHaveURL(/\/login/);
  await expect(page.locator('input[name="username"]')).toBeVisible();
});

test("preserves the requested url as redirectUrl", async ({ page }) => {
  await page.goto(BASE + "/settings");
  await expect(page).toHaveURL(/\/login\?redirectUrl=/);
});

test("rejects a wrong password", async ({ page }) => {
  await page.goto(BASE + "/");
  await page.locator('input[name="username"]').fill("admin");
  await page.locator('input[name="password"]').fill("not-the-password");
  await page.locator('button[type="submit"]').click();

  await expect(page.getByText("Username or password are not valid")).toBeVisible();
  await expect(page.getByTestId("settings")).toBeHidden();
});

test("api rejects unauthenticated requests with 401", async ({ request }) => {
  // A regression here is a data leak, not a UI bug: /api must never serve an anonymous caller.
  for (const path of ["/api/version", "/api/events/stream", "/api/notifications/rules"]) {
    const response = await request.get(BASE + path, { maxRedirects: 0 });
    expect(response.status(), `${path} should be unauthorized`).toBe(401);
  }
});

test("logout clears the session", async ({ page }) => {
  await page.goto(BASE + "/");
  await login(page);
  await expect(page.getByTestId("settings")).toBeVisible();

  await page.getByTestId("user-menu").locator("label").click();
  await page.getByRole("button", { name: "Logout" }).click();

  await expect(page.locator('input[name="username"]')).toBeVisible();

  // The cookie must be gone server-side, not just visually logged out.
  const response = await page.request.get(BASE + "/api/version", { maxRedirects: 0 });
  expect(response.status()).toBe(401);
});

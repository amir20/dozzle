import { test, expect } from "@playwright/test";

const BASE = "http://simple-auth:8080";

async function login(page: import("@playwright/test").Page, username = "admin") {
  await page.locator('input[name="username"]').fill(username);
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

test("container search stays available without the cloud role", async ({ page }) => {
  // The topbar search is container search first and cloud log search only when linked.
  // Gating the whole control on the cloud role took container search away from everyone
  // who does not use Dozzle Cloud.
  await page.goto(BASE + "/");
  await login(page, "viewer");

  const search = page.getByTestId("search");
  await expect(search).toBeVisible();
  await expect(search).toContainText("Search containers");
});

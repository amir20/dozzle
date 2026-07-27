import { test, expect } from "@playwright/test";

// Covers DOZZLE_REMOTE_HOST (tcp:// through the socket proxy), a different code path
// than DOZZLE_REMOTE_AGENT. See agent.spec.ts for that one.
test.beforeEach(async ({ page }) => {
  await page.goto("http://remote:8080/");
});

test("has right title", async ({ page }) => {
  await expect(page).toHaveTitle(/.* - Dozzle/);
});

test("shows the labeled remote host", async ({ page }) => {
  await expect(page.getByText("remote-host")).toBeVisible();
});

test("select running container", async ({ page }) => {
  await page.getByTestId("side-menu").getByRole("link", { name: "dozzle" }).click();
  await expect(page).toHaveURL(/\/container/);
  await expect(page.getByText("Accepting connections")).toBeVisible();
});

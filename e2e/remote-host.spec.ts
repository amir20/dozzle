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
  // The host column on the dashboard, rather than a bare getByText: the label also
  // appears in the sidebar and the merge link, and which of them exist depends on
  // whether containers have loaded yet.
  await expect(page.getByRole("cell", { name: "remote-host" }).first()).toBeVisible();
});

test("select running container", async ({ page }) => {
  await page.getByTestId("side-menu").getByRole("link", { name: "dozzle" }).click();
  await expect(page).toHaveURL(/\/container/);
  await expect(page.getByText("Accepting connections")).toBeVisible();
});

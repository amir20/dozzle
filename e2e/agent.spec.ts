import { test, expect } from "@playwright/test";

// Covers DOZZLE_REMOTE_AGENT (gRPC agent). See remote-host.spec.ts for DOZZLE_REMOTE_HOST.
test.beforeEach(async ({ page }) => {
  await page.goto("http://dozzle-with-agent:8080/");
});

test("has right title", async ({ page }) => {
  await expect(page).toHaveTitle(/.* - Dozzle/);
});

test("select running container", async ({ page }) => {
  await page.getByTestId("side-menu").getByRole("link", { name: "dozzle" }).click();
  await expect(page).toHaveURL(/\/container/);
  await expect(page.getByText("Accepting connections")).toBeVisible();
});

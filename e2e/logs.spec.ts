import { test, expect } from "@playwright/test";

// Targets a dedicated instance filtered to the `logspam` container, which prints a
// numbered line every second. That makes "logs are actually streaming" assertable
// without depending on incidental output from Dozzle itself.
test.beforeEach(async ({ page }) => {
  await page.goto("http://logs-viewer:8080/");
  await page.getByTestId("side-menu").getByRole("link", { name: "logspam" }).click();
  await expect(page).toHaveURL(/\/container\//);
});

test("renders historical logs on load", async ({ page }) => {
  await expect(page.locator("ul[data-logs] > li").first()).toBeVisible();
  await expect(page.getByTestId("no-logs")).toBeHidden();
  await expect(page.getByText(/logspam line \d+/).first()).toBeVisible();
});

test("streams new logs over SSE without a reload", async ({ page }) => {
  const entries = page.locator("ul[data-logs] > li");
  await expect(entries.first()).toBeVisible();

  const initial = await entries.count();
  // logspam emits ~1 line/sec, so a few new entries must arrive on the open stream.
  await expect.poll(() => entries.count(), { timeout: 20_000 }).toBeGreaterThan(initial);
});

test("keeps the newest log entry in view while streaming", async ({ page }) => {
  const entries = page.locator("ul[data-logs] > li");
  await expect(entries.first()).toBeVisible();

  const initial = await entries.count();
  await expect.poll(() => entries.count(), { timeout: 20_000 }).toBeGreaterThan(initial);
  await expect(entries.last()).toBeInViewport();
});

import { test, expect, devices, type Page } from "@playwright/test";

// Not dozzle:8080. Every page load there adds debug lines to dozzle_e2e_dozzle's own log,
// and agent.spec and remote-host.spec read that log for its startup line, which falls out
// of the initial window once enough traffic lands on top of it.
const BASE = "http://logs-viewer:8080";
const CONTAINER = "dozzle_e2e_logspam";

// A device preset carries defaultBrowserType, which Playwright only accepts at the top
// of a file. Both describes run in Chromium, so it is dropped rather than split in two.
const { defaultBrowserType: _, ...pixel5 } = devices["Pixel 5"];

async function openContainer(page: Page) {
  const row = page.locator("tr").filter({ has: page.getByTitle(CONTAINER, { exact: true }) });
  await row.getByTitle(CONTAINER, { exact: true }).click();
  await expect(page).toHaveURL(/\/container\//);
}

test.describe("mobile shell", () => {
  test.use(pixel5);

  test.beforeEach(async ({ page }) => {
    await page.goto(BASE + "/");
  });

  test("opens a log view as a pushed screen: no tab bar, header pinned to the top, back goes home", async ({
    page,
  }) => {
    await expect(page.getByTestId("browse")).toBeVisible();
    await openContainer(page);

    // The tab bar steps aside so the bottom of the stream stays readable.
    await expect(page.getByTestId("browse")).toBeHidden();

    const header = page.getByTestId("scrollable-header");
    await expect.poll(async () => (await header.boundingBox())?.y).toBeLessThanOrEqual(1);

    // ...and stays there once the logs scroll under it.
    await page.mouse.wheel(0, 600);
    await expect.poll(async () => (await header.boundingBox())?.y).toBeLessThanOrEqual(1);

    await page.getByRole("button", { name: "Back" }).click();
    await expect(page).toHaveURL(/\/$/);
    await expect(page.getByTestId("browse")).toBeVisible();
  });

  test("back from a log view opened cold goes home instead of leaving the app", async ({ page }) => {
    await openContainer(page);
    const url = page.url();

    // A fresh tab on a shared link: there is no in-app history to go back to.
    const cold = await page.context().newPage();
    await cold.goto(url);
    await cold.getByRole("button", { name: "Back" }).click();

    await expect(cold).toHaveURL(BASE + "/");
    await expect(cold.getByTestId("browse")).toBeVisible();
  });

  test("hides the tab bar on every kind of log view, not only a single container", async ({ page }) => {
    await page.getByTestId("browse").click();
    await page.getByTestId("navigation").locator('a[href*="/host/"]').first().click();

    await expect(page).toHaveURL(/\/host\//);
    await expect(page.getByTestId("browse")).toBeHidden();
    await expect(page.getByRole("button", { name: "Back" })).toBeVisible();
  });

  test("opens the log actions menu as a bottom sheet that closes on a pick", async ({ page }) => {
    await openContainer(page);
    await page.getByTestId("log-actions").click();

    const sheet = page.locator(".popover-sheet:popover-open");
    await expect(sheet).toBeVisible();

    // Pinned to the bottom edge, nearly the full width, rather than hung off the trigger.
    const viewport = page.viewportSize()!;
    const box = (await sheet.boundingBox())!;
    expect(viewport.height - (box.y + box.height)).toBeLessThanOrEqual(16);
    expect(box.width).toBeGreaterThanOrEqual(viewport.width - 24);

    // The row's text also carries its keyboard shortcut, so match the row, not the label.
    await sheet.locator("a", { hasText: "Clear" }).click();
    await expect(sheet).toBeHidden();
  });
});

test.describe("desktop", () => {
  test("keeps the log actions menu anchored to its trigger", async ({ page }) => {
    await page.goto(BASE + "/");
    await openContainer(page);

    const trigger = page.getByTestId("log-actions");
    await trigger.hover();

    const panel = page.locator(".popover-panel:popover-open");
    await expect(panel).toBeVisible();
    await expect(panel).not.toHaveClass(/popover-sheet/);

    // Placed bottom-end: right edges line up and the panel starts just under the button.
    const t = (await trigger.boundingBox())!;
    const p = (await panel.boundingBox())!;
    expect(Math.abs(p.x + p.width - (t.x + t.width))).toBeLessThanOrEqual(2);
    expect(p.y - (t.y + t.height)).toBeGreaterThanOrEqual(0);
    expect(p.y - (t.y + t.height)).toBeLessThanOrEqual(8);
  });
});

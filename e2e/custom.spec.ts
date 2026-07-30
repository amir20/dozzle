import { test, expect } from "@playwright/test";

const origin = "http://custom_base:8080";
const base = "/foobarbase";

test.beforeEach(async ({ page }) => {
  await page.goto(origin + base);
});

test("has right title", async ({ page }) => {
  await expect(page).toHaveTitle(/.* - Dozzle/);
});

test("url should have custom base", async ({ page }) => {
  await expect(page).toHaveURL(/foobarbase/);
});

test("static assets resolve under the custom base", async ({ page }) => {
  const requested: string[] = [];
  const broken: string[] = [];

  const isAsset = (url: string) => new URL(url).pathname.includes("/assets/");

  page.on("request", (request) => {
    if (isAsset(request.url())) requested.push(new URL(request.url()).pathname);
  });
  page.on("response", (response) => {
    if (isAsset(response.url()) && response.status() >= 400) {
      broken.push(`${response.status()} ${response.url()}`);
    }
  });

  await page.reload();
  await expect(page).toHaveTitle(/.* - Dozzle/);
  // fonts are only fetched when a glyph needs them, so force the @font-face src to resolve.
  // swallow the rejection on a bad url so the assertions below report which path was wrong
  await page.evaluate(() =>
    document.fonts.load('400 1rem "JetBrains Mono"').then(
      () => {},
      () => {},
    ),
  );

  // the css lives at <base>/assets/, so a relative url() in it must resolve back under <base>
  expect(requested.some((path) => path.endsWith(".woff2"))).toBe(true);
  expect(requested.filter((path) => !path.startsWith(`${base}/assets/`))).toEqual([]);
  expect(broken).toEqual([]);
});

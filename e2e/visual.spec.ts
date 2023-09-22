import { test, expect } from "@playwright/test";

test.beforeEach(async ({ page }) => {
  await page.goto("http://dozzle:8080/");
});

test.describe("default", () => {
  test("homepage", async ({ page, isMobile }) => {
    if (isMobile) {
      await page.getByTestId("hamburger").click();
    }
    await expect(page.getByTestId("navigation")).toHaveScreenshot();
  });
});

test.describe("dark", () => {
  test.use({ colorScheme: "dark" });
  test("homepage", async ({ page, isMobile }) => {
    if (isMobile) {
      await page.getByTestId("hamburger").click();
    }
    await expect(page.getByTestId("navigation")).toHaveScreenshot();
  });
});

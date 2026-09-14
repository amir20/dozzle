import { defineConfig, devices } from "@playwright/test";

/**
 * Read environment variables from file.
 * https://github.com/motdotla/dotenv
 */
// require('dotenv').config();

/**
 * See https://playwright.dev/docs/test-configuration.
 */
export default defineConfig({
  testDir: "./e2e",
  /* Run tests in files in parallel */
  fullyParallel: true,
  /* Fail the build on CI if you accidentally left test.only in the source code. */
  forbidOnly: !!process.env.CI,
  /* Retry on CI only */
  retries: process.env.CI ? 2 : 0,
  /* Opt out of parallel tests on CI. */
  workers: process.env.CI ? 1 : undefined,
  /* Reporter to use. See https://playwright.dev/docs/test-reporters */
  reporter: "html",
  /* Shared settings for all the projects below. See https://playwright.dev/docs/api/class-testoptions. */
  use: {
    /* Base URL to use in actions like `await page.goto('/')`. */
    // baseURL: 'http://127.0.0.1:3000',

    /* Collect trace when retrying the failed test. See https://playwright.dev/docs/trace-viewer */
    trace: "on-first-retry",

    /* The e2e instances run server mode with no auth and an empty profile, which is
     * exactly when the setup wizard opens by itself. Mark it seen on every origin so it
     * never covers the UI under test or lands in a visual snapshot. The key matches
     * useProfileStorage("setupSeen"). */
    storageState: {
      cookies: [],
      origins: [
        "http://custom_base:8080",
        "http://dozzle:8080",
        "http://dozzle-with-agent:8080",
        "http://logs-viewer:8080",
        "http://remote:8080",
        "http://simple-auth:8080",
      ].map((origin) => ({ origin, localStorage: [{ name: "DOZZLE_SETUPSEEN", value: "true" }] })),
    },
  },

  /* Configure projects for major browsers */
  projects: [
    {
      name: "chromium",
      use: { ...devices["Desktop Chrome"] },
    },

    {
      name: "Mobile Chrome",
      use: { ...devices["Pixel 5"] },
      testMatch: "**/visual.spec.ts",
    },

    /* Test against branded browsers. */
    // {
    //   name: 'Microsoft Edge',
    //   use: { ...devices['Desktop Edge'], channel: 'msedge' },
    // },
    // {
    //   name: 'Google Chrome',
    //   use: { ..devices['Desktop Chrome'], channel: 'chrome' },
    // },
  ],
  // webServer: {
  //   command: "docker compose up",
  //   url: "http://127.0.0.1:7070",
  //   reuseExistingServer: !process.env.CI,
  // },
});

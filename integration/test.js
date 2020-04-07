const puppeteer = require("puppeteer");
const iPhoneX = puppeteer.devices["iPhone X"];
const iPadLandscape = puppeteer.devices["iPad landscape"];

const { BASE } = process.env;

describe("home page", () => {
  beforeEach(async () => {
    await page.goto(BASE, { waitUntil: "networkidle2" });
  });

  it("renders full page on desktop", async () => {
    const image = await page.screenshot({ fullPage: true });

    expect(image).toMatchImageSnapshot();
  });

  it("renders ipad viewport", async () => {
    await page.emulate(iPadLandscape);
    const image = await page.screenshot();

    expect(image).toMatchImageSnapshot();
  });

  it("renders iphone viewport", async () => {
    await page.emulate(iPhoneX);
    const image = await page.screenshot();

    expect(image).toMatchImageSnapshot();
  });

  describe("has menu visible", () => {
    beforeAll(async () => {
      await jestPuppeteer.resetBrowser();
      // await page.setViewport({ width: 1920, height: 1200 });
    });

    beforeEach(async () => {
      await page.goto(BASE, { waitUntil: "networkidle2" });
    });

    it("and shows one container with correct title", async () => {
      const menuTitle = await page.$eval("aside ul.menu-list li a", (e) => e.title);

      expect(menuTitle).toEqual("dozzle");
    });

    it("and menu is clickable", async () => {
      await page.click("aside ul.menu-list li a");

      const className = await page.$eval("aside ul.menu-list li a", (e) => e.className);

      expect(className).toContain("router-link-exact-active");
    });

    it("and when clicked shows logs", async () => {
      await page.click("aside ul.menu-list li a");

      await page.waitForSelector("ul.events li span.text");
      const text = await page.$eval("ul.events li:nth-child(2) span.text", (e) => e.textContent);

      expect(text).toContain("Dozzle version dev");
    });
  });
});

const puppeteer = require("puppeteer");
const { removeTimes } = require("../utils");
const iPhoneX = puppeteer.devices["iPhone X"];
const iPadLandscape = puppeteer.devices["iPad landscape"];

const { DEFAULT_URL: URL } = process.env;

describe("home page", () => {
  beforeEach(async () => {
    await page.goto(URL, { waitUntil: "networkidle2" });
  });

  it("renders full page on desktop", async () => {
    await removeTimes(page);
    const image = await page.screenshot({ fullPage: true });

    expect(image).toMatchImageSnapshot();
  });

  it("renders ipad viewport", async () => {
    await page.emulate(iPadLandscape);
    await removeTimes(page);
    const image = await page.screenshot();

    expect(image).toMatchImageSnapshot();
  });

  it("renders iphone viewport", async () => {
    await page.emulate(iPhoneX);
    await removeTimes(page);
    const image = await page.screenshot();

    expect(image).toMatchImageSnapshot();
  });

  it("displays iphone menu", async () => {
    await page.emulate(iPhoneX);
    await page.click("a.navbar-burger");

    const menuText = await page.$eval("aside ul.menu-list.is-hidden-mobile li a", (e) => e.textContent);
    expect(menuText.trim()).toEqual("dozzle");
  });

  describe("has menu visible", () => {
    beforeAll(async () => {
      await jestPuppeteer.resetBrowser();
    });

    beforeEach(async () => {
      await page.goto(URL, { waitUntil: "networkidle2" });
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
      const text = await page.$eval("ul.events li:nth-child(1) span.text", (e) => e.textContent);

      expect(text).toContain("Dozzle version dev");
    });
  });
});

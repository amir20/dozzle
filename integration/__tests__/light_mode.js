const puppeteer = require("puppeteer");
const { removeTimes } = require("../utils");
const iPhoneX = puppeteer.devices["iPhone X"];
const iPadLandscape = puppeteer.devices["iPad landscape"];

const { DEFAULT_URL: URL } = process.env;

describe("Dozzle with light mode", () => {
  beforeAll(async () => {
    await page.goto(URL + "/settings", { waitUntil: "networkidle2" });
    await page.$$eval("label.switch", (elements) => {
      elements.filter((e) => e.textContent.trim() === "Use light theme")[0].click();
    });
  });
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
});

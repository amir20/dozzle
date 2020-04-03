const puppeteer = require("puppeteer");

const BASE = process.env.BASE;

describe("home page", () => {
  let browser;

  beforeAll(async () => {
    browser = await puppeteer.launch({
      args: ["--no-sandbox", "--disable-setuid-sandbox"],
      executablePath: process.env.CHROME_EXE_PATH || "",
      defaultViewport: { width: 1920, height: 1200 }
    });
  });

  it("renders full page on desktop", async () => {
    const page = await browser.newPage();
    await page.goto(BASE, { waitUntil: "networkidle2" });

    const image = await page.screenshot({ fullPage: true });

    expect(image).toMatchImageSnapshot();
  });

  it("renders ipad viewport", async () => {
    const page = await browser.newPage();
    await page.goto(BASE, { waitUntil: "networkidle2" });
    await page.setViewport({ width: 1024, height: 768 });
    const image = await page.screenshot();

    expect(image).toMatchImageSnapshot();
  });

  it("renders iphone viewport", async () => {
    const page = await browser.newPage();
    await page.goto(BASE, { waitUntil: "networkidle2" });
    await page.setViewport({ width: 372, height: 812 });
    const image = await page.screenshot();

    expect(image).toMatchImageSnapshot();
  });

  afterAll(async () => {
    await browser.close();
  });
});

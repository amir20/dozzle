const { removeTimes } = require("../utils");
const { CUSTOM_URL: URL } = process.env;

describe("Dozzle with custom base", () => {
  beforeEach(async () => {
    await page.goto(URL, { waitUntil: "networkidle2" });
  });

  it("renders full page on desktop", async () => {
    await removeTimes(page);
    const image = await page.screenshot({ fullPage: true });

    expect(image).toMatchImageSnapshot();
  });

  it("and shows one container with correct title", async () => {
    await removeTimes(page);
    const menuTitle = await page.$eval("aside ul.menu-list li a", (e) => e.title);

    expect(menuTitle).toEqual("custom_base");
  });
});

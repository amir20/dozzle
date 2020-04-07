module.exports = {
  launch: {
    headless: true,
    defaultViewport: { width: 1920, height: 1200 },
    args: ["--no-sandbox", "--disable-setuid-sandbox"],
    executablePath: process.env.CHROME_EXE_PATH || "",
  },
  browserContext: "incognito",
};

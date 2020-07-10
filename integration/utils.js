async function removeTimes(page) {
  await page.evaluate(() => {
    (document.querySelectorAll("time") || []).forEach((el) => el.remove());
  });
}

module.exports = { removeTimes };

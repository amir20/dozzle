/// <reference types="cypress" />

context("Dozzle dark mode", { baseUrl: Cypress.env("DOZZLE_DEFAULT") }, () => {
  beforeEach(() => {
    cy.visit("/");
    cy.window().then((win) => win.document.documentElement.setAttribute("data-theme", "dark"));
  });

  it("home screen", () => {
    cy.get("li.running", { timeout: 10000 }).removeDates().replaceSkippedElements().matchImage();
  });
});

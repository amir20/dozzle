/// <reference types="cypress" />

context.skip("Dozzle dark mode", { baseUrl: Cypress.env("DOZZLE_DEFAULT") }, () => {
  beforeEach(() => {
    Cypress.on("window:before:load", (win) => {
      cy.stub(win, "matchMedia")
        .withArgs("(prefers-color-scheme: dark)")
        .returns({
          matches: true,
        })
        .as("dark-media-query");
    });

    cy.visit("/", {});
  });

  it("home screen", () => {
    cy.get("li.running", { timeout: 10000 }).removeDates().matchImage();
  });
});

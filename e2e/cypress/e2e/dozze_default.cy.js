/// <reference types="cypress" />

context("Dozzle default mode", { baseUrl: Cypress.env("DOZZLE_DEFAULT") }, () => {
  beforeEach(() => {
    cy.visit("/");
  });

  it("home screen", () => {
    cy.get("li.running", { timeout: 10000 }).removeDates().replaceSkippedElements().matchImage();
  });

  it("correct title is shown", () => {
    cy.title().should("eq", "1 containers - Dozzle");

    cy.get("li.running:first a").click();

    cy.title().should("include", "- Dozzle");
  });

  it("navigating to setting page works ", () => {
    cy.get("a[href='/settings']").click();

    cy.contains("About");
  });

  it("shortcut for fuzzy search works", () => {
    cy.get("body").type("{ctrl}k");

    cy.get("input[placeholder='Search containers (⌘ + k, ⌃k)']").should("be.visible");
  });
});

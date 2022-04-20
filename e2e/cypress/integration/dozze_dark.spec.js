/// <reference types="cypress" />

context("Dozzle default mode", { baseUrl: Cypress.env("DOZZLE_DEFAULT") }, () => {
  beforeEach(() => {
    cy.visit("/");
  });

  it("home screen", () => {
    cy.get("li.running", { timeout: 10000 }).removeDates().replaceSkippedElements().matchImageSnapshot();
  });

  it("correct title", () => {
    cy.title().should("eq", "1 containers - Dozzle");

    cy.get("li.running:first a").click();

    cy.title().should("include", "- Dozzle");
  });

  it("settings page", () => {
    cy.get("a[href='/settings']").click();

    cy.contains("About");
  });
});

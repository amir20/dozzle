/// <reference types="cypress" />

context("Dozzle custom base", { baseUrl: Cypress.env("DOZZLE_CUSTOM") }, () => {
  beforeEach(() => {
    cy.visit("/");
  });

  it("custom base should work", () => {
    cy.get("p.menu-label").should("contain", "Containers");
  });

  it("url should be custom", () => {
    cy.url().should("include", "foobarbase");
  });
});

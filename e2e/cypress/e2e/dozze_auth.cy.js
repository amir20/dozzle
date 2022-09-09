/// <reference types="cypress" />

context("Dozzle default mode", { baseUrl: Cypress.env("DOZZLE_AUTH") }, () => {
  beforeEach(() => {
    cy.visit("/");
  });

  it("login screen", () => {
    cy.get("input[name=username]").type("foo");
    cy.get("input[name=password]").type("bar");
    cy.get("button[type=submit]").click();
    cy.get("p.menu-label").should("contain", "Containers");
  });
});

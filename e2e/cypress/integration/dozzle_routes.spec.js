/// <reference types="cypress" />

context("Dozzle routes", { baseUrl: Cypress.env("DOZZLE_DEFAULT") }, () => {
  it("show", () => {
    cy.visit("/show?name=dozzle").url().should("include", "/container/");
  });
});

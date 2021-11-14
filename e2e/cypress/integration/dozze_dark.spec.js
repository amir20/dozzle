/// <reference types="cypress" />

context("Dozzle default mode", { baseUrl: Cypress.env("DOZZLE_DEFAULT") }, () => {
  beforeEach(() => {
    cy.visit("/");
  });

  it("home screen", () => {
    cy.get("li.running", { timeout: 10000 })
      .removeDates()
      .then(() => cy.matchImageSnapshot());
  });
});

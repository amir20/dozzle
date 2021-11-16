/// <reference types="cypress" />

context("Dozzle light mode", { baseUrl: Cypress.env("DOZZLE_DEFAULT") }, () => {
  before(() => {
    cy.visit("/settings");
    cy.contains("Use light theme").click();
  });
  beforeEach(() => {
    cy.visit("/");
  });

  it("home screen", () => {
    cy.get("li.running", { timeout: 10000 }).removeDates().matchImageSnapshot();
  });
});

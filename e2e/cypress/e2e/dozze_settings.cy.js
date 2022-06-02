/// <reference types="cypress" />

context("Dozzle settings mode", { baseUrl: Cypress.env("DOZZLE_DEFAULT") }, () => {
  beforeEach(() => {
    cy.visit("/version").clearLocalStorage().visit("/settings");
  });

  it("scrollbars", () => {
    cy.contains("Use smaller scrollbars").click();
    cy.get("html").should("have.class", "has-custom-scrollbars");
  });

  it("stopped containers", () => {
    cy.contains("Show stopped containers")
      .click()
      .then(() => {
        expect(JSON.parse(localStorage.getItem("DOZZLE_SETTINGS")).showAllContainers).to.be.true;
      });
  });
});

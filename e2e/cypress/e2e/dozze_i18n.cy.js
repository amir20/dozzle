/// <reference types="cypress" />

context("Dozzle es lang", { baseUrl: Cypress.env("DOZZLE_DEFAULT") }, () => {
  beforeEach(() => {
    cy.visit("/", {
      onBeforeLoad(win) {
        Object.defineProperty(win.navigator, "language", {
          value: "es_MX",
        });
      },
    });
  });

  it("should find contenedores", () => {
    cy.get("p.menu-label").should("contain", "Contenedores");
  });
});

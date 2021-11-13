/// <reference types="cypress" />

context("Dozzle dark mode", () => {
  beforeEach(() => {
    cy.visit("http://localhost:9090/");
  });

  it("home screen", () => {
    cy.get("li.running", { timeout: 10000 })
      .removeDates()
      .then(() => cy.matchImageSnapshot());
  });
});

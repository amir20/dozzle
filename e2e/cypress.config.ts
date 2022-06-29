import { defineConfig } from "cypress";

export default defineConfig({
  fixturesFolder: false,
  projectId: "8cua4m",

  e2e: {
    setupNodeEvents(on, config) {
      // implement node event listeners here
    },
  },
});

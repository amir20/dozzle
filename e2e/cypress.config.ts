import { defineConfig } from "cypress";
import { initPlugin } from "@frsource/cypress-plugin-visual-regression-diff/dist/plugins";

export default defineConfig({
  fixturesFolder: false,
  projectId: "8cua4m",

  e2e: {
    setupNodeEvents(on, config) {
      initPlugin(on, config);
    },
  },
});

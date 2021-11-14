/** @type {import('ts-jest/dist/types').InitialOptionsTsJest} */
module.exports = {
  preset: "ts-jest",
  testEnvironment: "jsdom",
  testPathIgnorePatterns: ["node_modules", "<rootDir>/integration/", "<rootDir>/e2e/"],
  transform: {
    "^.+\\.vue$": "@vue/vue3-jest",
  },
};

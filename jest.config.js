/** @type {import('ts-jest/dist/types').InitialOptionsTsJest} */
module.exports = {
  preset: "ts-jest",
  testEnvironment: "node",
  testPathIgnorePatterns: ["node_modules", "<rootDir>/integration/"],
  transform: {
    "^.+\\.vue$": "@vue/vue3-jest",
  },
};

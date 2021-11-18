import type { Config } from "@jest/types";

const config: Config.InitialOptions = {
  preset: "ts-jest",
  testEnvironment: "jsdom",
  testPathIgnorePatterns: ["node_modules", "<rootDir>/integration/", "<rootDir>/e2e/"],
  transform: {
    "^.+\\.vue$": "@vue/vue3-jest",
  },
  moduleNameMapper: {
    "@/(.*)": ["<rootDir>/assets/$1"],
  },
};

export default config;

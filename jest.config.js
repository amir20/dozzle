module.exports = {
  clearMocks: true,
  testEnvironment: "jsdom",
  moduleFileExtensions: ["js", "json", "vue"],
  coveragePathIgnorePatterns: ["node_modules"],
  testPathIgnorePatterns: ["node_modules", "<rootDir>/integration/"],
  transformIgnorePatterns: ["node_modules"],
  watchPathIgnorePatterns: ["<rootDir>/node_modules/"],
  snapshotSerializers: ["jest-serializer-vue"],
  transform: {
    ".*\\.vue$": "vue-jest",
    "^.+\\.js$": "babel-jest",
  },
};

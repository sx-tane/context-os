/** @type {import('jest').Config} */
module.exports = {
  testEnvironment: "node",
  transform: {
    "^.+\\.tsx?$": [
      "@swc/jest",
      {
        jsc: {
          parser: { syntax: "typescript" },
          target: "es2020",
        },
        module: { type: "commonjs" },
      },
    ],
  },
  moduleNameMapper: {
    "^\\$lib/(.*)$": "<rootDir>/src/lib/$1",
  },
  testMatch: ["<rootDir>/src/**/__tests__/**/*.test.ts"],
  collectCoverageFrom: ["src/lib/**/*.ts", "!src/lib/generated/**"],
};

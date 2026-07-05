/** @type {import("eslint").Linter.Config} */
module.exports = {
  extends: [
    "next/core-web-vitals",
    "next/typescript",
  ],
  rules: {
    "no-console": "error",
    "@typescript-eslint/no-unused-vars": ["error", { argsIgnorePattern: "^_" }],
    "@typescript-eslint/no-explicit-any": "error",
  },
};

/** @type {import('jest').Config} */
module.exports = {
  preset: 'ts-jest',
  testEnvironment: 'node',
  roots: [
    '<rootDir>/packages',
    '<rootDir>/apps/customer/lib',
    '<rootDir>/apps/admin/lib',
    '<rootDir>/apps/driver/lib',
    '<rootDir>/apps/merchant/lib',
  ],
  testMatch: ['**/__tests__/**/*.test.ts'],
  moduleNameMapper: {
    '^@speedplus/types$': '<rootDir>/packages/types/src/index.ts',
    '^@speedplus/utils$': '<rootDir>/packages/utils/src/index.ts',
    '^@speedplus/api-client$': '<rootDir>/packages/api-client/src/index.ts',
    '^@speedplus/ui$': '<rootDir>/packages/ui/src/index.ts',
    '^@speedplus/config$': '<rootDir>/packages/config',
    '^@/(.*)$': '<rootDir>/apps/customer/$1',
  },
  transform: {
    '^.+\\.tsx?$': ['ts-jest', { tsconfig: '<rootDir>/tsconfig.test.json' }],
  },
  // zustand persist warns about missing localStorage in Node — expected, not a failure
  silent: true,
};

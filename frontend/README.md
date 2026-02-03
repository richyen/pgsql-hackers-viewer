# Frontend Development Guide

## Setup

Install dependencies:
```bash
npm install
```

Initialize git hooks (for pre-commit linting):
```bash
npm run prepare
```

## Development

Start the dev server:
```bash
npm start
```

Run linting:
```bash
npm run lint
```

Auto-fix linting issues:
```bash
npm run lint:fix
```

## Pre-commit Hooks

This project uses Husky and lint-staged to automatically lint TypeScript/React files before each commit.

When you commit, the following happens automatically:
1. All staged `.ts` and `.tsx` files are linted
2. Auto-fixable issues are corrected
3. Fixed files are added back to the commit
4. If there are unfixable errors, the commit is blocked

To bypass hooks (not recommended):
```bash
git commit --no-verify
```

## Common Linting Rules

- No unused variables or imports
- Consistent code formatting
- React best practices
- TypeScript strict checks

## Troubleshooting

**Pre-commit hook not running?**
```bash
npm run prepare
chmod +x .husky/pre-commit
```

**Linting errors in Docker?**
The container auto-reloads on file changes. Check the container logs for any TypeScript/ESLint errors.

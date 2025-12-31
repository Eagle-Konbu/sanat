# AI Development Guidelines

This document outlines the core development guidelines for this project. All AI coding assistants should follow these guidelines.

## Commit Messages

All commit messages must be written in **English** and follow the [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/#summary) specification.

### Format

```
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

### Types

- **feat**: A new feature
- **fix**: A bug fix
- **docs**: Documentation only changes
- **style**: Changes that do not affect the meaning of the code (white-space, formatting, etc)
- **refactor**: A code change that neither fixes a bug nor adds a feature
- **perf**: A code change that improves performance
- **test**: Adding missing tests or correcting existing tests
- **build**: Changes that affect the build system or external dependencies
- **ci**: Changes to CI configuration files and scripts
- **chore**: Other changes that don't modify src or test files

### Examples

```
feat: add user authentication
fix: resolve memory leak in data processor
docs: update API documentation
refactor: simplify error handling logic
```

### Breaking Changes

Breaking changes must be indicated by:
- Adding `!` after the type/scope: `feat!: remove deprecated API`
- Including `BREAKING CHANGE:` in the footer

## Code Comments

### Principles

1. **Minimize comments**: Strive to write self-explanatory code
2. **Comment intent, not implementation**: Only add comments when the intent is unclear
3. **Self-documenting code is paramount**: Use descriptive variable names, function names, and clear structure

### When to Write Comments

- ✅ When explaining **why** something is done a certain way
- ✅ When documenting complex algorithms or business logic
- ✅ When clarifying non-obvious behavior or edge cases
- ❌ Not for explaining **what** the code does (the code should be clear enough)
- ❌ Not for redundant descriptions of obvious operations

### Examples

**Bad (unnecessary comment):**
```typescript
// Increment counter by 1
counter++;
```

**Good (explains intent):**
```typescript
// Skip validation for admin users to allow bulk imports
if (user.isAdmin) {
  return processWithoutValidation(data);
}
```

## Philosophy

> "Code should be written for humans to read, and only incidentally for machines to execute."

Focus on clarity, simplicity, and maintainability. Let the code speak for itself.
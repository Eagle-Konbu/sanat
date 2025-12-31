# Skill: Commit Messages

## Purpose
Ensure consistent, parsable, and meaningful commit history using Conventional Commits specification.

## Inputs
- Code changes to be committed
- Context of the change (feature, fix, refactor, etc.)

## Outputs
- Commit message in English following Conventional Commits format

## Procedure
1. Determine change type: feat, fix, docs, style, refactor, perf, test, build, ci, chore
2. Identify scope (optional): component or module affected
3. Write concise description in imperative mood
4. Add body for context (optional)
5. Add footer for breaking changes or issue references (optional)

## Constraints
- **Language**: English only
- **Format**: `<type>[optional scope]: <description>`
- **Description**: Imperative mood, lowercase, no period at end
- **Breaking changes**: Add `!` after type or `BREAKING CHANGE:` in footer
- **Length**: Description ≤ 72 characters

## Definition of Done
- [ ] Message follows `<type>[scope]: <description>` format
- [ ] Type is one of allowed types
- [ ] Description is in English, imperative mood
- [ ] Breaking changes are properly indicated (if applicable)

## Examples

### Good Examples
```
feat: add user authentication
fix: resolve memory leak in data processor
docs: update API documentation
refactor: simplify error handling logic
feat!: remove deprecated API
```

### Bad Examples
```
Added new feature (not imperative mood)
fix: Fixed a bug. (capital F, period at end)
update (missing type format)
feat: 新機能を追加 (not in English)
```

## References
- [Conventional Commits Specification](https://www.conventionalcommits.org/en/v1.0.0/)

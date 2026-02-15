# AI Development Instructions

## Code Guidelines

### Philosophy

- **Clarity > Cleverness**: Prefer obvious over clever
- **Simplicity > Optimization**: Optimize only when necessary
- **Consistency > Personal preference**: Follow project conventions
- **Maintainability > Brevity**: Favor readable over terse

Write code as if explaining to another developer. Variable and function names should clearly convey purpose. Control flow should be straightforward. No unnecessary complexity or premature optimization.

### Comments

Minimize comments by writing self-explanatory code. Add comments only when intent is unclear from code alone.

1. **First**: Make code self-explanatory (descriptive names, extract complex logic into well-named functions, simplify control flow)
2. **Only if step 1 fails**: Add comment explaining **why**, not **what**
3. **Review**: Can the comment be eliminated by refactoring?

Rules:
- Comment **intent**: Why this approach was chosen
- Comment **non-obvious behavior**: Edge cases, performance considerations
- Comment **complex algorithms**: Business logic requiring context
- Do not comment **implementation**: What code does (should be obvious)
- Do not comment **obvious operations**: Counter increments, variable assignments

## Commit Messages

Follow the [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/) specification. See [commit-messages skill](skills/commit-messages.md) for full details.

- **Format**: `<type>[optional scope]: <description>`
- **Language**: English only
- **Description**: Imperative mood, lowercase, no period at end, ≤ 72 characters
- **Types**: feat, fix, docs, style, refactor, perf, test, build, ci, chore

## Execution Protocol

1. **Before writing code**: Review these guidelines
2. **During implementation**: Apply code guidelines
3. **Before completing task**: Verify code quality

### Workflow

1. **Design** — Choose simplest solution that meets requirements
2. **Implementation** — Write self-explanatory code with descriptive names
3. **Documentation** — Add comments only if intent remains unclear after refactoring
4. **Commit** — Generate commit message following Conventional Commits

## Code Review

- Check code against code guidelines (clarity, simplicity, consistency)
- Verify comments explain "why" not "what"; suggest refactoring to eliminate unnecessary comments
- Validate commit messages against Conventional Commits spec

## Multi-file Changes

- Apply guidelines consistently across all files
- Use scope in commit message to indicate affected component

## Task Commands

This project uses [Task](https://taskfile.dev/) for build automation. **Prefer `task` commands over direct `go` commands when available.**

### Available Tasks
- `task test` - Run all tests
- `task test:coverage` - Run tests with coverage report
- `task lint` - Run linter (golangci-lint)
- `task lint:fix` - Run linter with auto-fix
- `task build` - Build the project
- `task fmt` - Format code
- `task vet` - Run go vet
- `task tidy` - Tidy go modules
- `task check` - Run all checks (fmt, vet, lint, test)
- `task ci` - Run CI checks
- `task clean` - Clean build artifacts

### Command Preferences
1. **Prefer** `task` commands when they exist (e.g., `task test` over `go test`)
2. Use `task lint` instead of `golangci-lint run`
3. Use `task build` instead of `go build`
4. Use `task check` before committing to ensure code quality
5. **Fallback**: If `task` is unavailable or a specific task doesn't exist, use direct commands

## References

- Skills: `.ai/skills/`
- [Conventional Commits Specification](https://www.conventionalcommits.org/en/v1.0.0/)

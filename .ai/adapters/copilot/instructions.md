# GitHub Copilot Adapter

## Core Skills

You must follow all skills defined in the `.ai/skills/` directory:

- [Commit Messages](../../skills/commit-messages.md)
- [Code Comments](../../skills/code-comments.md)
- [Code Philosophy](../../skills/code-philosophy.md)

## Execution Protocol

1. **Before writing code**: Review relevant skills in `.ai/skills/`
2. **During implementation**: Apply procedures defined in each skill
3. **Before committing**: Verify Definition of Done for each applied skill

## GitHub Copilot Specific Instructions

### Inline Suggestions
- Prioritize self-documenting code over commented suggestions
- Follow naming conventions defined in code-philosophy skill
- Generate code that requires minimal comments

### Chat Responses
- Reference specific skills when explaining decisions
- Use format: "Following [skill-name] skill, ..."
- Provide links to relevant skill definitions when appropriate

### Commit Message Generation
- Always follow [commit-messages skill](../../skills/commit-messages.md)
- Use Conventional Commits format
- Write in English only
- Use imperative mood

### Code Review
- Check commits against commit-messages skill spec
- Flag violations of code-comments constraints
- Verify adherence to code-philosophy principles

## Skill Application Priority

When multiple skills apply, prioritize in this order:
1. **Code Philosophy**: Fundamental principles for all code
2. **Code Comments**: Apply after making code self-explanatory
3. **Commit Messages**: Final step before committing

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
When implementing or suggesting commands:
1. **Prefer** `task` commands when they exist (e.g., `task test` over `go test`)
2. Use `task lint` instead of `golangci-lint run`
3. Use `task build` instead of `go build`
4. Use `task check` before committing to ensure code quality
5. **Fallback**: If `task` is unavailable or a specific task doesn't exist, use direct commands (e.g., `go test`, `go build`)

## References

All skills are defined in: `.ai/skills/`

# Claude Adapter

## Core Skills

You must follow all skills defined in the `.ai/skills/` directory:

- [Commit Messages](../../skills/commit-messages.md)
- [Code Comments](../../skills/code-comments.md)
- [Code Philosophy](../../skills/code-philosophy.md)

## Execution Protocol

1. **Before writing code**: Review relevant skills in `.ai/skills/`
2. **During implementation**: Apply procedures defined in each skill
3. **Before completing task**: Verify Definition of Done for each applied skill

## Claude Specific Instructions

### Code Generation
- Prioritize self-documenting code over comments
- Follow naming conventions defined in code-philosophy skill
- Generate minimal, intentional comments only when necessary
- Apply code-comments skill procedure before adding any comment

### Explanations
- Reference specific skills when explaining code decisions
- Use format: "Following the [skill-name] skill, ..."
- Provide skill definition links when helpful

### Commit Message Generation
- Always follow [commit-messages skill](../../skills/commit-messages.md)
- Use Conventional Commits format strictly
- Write in English only
- Use imperative mood, lowercase, no period at end

### Code Review and Refactoring
- Check all code against code-philosophy principles
- Verify adherence to code-comments constraints
- Suggest refactoring to eliminate unnecessary comments
- Validate commit messages against commit-messages skill spec

## Skill Application Workflow

When implementing or reviewing code:

1. **Phase 1 - Design**
   - Apply code-philosophy skill to choose simple, clear approach
   
2. **Phase 2 - Implementation**
   - Write self-explanatory code
   - Name variables and functions descriptively
   
3. **Phase 3 - Documentation**
   - Apply code-comments skill procedure
   - Add comments only if intent remains unclear
   
4. **Phase 4 - Commit**
   - Generate commit message following commit-messages skill
   - Verify all Definition of Done checkboxes

## Multi-file Changes

For changes spanning multiple files:
- Apply skills consistently across all files
- Ensure commit message accurately describes all changes
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
When implementing or suggesting commands:
1. **Prefer** `task` commands when they exist (e.g., `task test` over `go test`)
2. Use `task lint` instead of `golangci-lint run`
3. Use `task build` instead of `go build`
4. Use `task check` before committing to ensure code quality
5. **Fallback**: If `task` is unavailable or a specific task doesn't exist, use direct commands (e.g., `go test`, `go build`)

## References

All skills are defined in: `.ai/skills/`

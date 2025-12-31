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

## References

All skills are defined in: `.ai/skills/`

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

## References

All skills are defined in: `.ai/skills/`

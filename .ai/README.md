# AI Development Guidelines

This directory contains AI development guidelines for this project.

## Structure

```
.ai/
  instructions.md    # Shared guidelines for all AI assistants
  skills/            # Reusable skill definitions
  README.md          # This file
```

### Instructions

[instructions.md](instructions.md) contains shared guidelines applied by all AI assistants:
- Code philosophy and comment policy
- Commit message format
- Execution protocol and code review
- Task commands

### Skills

Skills define **procedures, constraints, and contracts**:

- [**commit-messages.md**](skills/commit-messages.md) - Conventional Commits specification

### Entry Points

- `CLAUDE.md` (project root) → references `.ai/instructions.md`
- `.github/copilot-instructions.md` → references `.ai/instructions.md`

## Adding New Skills

1. Create a new file in `.ai/skills/` following the skill template
2. Include all required sections: Purpose, Inputs, Outputs, Procedure, Constraints, Definition of Done
3. Update `instructions.md` to reference the new skill
4. Update this README to list the new skill

## References

- [Conventional Commits](https://www.conventionalcommits.org/)
- Shared instructions: `.ai/instructions.md`
- Skills: `.ai/skills/`

# AI Development Guidelines

This directory contains skills-based AI development guidelines for this project.

## Overview

This project uses a **skills-based approach** for AI-assisted development. Guidelines are defined as reusable, LLM-agnostic skills that can be applied by any AI coding assistant through specific adapters.

## Structure

```
.ai/
  skills/          # LLM-agnostic skill definitions
  adapters/        # LLM-specific instructions
  README.md        # This file
```

### Skills

Skills define **procedures, constraints, and contracts** that are independent of any specific LLM:

- [**commit-messages.md**](skills/commit-messages.md) - Conventional Commits specification
- [**code-comments.md**](skills/code-comments.md) - Comment minimization principles
- [**code-philosophy.md**](skills/code-philosophy.md) - Code clarity and maintainability

Each skill follows a standard structure:
- **Purpose**: What problem it solves
- **Inputs**: What information is needed
- **Outputs**: What it produces
- **Procedure**: Step-by-step execution
- **Constraints**: Rules to follow
- **Definition of Done**: Completion criteria

### Adapters

Adapters provide LLM-specific instructions that reference skills:

- [**adapters/copilot/**](adapters/copilot/instructions.md) - GitHub Copilot instructions
- [**adapters/claude/**](adapters/claude/instructions.md) - Claude instructions

## Usage

### For GitHub Copilot
GitHub Copilot automatically loads instructions from `.github/copilot-instructions.md`, which references the Copilot adapter.

### For Claude
Claude reads instructions from `CLAUDE.md` in the project root, which references the Claude adapter.

### For Other LLMs
Create a new adapter in `.ai/adapters/<llm-name>/` that references the skills in `.ai/skills/`.

## Design Principles

1. **Single Source of Truth**: Skills are the authoritative definitions
2. **LLM Agnostic**: Skills can be used by any AI coding assistant
3. **Separation of Concerns**: LLM-specific behavior lives in adapters
4. **Reusability**: Skills can be referenced by multiple adapters
5. **Clarity**: Each skill has clear inputs, outputs, and procedures

## Migration from AI_GUIDELINES.md

This skills-based structure replaces the previous `AI_GUIDELINES.md` monolithic approach:

- **Commit Messages** section → `skills/commit-messages.md`
- **Code Comments** section → `skills/code-comments.md`
- **Philosophy** section → `skills/code-philosophy.md`

The new structure provides:
- Better separation of concerns
- Easier maintenance and updates
- Support for multiple AI assistants
- Clear contracts and procedures

## Adding New Skills

To add a new skill:

1. Create a new file in `.ai/skills/` following the skill template
2. Include all required sections: Purpose, Inputs, Outputs, Procedure, Constraints, Definition of Done
3. Update adapter instructions to reference the new skill
4. Update this README to list the new skill

## Skill Template

```markdown
# Skill: <Skill Name>

## Purpose
<What problem this skill solves>

## Inputs
- Input 1: <description>

## Outputs
- Output 1: <description>

## Procedure
1. Step 1
2. Step 2

## Constraints
- Constraint 1

## Definition of Done
- [ ] Condition 1
```

## References

- [Conventional Commits](https://www.conventionalcommits.org/)
- Project skills: `.ai/skills/`
- Adapter instructions: `.ai/adapters/`

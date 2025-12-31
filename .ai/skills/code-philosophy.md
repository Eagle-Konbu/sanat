# Skill: Code Philosophy

## Purpose
Write code optimized for human understanding, maintainability, and clarity.

## Inputs
- Requirements or specifications
- Existing codebase context

## Outputs
- Clear, simple, maintainable code

## Procedure
1. **Understand**: Clarify requirements before coding
2. **Design**: Choose simplest solution that meets requirements
3. **Implement**: Write code as if explaining to another developer
4. **Review**: Read code from reader's perspective
5. **Refine**: Simplify without losing clarity

## Constraints
- **Clarity > Cleverness**: Prefer obvious over clever
- **Simplicity > Optimization**: Optimize only when necessary
- **Consistency > Personal preference**: Follow project conventions
- **Maintainability > Brevity**: Favor readable over terse

## Definition of Done
- [ ] Code can be understood without external documentation
- [ ] Variable and function names clearly convey purpose
- [ ] Control flow is straightforward
- [ ] No unnecessary complexity or premature optimization
- [ ] Code follows project conventions

## Principles

### 1. Write for Humans First
Code is read far more often than it is written. Optimize for readability.

**Bad:**
```go
func p(s []int) int {
    r := 0
    for _, v := range s {
        r += v
    }
    return r
}
```

**Good:**
```go
func sum(numbers []int) int {
    total := 0
    for _, number := range numbers {
        total += number
    }
    return total
}
```

### 2. Prefer Simplicity
The simplest solution that works is usually the best solution.

**Bad (premature optimization):**
```go
// Using complex bit manipulation for no measurable benefit
func isEven(n int) bool {
    return (n & 1) == 0
}
```

**Good:**
```go
func isEven(n int) bool {
    return n%2 == 0
}
```

### 3. Be Consistent
Follow existing project conventions even if you have different preferences.

### 4. Avoid Cleverness
If it requires a comment to explain how it works, it's probably too clever.

## Core Philosophy
> "Code should be written for humans to read, and only incidentally for machines to execute."

Focus on clarity, simplicity, and maintainability. Let the code speak for itself.

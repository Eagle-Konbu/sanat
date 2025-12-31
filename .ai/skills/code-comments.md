# Skill: Code Comments

## Purpose
Minimize comments by writing self-explanatory code. Add comments only when intent is unclear from code alone.

## Inputs
- Code to be documented
- Context requiring clarification

## Outputs
- Self-documenting code with minimal, intentional comments

## Procedure
1. **First**: Attempt to make code self-explanatory
   - Use descriptive variable names
   - Extract complex logic into well-named functions
   - Simplify control flow
2. **Only if step 1 fails**: Add comment explaining **why**, not **what**
3. **Review**: Can comment be eliminated by refactoring?

## Constraints
- ✅ Comment **intent**: Why this approach was chosen
- ✅ Comment **non-obvious behavior**: Edge cases, performance considerations
- ✅ Comment **complex algorithms**: Business logic requiring context
- ❌ Do not comment **implementation**: What code does (should be obvious)
- ❌ Do not comment **obvious operations**: Counter increments, variable assignments

## Definition of Done
- [ ] Code is self-explanatory without comments
- [ ] Any remaining comments explain **why**, not **what**
- [ ] No redundant or obvious comments exist
- [ ] Comments are in English

## Examples

### Bad (unnecessary comment)
```go
// Increment counter by 1
counter++

// Create a new user
user := &User{}

// Loop through items
for _, item := range items {
    // Process item
    processItem(item)
}
```

### Good (explains intent)
```go
// Skip validation for admin users to allow bulk imports
if user.IsAdmin {
    return processWithoutValidation(data)
}

// Use binary search here instead of linear scan because
// the dataset can contain millions of records
index := binarySearch(sortedData, target)

// Cache result for 5 minutes to reduce database load
// during peak hours (identified in profiling session)
cache.Set(key, result, 5*time.Minute)
```

## Philosophy
> "Code should be written for humans to read, and only incidentally for machines to execute."

Focus on clarity through good naming and structure, not comments.

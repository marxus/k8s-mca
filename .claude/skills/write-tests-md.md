# Write TESTS.md Documentation

Generate TESTS.md documentation files for Go test suites.

## Guidelines

### File Location
- Place TESTS.md in the same directory as the corresponding test file (e.g., `pkg/certs/TESTS.md` for `pkg/certs/certs_test.go`)

### Structure

```markdown
# [Package Name] Tests

This document describes the test suite for the [package name] package (`pkg/[package]`).

## Overview

[1-2 paragraph description of what the package does and what aspects the tests validate]

## Test Coverage

[Organize tests by function/feature if there are logical groupings]

- **TestName** - one-liner description of what the test validates
- **AnotherTest** - one-liner description of what the test validates
```

### Content Guidelines

**Include:**
- Title with package name
- Overview explaining package purpose and test focus
- Bullet-point list of tests with one-liner descriptions
- Group tests by function or feature when it makes sense

**Exclude:**
- "How to Run" sections
- "Dependencies" sections
- "Test Results" sections
- Detailed step-by-step test explanations
- Code examples

### Description Style

- Use bullet points with bold test names
- Keep descriptions to one line
- Start descriptions with "validates that..." or "validates..."
- Focus on what is being tested, not how
- Be specific about the expected behavior

**Good examples:**
- `validates that Authorization header is removed from requests before forwarding while preserving other headers`
- `validates that certificate generation succeeds with nil/empty DNS names and IP addresses without errors`

**Bad examples:**
- `tests the authorization header` (too vague)
- `validates that: 1) header is removed, 2) request is forwarded, 3) response is received` (too detailed, use multiple test cases)

### Grouping Tests

When a package has many tests, organize them by the function or feature they test:

```markdown
## Test Coverage

### FunctionName
- **TestFunction_Case1** - description
- **TestFunction_Case2** - description

### AnotherFunction
- **TestAnotherFunction_Case1** - description
```

## Example

See `pkg/certs/TESTS.md`, `pkg/inject/TESTS.md`, or `pkg/proxy/TESTS.md` for complete examples.

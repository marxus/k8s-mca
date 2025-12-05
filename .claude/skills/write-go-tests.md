# Write Go Tests

Write idiomatic Go tests following standard library patterns with table-driven test organization.

## Core Principles

1. **Self-documenting tests** - Test names and structure should explain what's being tested
2. **Table-driven when beneficial** - Use tables for tests with similar structure but different inputs
3. **Minimal comments** - Only document non-obvious behavior or test setup
4. **Clear error messages** - Use "got X, want Y" format in assertions
5. **Package comments only** - No doc comments on individual test functions

## Package-Level Comments

Every `_test.go` file should have a minimal package comment describing what aspects are tested:

```go
// Package proxy tests the HTTP reverse proxy server behavior.
package proxy

// Helper function tests for volume mount configuration.
package inject

// Webhook configuration and patching tests.
package serve
```

**Format:**
- Simple one-line comment explaining test scope
- No elaborate documentation - tests are code, not prose
- Located before `package` declaration

## When to Use Table-Driven Tests

Use table-driven tests when you have:

### ✅ Good Candidates

1. **Multiple similar test cases** - Same test logic, different inputs/outputs
   ```go
   // Good: Testing same function with different inputs
   TestAddVolumeMount - (empty mounts, matching mount, non-matching mount)
   TestWriteCACertificate - (success case, error case)
   ```

2. **Success and error paths** - Testing both happy path and error conditions
   ```go
   // Good: Same operation, different outcomes
   TestViaCLI - (valid YAML, invalid YAML)
   TestWriteNamespaceFile - (success, file not found)
   ```

3. **Variations of behavior** - Same function, different configurations
   ```go
   // Good: Different configurations of env vars
   TestAddEnvVars - (adds new, updates existing, preserves others)
   ```

### ❌ Poor Candidates

1. **Unique test logic** - Each test validates completely different behavior
2. **Complex setup** - Test setup is more complex than the test itself
3. **Few cases** - Only 1-2 test cases (just write separate functions)
4. **Different assertions** - Each case needs fundamentally different validations

## Table-Driven Test Structure

```go
func TestFunctionName(t *testing.T) {
    tests := []struct {
        name       string           // descriptive test case name
        input      InputType        // test inputs
        wantOutput ExpectedType     // expected outputs
        wantErr    bool            // whether error is expected
        errMsg     string          // expected error message (optional)
    }{
        {
            name:    "descriptive case name",
            input:   someInput,
            wantOutput: expectedOutput,
            wantErr: false,
        },
        {
            name:    "error case description",
            input:   invalidInput,
            wantErr: true,
            errMsg:  "expected error substring",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := FunctionName(tt.input)

            if tt.wantErr {
                assert.Error(t, err)
                assert.Contains(t, err.Error(), tt.errMsg)
            } else {
                require.NoError(t, err)
                assert.Equal(t, tt.wantOutput, result)
            }
        })
    }
}
```

## Field Naming Conventions

Use clear, descriptive field names in test structs:

### Input Fields
- Use descriptive names: `podYAML`, `volumeMounts`, `initialEnv`
- Not generic: `input`, `data`, `param`

### Expected Output Fields
- Prefix with `want`: `wantErr`, `wantResult`, `wantValue`, `wantLen`
- Or use descriptive names: `expectedName`, `shouldMatch`

### Setup/Configuration Fields
- Use function closures: `setupFS func() afero.Fs`
- Or descriptive names: `createReadOnlyFS`, `mockClient`

**Good examples:**
```go
tests := []struct {
    name           string
    volumeMounts   []corev1.VolumeMount
    wantResult     bool
    wantName       string
}{
    // ...
}
```

**Avoid:**
```go
tests := []struct {
    name   string
    input  interface{}  // too generic
    output interface{}  // too generic
    err    bool
}{
    // ...
}
```

## Test Case Names

Use descriptive names that explain the scenario:

**Good:**
- `"successfully writes certificate"`
- `"fails on read-only filesystem"`
- `"updates existing serviceaccount mount"`
- `"handles empty volume mounts"`
- `"preserves other env vars"`

**Avoid:**
- `"test 1"`, `"case 2"` - not descriptive
- `"works"`, `"fails"` - too vague
- `"TestCase1"` - redundant prefix

## Common Patterns

### Success + Error Cases

```go
func TestWriteFile(t *testing.T) {
    tests := []struct {
        name    string
        setupFS func() afero.Fs
        wantErr bool
        errMsg  string
    }{
        {
            name: "successfully writes file",
            setupFS: func() afero.Fs {
                fs := afero.NewMemMapFs()
                fs.MkdirAll("/path", 0755)
                return fs
            },
            wantErr: false,
        },
        {
            name: "fails on read-only filesystem",
            setupFS: func() afero.Fs {
                return afero.NewReadOnlyFs(afero.NewMemMapFs())
            },
            wantErr: true,
            errMsg:  "failed to write",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            fs := tt.setupFS()
            err := WriteFile(fs, "/path/file")

            if tt.wantErr {
                assert.Error(t, err)
                assert.Contains(t, err.Error(), tt.errMsg)
            } else {
                require.NoError(t, err)
            }
        })
    }
}
```

### Multiple Variations

```go
func TestAddEnvVars(t *testing.T) {
    tests := []struct {
        name        string
        initialEnv  []corev1.EnvVar
        wantEnvLen  int
        wantEnvVars map[string]string
    }{
        {
            name:       "adds new env vars to empty container",
            initialEnv: []corev1.EnvVar{},
            wantEnvLen: 2,
            wantEnvVars: map[string]string{
                "KUBERNETES_SERVICE_HOST": "127.0.0.1",
                "KUBERNETES_SERVICE_PORT": "6443",
            },
        },
        {
            name: "updates existing env vars",
            initialEnv: []corev1.EnvVar{
                {Name: "KUBERNETES_SERVICE_HOST", Value: "old-value"},
            },
            wantEnvLen: 2,
            wantEnvVars: map[string]string{
                "KUBERNETES_SERVICE_HOST": "127.0.0.1",
                "KUBERNETES_SERVICE_PORT": "6443",
            },
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            container := &corev1.Container{
                Name: "app",
                Env:  tt.initialEnv,
            }

            addEnvVars(container)

            require.Len(t, container.Env, tt.wantEnvLen)

            envMap := make(map[string]string)
            for _, env := range container.Env {
                envMap[env.Name] = env.Value
            }

            for key, value := range tt.wantEnvVars {
                assert.Equal(t, value, envMap[key])
            }
        })
    }
}
```

### Inline Function Closures for Complex Setup

```go
func TestServer_HandleMutate(t *testing.T) {
    tests := []struct {
        name           string
        requestBody    []byte
        wantStatusCode int
    }{
        {
            name: "valid admission request",
            requestBody: func() []byte {
                pod := corev1.Pod{
                    ObjectMeta: metav1.ObjectMeta{
                        Name: "test-pod",
                    },
                }
                podJSON, _ := json.Marshal(pod)
                admissionReview := admissionv1.AdmissionReview{
                    Request: &admissionv1.AdmissionRequest{
                        Object: runtime.RawExtension{Raw: podJSON},
                    },
                }
                body, _ := json.Marshal(admissionReview)
                return body
            }(),
            wantStatusCode: http.StatusOK,
        },
    }
    // ...
}
```

## When NOT to Use Table-Driven Tests

Don't force table-driven tests when:

1. **Single test case** - Just write a normal test function
2. **Completely different test logic** - Each test validates different aspects
3. **Complex per-case setup** - Table becomes harder to read than separate functions
4. **Integration tests** - Often have unique setup/teardown per test

**Example of when to keep separate:**
```go
// Good: Each test validates fundamentally different behavior
func TestInjectProxy_AddsProxyInitContainer(t *testing.T) { /* ... */ }
func TestInjectProxy_UpdatesVolumeMountAndAddsEnvVars(t *testing.T) { /* ... */ }
func TestInjectProxy_AddsRequiredVolume(t *testing.T) { /* ... */ }
```

## Test Helpers

Document helper functions that are used across multiple tests:

```go
// createTestPod returns a basic pod for testing with service account volume mount.
func createTestPod() corev1.Pod {
    return corev1.Pod{
        Spec: corev1.PodSpec{
            Containers: []corev1.Container{{
                Name: "app",
                VolumeMounts: []corev1.VolumeMount{{
                    MountPath: "/var/run/secrets/kubernetes.io/serviceaccount",
                }},
            }},
        },
    }
}
```

## Validation Checklist

- [ ] Package comment describes what the test file validates
- [ ] No doc comments on individual test functions (test names are self-documenting)
- [ ] Table-driven tests use descriptive field names (`wantErr`, not `err`)
- [ ] Test case names are descriptive (`"handles empty input"`, not `"test1"`)
- [ ] Error cases check error message content with `assert.Contains`
- [ ] Related test cases are grouped in table-driven tests
- [ ] Tests use `t.Run` for subtests to improve output
- [ ] Test logic is simple and easy to understand
- [ ] Helper functions have doc comments explaining their purpose

## Examples from Codebase

Study these files for reference:
- `pkg/inject/inject_test.go` - Helper function testing with tables
- `pkg/serve/proxy_test.go` - File operation tests with success/error cases
- `pkg/serve/webhook_test.go` - Simple table-driven test
- `pkg/webhook/server_test.go` - HTTP handler tests with tables

## References

- [Go Wiki: Test Comments](https://go.dev/wiki/TestComments)
- [Go Wiki: Table Driven Tests](https://go.dev/wiki/TableDrivenTests)
- [testing package](https://pkg.go.dev/testing)

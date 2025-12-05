# Write Go Documentation

Generate Go documentation comments following official Go standards for godoc and pkg.go.dev.

## Core Principles

1. **Every exported (capitalized) name must have a doc comment**
2. Doc comments appear directly before declarations with **no blank lines**
3. First sentence is crucial - appears in search results and package listings
4. Use complete sentences starting with the declared name
5. Explain **what** the code does, not **how** it works

## Documentation by Type

### Package Comments

```go
// Package [name] provides [brief description].
// [Optional: Additional details about the package purpose and usage]
//
// [Optional: More detailed explanation, examples, or important notes]
package name
```

**Guidelines:**
- Start with "Package [name]"
- Make first sentence concise and complete
- Include important usage information
- Only one package comment per package (even in multi-file packages)

**Example:**
```go
// Package proxy provides an HTTP reverse proxy that intercepts Kubernetes API requests.
// It removes Authorization headers and forwards requests to configured cluster endpoints.
//
// The proxy supports multiple target clusters through a map of reverse proxy instances,
// allowing for multi-cluster API request routing.
package proxy
```

### Function/Method Comments

```go
// FunctionName [verb phrase describing what it does or returns].
// [Optional: Additional details about behavior, parameters, return values]
//
// [Optional: Error conditions, special cases, or examples]
func FunctionName(param Type) (ReturnType, error)
```

**Guidelines:**
- Start with the function name
- For functions returning values: describe what they return
- For functions with side effects: describe what they do
- Use "reports whether" for boolean returns
- Reference parameters and results naturally (no special syntax needed)
- Document error conditions

**Examples:**
```go
// NewServer creates a new proxy server with the given TLS certificate and reverse proxies.
// The reverseProxies map must contain at least an "in-cluster" key.
//
// Returns an error if the certificate is invalid.
func NewServer(tlsCert tls.Certificate, reverseProxies map[string]*httputil.ReverseProxy) (*Server, error)

// Start starts the proxy server and blocks until it exits.
// It returns an error if the server fails to start or encounters a fatal error.
func (s *Server) Start() error

// IsValid reports whether the configuration is valid.
func (c *Config) IsValid() bool
```

### Type Comments

```go
// TypeName [describes what each instance represents or provides].
// [Optional: Concurrency safety, zero value behavior, usage guidelines]
type TypeName struct
```

**Guidelines:**
- Describe what each instance represents
- Document concurrency safety if relevant
- Explain zero value meaning if non-obvious
- For structs with exported fields, document field purposes

**Example:**
```go
// Server represents an HTTPS proxy server that intercepts Kubernetes API calls.
// It is safe for concurrent use by multiple goroutines.
//
// The zero value is not valid; use NewServer to create instances.
type Server struct {
    tlsCert        tls.Certificate
    reverseProxies map[string]*httputil.ReverseProxy
}
```

### Constant/Variable Comments

```go
// ConstName [describes the constant's purpose or value].
const ConstName = value

// VarName [describes the variable's purpose].
var VarName Type
```

**Grouped constants:**
```go
// Common HTTP status codes used by the proxy.
const (
    // StatusOK indicates successful request processing.
    StatusOK = 200
    // StatusBadGateway indicates upstream server failure.
    StatusBadGateway = 502
)
```

## Formatting Syntax

### Paragraphs

Separate paragraphs with blank comment lines:
```go
// First paragraph explaining the main concept.
//
// Second paragraph with additional details.
```

### Headings

Use `#` followed by space for headings:
```go
// # Configuration
//
// The server accepts the following configuration options...
//
// # Performance
//
// The proxy is optimized for high throughput...
```

### Links

**Doc links** (to other Go symbols):
```go
// See [NewServer] for initialization.
// Use [Server.Start] to begin serving requests.
// Import the [net/http] package for HTTP types.
```

**URLs:**
```go
// For more information, see the guide:
// [Kubernetes API]: https://kubernetes.io/docs/reference/
```

### Lists

**Bullet lists:**
```go
// The proxy performs the following operations:
//   - Removes Authorization headers
//   - Forwards requests to target clusters
//   - Logs all API calls
```

**Numbered lists:**
```go
// To configure the proxy:
//  1. Generate TLS certificates
//  2. Create reverse proxy instances
//  3. Initialize the server
```

### Code Blocks

Indent with spaces or tabs (no fence markers):
```go
// Example usage:
//
//     cert := generateCert()
//     proxies := map[string]*httputil.ReverseProxy{
//         "in-cluster": reverseProxy,
//     }
//     server := NewServer(cert, proxies)
//     server.Start()
```

### Deprecation

```go
// Deprecated: Use NewServerV2 instead. This function will be removed in v2.0.0.
func OldServer() *Server
```

## What to Document

### Always Document:
- Exported types, functions, methods, constants, variables
- What the code does (not how)
- Parameters and return values (naturally, not in special syntax)
- Error conditions and return values
- Special cases and edge conditions

### Include When Relevant:
- Concurrency safety guarantees
- Zero value behavior if non-obvious
- Performance characteristics (time/space complexity)
- Resource management (cleanup, closing, etc.)
- Invariants and preconditions
- Examples for complex APIs

### Don't Document:
- Implementation details
- Unexported (lowercase) names
- Obvious behavior
- How the code works internally

## Common Patterns

### Constructor Functions
```go
// NewThing creates and initializes a new Thing with the given options.
// Returns an error if validation fails.
func NewThing(opts Options) (*Thing, error)
```

### Builder Methods
```go
// WithTimeout sets the timeout duration and returns the modified Config.
// The timeout must be positive or this method panics.
func (c *Config) WithTimeout(d time.Duration) *Config
```

### Error Handling
```go
// Process processes the data and returns the result.
// It returns ErrInvalidInput if the data is malformed,
// or ErrTimeout if processing exceeds the deadline.
func Process(data []byte) (Result, error)
```

### Interfaces
```go
// Handler handles Kubernetes API requests.
// Implementations must be safe for concurrent use.
type Handler interface {
    // Handle processes the request and writes the response.
    Handle(w http.ResponseWriter, r *http.Request)
}
```

## Tools

### View Documentation Locally
```bash
# View package documentation
go doc github.com/marxus/k8s-mca/pkg/proxy

# View specific symbol
go doc github.com/marxus/k8s-mca/pkg/proxy.Server

# Run local documentation server
go install golang.org/x/pkgsite/cmd/pkgsite@latest
pkgsite
```

### Format Documentation
```bash
# gofmt automatically formats doc comments
gofmt -w .
```

## Validation Checklist

- [ ] All exported names have doc comments
- [ ] Comments start with the declared name
- [ ] First sentence is complete and concise
- [ ] No blank lines between comment and declaration
- [ ] Lists and code blocks are properly indented
- [ ] Cross-references use `[Name]` syntax
- [ ] Deprecations use "Deprecated:" prefix
- [ ] Comments explain what, not how
- [ ] Special cases and errors are documented
- [ ] Concurrency safety is documented where relevant

## Examples from Standard Library

Study these for reference:
- `net/http` - HTTP server and client
- `context` - Context package
- `io` - I/O interfaces
- `encoding/json` - JSON encoding

## References

- [Go Doc Comments](https://tip.golang.org/doc/comment)
- [Effective Go - Commentary](https://go.dev/doc/effective_go#commentary)
- [pkg.go.dev](https://pkg.go.dev)

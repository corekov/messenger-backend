---
name: go-api
description: |
  HTTP API development patterns for Go. Use when creating endpoints, handlers, middleware, or error responses. Triggers: API design, HTTP handlers, error handling, functional options, option merging, response formatting.
metadata:
  mcpmarket-version: 1.0.0
---
# Go API Development

## When This Skill Applies

- Creating HTTP endpoints
- Implementing handlers and middleware
- Designing error responses
- Working with API configuration

## Principles

### 1. Option Merging Behavior

Configuration provides baseline values; runtime options override them.

```go
// Model configuration
type ModelConfig struct {
    Temperature float64            `toml:"temperature"`
    MaxTokens   int                `toml:"max_tokens"`
    Options     map[string]any     `toml:"options"`
}

// Runtime call
type CallOptions struct {
    Temperature *float64           // nil = use config value
    MaxTokens   *int
    Extra       map[string]any
}

func (m *Model) Call(ctx context.Context, input string, opts CallOptions) (string, error) {
    // Merge: runtime overrides config
    temp := m.config.Temperature
    if opts.Temperature != nil {
        temp = *opts.Temperature
    }

    maxTokens := m.config.MaxTokens
    if opts.MaxTokens != nil {
        maxTokens = *opts.MaxTokens
    }

    // Build final options
    finalOpts := make(map[string]any)
    for k, v := range m.config.Options {
        finalOpts[k] = v
    }
    for k, v := range opts.Extra {
        finalOpts[k] = v  // Runtime overrides
    }

    // Execute with merged options
}
```

### 2. Error Type Categorization

Define domain-specific error types with clear categorization.

```go
type ErrorType string

const (
    ErrorTypeValidation ErrorType = "validation"
    ErrorTypeNotFound   ErrorType = "not_found"
    ErrorTypeConflict   ErrorType = "conflict"
    ErrorTypeInternal   ErrorType = "internal"
    ErrorTypeExternal   ErrorType = "external"
)

type APIError struct {
    Type      ErrorType `json:"type"`
    Code      string    `json:"code"`
    Message   string    `json:"message"`
    Details   any       `json:"details,omitempty"`
    RequestID string    `json:"request_id,omitempty"`
    cause     error     // Private, for Unwrap
}

func (e *APIError) Error() string {
    return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *APIError) Unwrap() error {
    return e.cause
}

// HTTP status mapping
func (e *APIError) HTTPStatus() int {
    switch e.Type {
    case ErrorTypeValidation:
        return http.StatusBadRequest
    case ErrorTypeNotFound:
        return http.StatusNotFound
    case ErrorTypeConflict:
        return http.StatusConflict
    case ErrorTypeExternal:
        return http.StatusBadGateway
    default:
        return http.StatusInternalServerError
    }
}
```

### 3. Functional Options Pattern

Use functional options for extensible, self-documenting APIs.

```go
type HandlerOption func(*Handler)

func WithLogger(l *slog.Logger) HandlerOption {
    return func(h *Handler) {
        h.logger = l
    }
}

func WithTimeout(d time.Duration) HandlerOption {
    return func(h *Handler) {
        h.timeout = d
    }
}

func WithMiddleware(mw ...Middleware) HandlerOption {
    return func(h *Handler) {
        h.middleware = append(h.middleware, mw...)
    }
}

func NewHandler(service Service, opts ...HandlerOption) *Handler {
    h := &Handler{
        service: service,
        logger:  slog.Default(),
        timeout: 30 * time.Second,
    }
    for _, opt := range opts {
        opt(h)
    }
    return h
}
```

## Patterns

### Request/Response Types

```go
// Request with validation
type CreateUserRequest struct {
    Email    string `json:"email"`
    Name     string `json:"name"`
    Password string `json:"password"`
}

func (r *CreateUserRequest) Validate() error {
    var errs []error
    if r.Email == "" {
        errs = append(errs, errors.New("email required"))
    }
    if len(r.Password) < 8 {
        errs = append(errs, errors.New("password must be at least 8 characters"))
    }
    return errors.Join(errs...)
}

// Response with consistent structure
type Response[T any] struct {
    Data      T      `json:"data,omitempty"`
    Error     *Error `json:"error,omitempty"`
    RequestID string `json:"request_id"`
}
```

### Handler Structure

```go
func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    requestID := middleware.GetRequestID(ctx)

    // 1. Parse request
    var req CreateUserRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        h.writeError(w, &APIError{
            Type:      ErrorTypeValidation,
            Code:      "INVALID_JSON",
            Message:   "Invalid request body",
            RequestID: requestID,
        })
        return
    }

    // 2. Validate
    if err := req.Validate(); err != nil {
        h.writeError(w, &APIError{
            Type:      ErrorTypeValidation,
            Code:      "VALIDATION_FAILED",
            Message:   err.Error(),
            RequestID: requestID,
        })
        return
    }

    // 3. Execute business logic
    user, err := h.service.CreateUser(ctx, req)
    if err != nil {
        h.handleServiceError(w, err, requestID)
        return
    }

    // 4. Return response
    h.writeJSON(w, http.StatusCreated, Response[User]{
        Data:      user,
        RequestID: requestID,
    })
}
```

### Middleware Pattern

```go
type Middleware func(http.Handler) http.Handler

func RequestIDMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        requestID := r.Header.Get("X-Request-ID")
        if requestID == "" {
            requestID = uuid.New().String()
        }

        ctx := context.WithValue(r.Context(), requestIDKey, requestID)
        w.Header().Set("X-Request-ID", requestID)

        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

func LoggingMiddleware(logger *slog.Logger) Middleware {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            start := time.Now()

            // Wrap response writer to capture status
            wrapped := &responseWriter{ResponseWriter: w, status: 200}

            next.ServeHTTP(wrapped, r)

            logger.Info("request completed",
                "method", r.Method,
                "path", r.URL.Path,
                "status", wrapped.status,
                "duration", time.Since(start),
                "request_id", GetRequestID(r.Context()),
            )
        })
    }
}
```

### Error Response (RFC 7807)

```go
type ProblemDetail struct {
    Type     string `json:"type"`
    Title    string `json:"title"`
    Status   int    `json:"status"`
    Detail   string `json:"detail,omitempty"`
    Instance string `json:"instance,omitempty"`
}

func (h *Handler) writeProblem(w http.ResponseWriter, status int, title, detail string) {
    problem := ProblemDetail{
        Type:   fmt.Sprintf("/errors/%d", status),
        Title:  title,
        Status: status,
        Detail: detail,
    }

    w.Header().Set("Content-Type", "application/problem+json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(problem)
}
```

## Anti-Patterns

### Panicking in Handlers

```go
// Bad: Panic crashes the server
func (h *Handler) GetUser(w http.ResponseWriter, r *http.Request) {
    user := h.service.MustGetUser(id)  // Panics on error
}
```

```go
// Good: Return error response
func (h *Handler) GetUser(w http.ResponseWriter, r *http.Request) {
    user, err := h.service.GetUser(ctx, id)
    if err != nil {
        h.handleError(w, err)
        return
    }
    h.writeJSON(w, http.StatusOK, user)
}
```

### Inconsistent Error Formats

```go
// Bad: Different error shapes
{"error": "not found"}
{"message": "validation failed", "code": 400}
{"err": {"type": "internal"}}
```

```go
// Good: Consistent error shape
{"error": {"type": "not_found", "code": "USER_NOT_FOUND", "message": "User not found"}}
{"error": {"type": "validation", "code": "INVALID_EMAIL", "message": "Invalid email format"}}
```

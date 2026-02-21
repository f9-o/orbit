// Package errs provides structured, user-friendly errors with machine-parseable codes.
package errs

import (
	"errors"
	"fmt"
)

// ErrorCode is a machine-parseable error identifier.
type ErrorCode string

const (
	// General
	ErrUnknown    ErrorCode = "ERR-000"
	ErrInternal   ErrorCode = "ERR-001"
	ErrConfig     ErrorCode = "ERR-002"
	ErrValidation ErrorCode = "ERR-003"

	// Node errors
	ErrNodeNotFound    ErrorCode = "ERR-NODE-001"
	ErrNodeConnect     ErrorCode = "ERR-NODE-002"
	ErrNodeTimeout     ErrorCode = "ERR-NODE-003"
	ErrNodeKeyMismatch ErrorCode = "ERR-NODE-004"
	ErrNodeUnknownKey  ErrorCode = "ERR-NODE-005"

	// Service errors
	ErrServiceNotFound   ErrorCode = "ERR-SVC-001"
	ErrServiceStart      ErrorCode = "ERR-SVC-002"
	ErrServiceStop       ErrorCode = "ERR-SVC-003"
	ErrServiceHealthFail ErrorCode = "ERR-SVC-004"
	ErrServiceRollback   ErrorCode = "ERR-SVC-005"

	// Docker errors
	ErrDockerConnect ErrorCode = "ERR-DOCKER-001"
	ErrDockerPull    ErrorCode = "ERR-DOCKER-002"
	ErrDockerRun     ErrorCode = "ERR-DOCKER-003"
	ErrDockerRemove  ErrorCode = "ERR-DOCKER-004"
	ErrDockerInspect ErrorCode = "ERR-DOCKER-005"

	// SSL errors
	ErrSSLIssueFail    ErrorCode = "ERR-SSL-001"
	ErrSSLRenewFail    ErrorCode = "ERR-SSL-002"
	ErrSSLCertNotFound ErrorCode = "ERR-SSL-003"

	// State errors
	ErrStateRead  ErrorCode = "ERR-STATE-001"
	ErrStateWrite ErrorCode = "ERR-STATE-002"
)

// OrbitError is the standard structured error type used across all Orbit packages.
type OrbitError struct {
	Code   ErrorCode // Machine-parseable error code
	Op     string    // Operation chain, e.g., "deploy.rolling.healthcheck"
	Node   string    // Resource identifier (node name, service name, etc.)
	Cause  error     // Wrapped upstream error
	Advice string    // Human-readable remediation hint
}

func (e *OrbitError) Error() string {
	if e.Node != "" {
		return fmt.Sprintf("[%s] %s (%s): %v", e.Code, e.Op, e.Node, e.Cause)
	}
	return fmt.Sprintf("[%s] %s: %v", e.Code, e.Op, e.Cause)
}

func (e *OrbitError) Unwrap() error {
	return e.Cause
}

// UserMessage returns the formatted user-facing error message with remediation advice.
func (e *OrbitError) UserMessage() string {
	msg := fmt.Sprintf("%s: %s", e.Code, e.Op)
	if e.Node != "" {
		msg += fmt.Sprintf(" (resource: %s)", e.Node)
	}
	if e.Advice != "" {
		msg += fmt.Sprintf("\n  → %s", e.Advice)
	}
	return msg
}

// New creates a new OrbitError.
func New(code ErrorCode, op string, cause error) *OrbitError {
	return &OrbitError{Code: code, Op: op, Cause: cause}
}

// Newf creates a new OrbitError with a formatted message as the cause.
func Newf(code ErrorCode, op, format string, args ...any) *OrbitError {
	return &OrbitError{Code: code, Op: op, Cause: fmt.Errorf(format, args...)}
}

// WithNode sets the node/resource identifier on an OrbitError.
func (e *OrbitError) WithNode(node string) *OrbitError {
	e.Node = node
	return e
}

// WithAdvice sets the human-readable remediation hint on an OrbitError.
func (e *OrbitError) WithAdvice(advice string) *OrbitError {
	e.Advice = advice
	return e
}

// Wrap wraps an existing error as an OrbitError at a new operation boundary.
func Wrap(err error, code ErrorCode, op string) *OrbitError {
	if err == nil {
		return nil
	}
	return &OrbitError{Code: code, Op: op, Cause: err}
}

// IsCode reports whether err is an OrbitError with the given code.
func IsCode(err error, code ErrorCode) bool {
	var oe *OrbitError
	if errors.As(err, &oe) {
		return oe.Code == code
	}
	return false
}

// AsOrbit extracts the *OrbitError from err, or returns nil.
func AsOrbit(err error) *OrbitError {
	var oe *OrbitError
	if errors.As(err, &oe) {
		return oe
	}
	return nil
}

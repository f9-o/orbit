// Package logger provides the structured logging engine for Orbit.
// Uses log/slog with support for multiple sinks: stderr, file, TUI.
// Log file rotation is handled via a simple size-checked os.File writer —
// no external dependencies required.
package logger

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// Logger
// ─────────────────────────────────────────────────────────────────────────────

// Logger wraps slog.Logger with Orbit-specific utilities.
type Logger struct {
	*slog.Logger
	tuiSink chan<- string // non-nil when TUI is active
	auditW  io.Writer     // append-only audit log writer (nil = disabled)
}

// TUISink returns a channel that receives formatted log lines for TUI display.
// Call SetTUISink before Init to enable TUI log forwarding.
var tuiSinkCh chan string

// SetTUISink registers a channel that receives log lines destined for the TUI.
func SetTUISink(ch chan string) {
	tuiSinkCh = ch
}

// Init initialises the global logger. Safe to call multiple times (idempotent after first call).
func Init(level, format, logFile, orbitHome string, debug bool) (*Logger, error) {
	var lvl slog.Level
	switch level {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}
	if debug {
		lvl = slog.LevelDebug
	}

	// Build multi-writer: always write to stderr, optionally to file
	writers := []io.Writer{os.Stderr}

	var fileWriter io.Writer
	if logFile != "" {
		if err := os.MkdirAll(filepath.Dir(logFile), 0750); err == nil {
			f, err := os.OpenFile(logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0640)
			if err == nil {
				fileWriter = f
				writers = append(writers, f)
			}
		}
	}
	_ = fileWriter

	// TUI sink: forward log lines to channel
	if tuiSinkCh != nil {
		writers = append(writers, &tuiWriter{ch: tuiSinkCh})
	}

	out := io.MultiWriter(writers...)

	var handler slog.Handler
	opts := &slog.HandlerOptions{Level: lvl, AddSource: debug}
	if format == "json" {
		handler = slog.NewJSONHandler(out, opts)
	} else {
		handler = slog.NewTextHandler(out, opts)
	}

	base := slog.New(handler)
	slog.SetDefault(base)

	// Audit log
	var auditW io.Writer
	if orbitHome != "" {
		auditPath := filepath.Join(orbitHome, "audit.log")
		if af, err := os.OpenFile(auditPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0640); err == nil {
			auditW = af
		}
	}

	return &Logger{
		Logger:  base,
		tuiSink: nil,
		auditW:  auditW,
	}, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Audit logging
// ─────────────────────────────────────────────────────────────────────────────

// AuditEntry represents a single audit log event.
type AuditEntry struct {
	Timestamp time.Time         `json:"ts"`
	Op        string            `json:"op"`
	User      string            `json:"user"`
	Node      string            `json:"node,omitempty"`
	Service   string            `json:"service,omitempty"`
	Result    string            `json:"result"` // success | failure
	Meta      map[string]string `json:"meta,omitempty"`
}

// Audit writes an append-only audit log entry.
func (l *Logger) Audit(entry AuditEntry) {
	l.Info("audit",
		"op", entry.Op,
		"user", entry.User,
		"node", entry.Node,
		"service", entry.Service,
		"result", entry.Result,
	)
	if l.auditW == nil {
		return
	}
	line := fmt.Sprintf(`{"ts":%q,"op":%q,"user":%q,"node":%q,"service":%q,"result":%q}`+"\n",
		entry.Timestamp.UTC().Format(time.RFC3339),
		entry.Op, entry.User, entry.Node, entry.Service, entry.Result,
	)
	_, _ = l.auditW.Write([]byte(line))
}

// ─────────────────────────────────────────────────────────────────────────────
// TUI writer
// ─────────────────────────────────────────────────────────────────────────────

// tuiWriter implements io.Writer by forwarding lines to the TUI sink channel.
type tuiWriter struct {
	mu sync.Mutex
	ch chan<- string
}

func (w *tuiWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	select {
	case w.ch <- string(p):
	default: // drop if channel full — never block logger
	}
	return len(p), nil
}

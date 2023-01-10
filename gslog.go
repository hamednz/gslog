package gslog

import (
	"bytes"
	"cloud.google.com/go/logging"
	"encoding/json"
	"golang.org/x/exp/slog"
	"io"
	"log"
	"sync"
)

type GoogleHandler struct {
	w       *bytes.Buffer
	logger  *logging.Logger
	handler slog.Handler
	mu      sync.Mutex
}

func (g *GoogleHandler) Enabled(level slog.Level) bool {
	return g.handler.Enabled(level)
}

func (g *GoogleHandler) Handle(r slog.Record) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	err := g.handler.Handle(r)
	if err != nil {
		return err
	}

	b, err := io.ReadAll(g.w)
	if err != nil {
		return err
	}
	g.w.Reset()

	var payload map[string]interface{}
	err = json.Unmarshal(b, &payload)
	if err != nil {
		log.Printf("json.Unmarshal: %v", err)
		return err
	}

	defer g.logger.Flush() // Ensure the entry is written.

	var level logging.Severity

	switch r.Level {
	case slog.LevelDebug:
		level = logging.Debug
	case slog.LevelInfo:
		level = logging.Info
	case slog.LevelWarn:
		level = logging.Warning
	case slog.LevelError:
		level = logging.Error
	default:
		level = logging.Default
	}

	g.logger.Log(logging.Entry{
		Timestamp: r.Time,
		Payload:   payload,
		Severity:  level,
	})

	return nil
}

func (g *GoogleHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	g.handler = g.handler.WithAttrs(attrs)
	return g
}

func (g *GoogleHandler) WithGroup(name string) slog.Handler {
	g.handler = g.handler.WithGroup(name)
	return g
}

type GCPConfig struct {
	logger *logging.Logger
	w      io.ReadWriter
	opts   *slog.HandlerOptions
}

func NewGCPHandler(conf GCPConfig) *GoogleHandler {
	var b []byte

	buf := bytes.NewBuffer(b)
	mw := io.MultiWriter(conf.w, buf)

	var h slog.Handler
	h = slog.NewJSONHandler(mw)

	if conf.opts != nil {
		h = (conf.opts).NewJSONHandler(mw)
	}

	return &GoogleHandler{
		logger:  conf.logger,
		handler: h,
		w:       buf,
	}
}

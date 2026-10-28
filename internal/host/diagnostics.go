package host

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const DefaultDiagnosticRingBufferCapacity = 100

type DiagnosticEntry struct {
	Timestamp         string `json:"timestamp"`
	PluginID          string `json:"plugin_id"`
	Method            string `json:"method"`
	ErrorClass        string `json:"error_class"`
	File              string `json:"file"`
	Line              int    `json:"line"`
	StackTrace        string `json:"stack_trace"`
	DurationMS        int64  `json:"duration_ms"`
	SQLiteLockWaitMS  int64  `json:"sqlite_lock_wait_ms"`
	SQLiteWriteHoldMS int64  `json:"sqlite_write_hold_ms"`
}

type DiagnosticBundle struct {
	SchemaVersion      string            `json:"schema_version"`
	SchoolURN          string            `json:"school_urn"`
	GeneratedAt        string            `json:"generated_at"`
	GeneratedByStaffID string            `json:"generated_by_staff_id"`
	TelemetryMode      string            `json:"telemetry_mode"`
	RingBufferCapacity int               `json:"ring_buffer_capacity"`
	EntryCount         int               `json:"entry_count"`
	PIIScrubbed        bool              `json:"pii_scrubbed"`
	Entries            []DiagnosticEntry `json:"entries"`
	LogArchive         string            `json:"log_archive"`
}

type sqlLockTracker struct {
	lockWaitDuration  time.Duration
	writeHoldStart    time.Time
	writeHoldDuration time.Duration
}

func (t *sqlLockTracker) finishHold() {
	if t != nil && !t.writeHoldStart.IsZero() {
		t.writeHoldDuration += time.Since(t.writeHoldStart)
		t.writeHoldStart = time.Time{}
	}
}

var (
	luaChunkLineRegex = regexp.MustCompile(`src/init\.lua:(\d+):`)
	luaFrameFuncRegex = regexp.MustCompile(`in function '([a-zA-Z0-9_]+)'`)
	ulnTokenRegex     = regexp.MustCompile(`\b[0-9]{10}\b`)
	entityIDRegex     = regexp.MustCompile(`(?i)\b(pup|per|stf|grd|hh|sfg|pnote|wlog|usr|req|wrk|sess)_[a-z0-9_-]+\b`)
	upnTokenRegex     = regexp.MustCompile(`\b[A-Z][0-9]{12}[A-Z0-9]?\b`)
)

func scrubDiagnosticText(raw string, extraSecrets []string) string {
	s := raw
	for _, secret := range extraSecrets {
		trimmed := strings.TrimSpace(secret)
		if len(trimmed) >= 3 {
			s = strings.ReplaceAll(s, trimmed, "[REDACTED]")
		}
	}
	s = piiScrubPattern.ReplaceAllString(s, "[REDACTED]")
	s = entityIDRegex.ReplaceAllString(s, "[REDACTED]")
	s = upnTokenRegex.ReplaceAllString(s, "[REDACTED]")
	s = ulnTokenRegex.ReplaceAllString(s, "[REDACTED]")
	return s
}

func collectArgSecretStrings(args map[string]any) []string {
	if len(args) == 0 {
		return nil
	}
	var out []string
	var walk func(v any)
	walk = func(v any) {
		switch val := v.(type) {
		case string:
			if strings.TrimSpace(val) != "" {
				out = append(out, val)
			}
		case map[string]any:
			for _, item := range val {
				walk(item)
			}
		case []any:
			for _, item := range val {
				walk(item)
			}
		case []string:
			for _, item := range val {
				if strings.TrimSpace(item) != "" {
					out = append(out, item)
				}
			}
		}
	}
	walk(args)
	return out
}

func extractSanitizedLuaStackTrace(
	rawErr error,
	method string,
	safeErrorClass string,
	extraSecrets []string,
) (string, int, string) {
	if rawErr == nil {
		trace := fmt.Sprintf("[host]:0: in method '%s' (error_class=%s)", method, safeErrorClass)
		return "[host]", 0, trace
	}

	normalized := strings.ReplaceAll(rawErr.Error(), "<string>:", "src/init.lua:")
	normalized = scrubDiagnosticText(normalized, extraSecrets)

	file := "[host]"
	line := 0

	if m := luaChunkLineRegex.FindStringSubmatch(normalized); len(m) == 2 {
		file = "src/init.lua"
		if parsedLine, err := strconv.Atoi(m[1]); err == nil && parsedLine > 0 {
			line = parsedLine
		}
	}

	var sanitizedFrames []string
	lines := strings.Split(normalized, "\n")
	for _, rawLine := range lines {
		trimmed := strings.TrimSpace(rawLine)
		if trimmed == "" || trimmed == "stack traceback:" {
			continue
		}
		if strings.HasPrefix(trimmed, "src/init.lua:") {
			if m := luaChunkLineRegex.FindStringSubmatch(trimmed); len(m) == 2 {
				fnName := method
				if fm := luaFrameFuncRegex.FindStringSubmatch(trimmed); len(fm) == 2 {
					fnName = fm[1]
				}
				frame := fmt.Sprintf("src/init.lua:%s: in function '%s'", m[1], fnName)
				if len(sanitizedFrames) == 0 || sanitizedFrames[len(sanitizedFrames)-1] != frame {
					sanitizedFrames = append(sanitizedFrames, frame)
				}
			}
		}
	}

	if len(sanitizedFrames) == 0 {
		if file == "src/init.lua" && line > 0 {
			sanitizedFrames = append(sanitizedFrames, fmt.Sprintf("src/init.lua:%d: in function '%s'", line, method))
		} else {
			sanitizedFrames = append(sanitizedFrames, fmt.Sprintf("[host]:0: in method '%s'", method))
		}
	}

	trace := fmt.Sprintf("%s (error_class=%s)", strings.Join(sanitizedFrames, " <- "), safeErrorClass)
	trace = scrubDiagnosticText(trace, extraSecrets)
	return file, line, trace
}

func formatDiagnosticLogLine(entry DiagnosticEntry) string {
	return fmt.Sprintf(
		"[%s] plugin=%s method=%s error_class=%s file=%s line=%d duration_ms=%d sqlite_lock_wait_ms=%d sqlite_write_hold_ms=%d stack_trace=%q",
		entry.Timestamp,
		entry.PluginID,
		entry.Method,
		entry.ErrorClass,
		entry.File,
		entry.Line,
		entry.DurationMS,
		entry.SQLiteLockWaitMS,
		entry.SQLiteWriteHoldMS,
		entry.StackTrace,
	)
}

func (h *Host) recordDiagnosticDetailed(
	pluginID string,
	method string,
	errClass string,
	rawErr error,
	duration time.Duration,
	lockTracker *sqlLockTracker,
	extraSecrets []string,
) {
	h.mu.Lock()
	defer h.mu.Unlock()

	safeClass := scrubDiagnosticText(errClass, extraSecrets)
	safeMethod := scrubDiagnosticText(method, extraSecrets)
	file, line, stackTrace := extractSanitizedLuaStackTrace(rawErr, safeMethod, safeClass, extraSecrets)

	var lockWaitMS, writeHoldMS int64
	if lockTracker != nil {
		lockTracker.finishHold()
		lockWaitMS = lockTracker.lockWaitDuration.Milliseconds()
		writeHoldMS = lockTracker.writeHoldDuration.Milliseconds()
	}

	entry := DiagnosticEntry{
		Timestamp:         time.Now().UTC().Format(time.RFC3339Nano),
		PluginID:          pluginID,
		Method:            safeMethod,
		ErrorClass:        safeClass,
		File:              file,
		Line:              line,
		StackTrace:        stackTrace,
		DurationMS:        duration.Milliseconds(),
		SQLiteLockWaitMS:  lockWaitMS,
		SQLiteWriteHoldMS: writeHoldMS,
	}

	capLimit := h.diagnosticCap
	if capLimit <= 0 {
		capLimit = DefaultDiagnosticRingBufferCapacity
		h.diagnosticCap = capLimit
	}

	h.diagnosticEntries = append(h.diagnosticEntries, entry)
	if len(h.diagnosticEntries) > capLimit {
		overflow := len(h.diagnosticEntries) - capLimit
		h.diagnosticEntries = append([]DiagnosticEntry(nil), h.diagnosticEntries[overflow:]...)
	}

	legacyLine := formatDiagnosticLogLine(entry)
	h.diagnostics = append(h.diagnostics, legacyLine)
	if len(h.diagnostics) > capLimit {
		overflow := len(h.diagnostics) - capLimit
		h.diagnostics = append([]string(nil), h.diagnostics[overflow:]...)
	}
}

func (h *Host) ResetDiagnostics(capacity int) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if capacity <= 0 {
		capacity = DefaultDiagnosticRingBufferCapacity
	}
	h.diagnosticCap = capacity
	h.diagnosticEntries = make([]DiagnosticEntry, 0, capacity)
	h.diagnostics = make([]string, 0, capacity)
}

func (h *Host) ExportDiagnosticBundle(caller CallerContext, filterPluginID string) (*DiagnosticBundle, error) {
	if !isCallerAdminOrDPO(caller, false) {
		h.logError(fmt.Sprintf("Unauthorized diagnostic bundle export attempt by caller %s (staff %s)", caller.UserID, caller.StaffID))
		return nil, errors.New("UNAUTHORIZED_ADMIN_DIAGNOSTICS_SCOPE: downloading plugin diagnostic bundle requires System Admin / DPO authority")
	}

	trimmedFilter := strings.TrimSpace(filterPluginID)

	h.mu.RLock()
	capLimit := h.diagnosticCap
	if capLimit <= 0 {
		capLimit = DefaultDiagnosticRingBufferCapacity
	}
	entries := make([]DiagnosticEntry, 0, len(h.diagnosticEntries))
	for _, entry := range h.diagnosticEntries {
		if trimmedFilter != "" && entry.PluginID != trimmedFilter {
			continue
		}
		entries = append(entries, entry)
	}
	h.mu.RUnlock()

	logLines := make([]string, 0, len(entries))
	for _, entry := range entries {
		logLines = append(logLines, formatDiagnosticLogLine(entry))
	}

	h.logInfo(fmt.Sprintf("Exported PII-scrubbed plugin diagnostic bundle (%d entries) by staff %s", len(entries), caller.StaffID))
	return &DiagnosticBundle{
		SchemaVersion:      "1.0",
		SchoolURN:          h.cfg.SchoolURN,
		GeneratedAt:        time.Now().UTC().Format(time.RFC3339),
		GeneratedByStaffID: caller.StaffID,
		TelemetryMode:      "ADMIN_EXPORT_ONLY_ZERO_OUTBOUND",
		RingBufferCapacity: capLimit,
		EntryCount:         len(entries),
		PIIScrubbed:        true,
		Entries:            entries,
		LogArchive:         strings.Join(logLines, "\n"),
	}, nil
}

func (h *Host) RegisterDiagnosticRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/admin/diagnostics/bundle", func(w http.ResponseWriter, r *http.Request) {
		staffID := strings.TrimSpace(r.URL.Query().Get("staff_id"))
		pluginID := strings.TrimSpace(r.URL.Query().Get("plugin_id"))

		caller, err := h.ResolveRequestCaller(r, staffID)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]any{
				"ok":         false,
				"error_code": err.Error(),
			})
			return
		}

		bundle, err := h.ExportDiagnosticBundle(caller, pluginID)
		if err != nil {
			status := http.StatusBadRequest
			if strings.Contains(err.Error(), "UNAUTHORIZED_ADMIN_DIAGNOSTICS_SCOPE") {
				status = http.StatusForbidden
			}
			writeJSON(w, status, map[string]any{
				"ok":         false,
				"error_code": err.Error(),
			})
			return
		}

		filename := fmt.Sprintf("optimis-plugin-diagnostics-%s.json", h.cfg.SchoolURN)
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":     true,
			"bundle": bundle,
		})
	})

	mux.HandleFunc("POST /api/test/diagnostics/reset", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Capacity int `json:"capacity"`
		}
		if r.Body != nil {
			_ = json.NewDecoder(r.Body).Decode(&body)
		}
		h.ResetDiagnostics(body.Capacity)
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":                   true,
			"ring_buffer_capacity": h.diagnosticCap,
		})
	})
}

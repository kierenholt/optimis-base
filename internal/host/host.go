package host

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"optimis/internal/dal"
)

type Config struct {
	DataDirPath      string
	SchoolURN        string
	HostKMSKeyHandle string
	PluginsDir       string
	CoreDBPath       string
}

type EndpointSpec struct {
	Scopes         []string `json:"scopes"`
	RequireRoles   []string `json:"require_roles,omitempty"`
	Mutates        bool     `json:"mutates"`
	TimeoutMS      int      `json:"timeout_ms,omitempty"`
	Events         []string `json:"events,omitempty"`
	Event          string   `json:"event,omitempty"`
	Cron           string   `json:"cron,omitempty"`
	MaxConcurrency int      `json:"max_concurrency,omitempty"`
}

type EgressSpec struct {
	AllowedDomains []string `json:"allowed_domains"`
	TimeoutMS      int      `json:"timeout_ms,omitempty"`
}

type DatabaseSpec struct {
	Alias       string `json:"alias,omitempty"`
	MaxDBSizeMB int    `json:"max_db_size_mb,omitempty"`
}

type GDPRHooksSpec struct {
	ExportPupilData string `json:"export_pupil_data"`
	PurgePupilData  string `json:"purge_pupil_data"`
}

type WidgetSpec struct {
	ID       string   `json:"id"`
	Title    string   `json:"title"`
	Entry    string   `json:"entry"`
	Width    int      `json:"width"`
	Height   int      `json:"height"`
	Keywords []string `json:"keywords,omitempty"`
}

type ViewSpec struct {
	ID                 string   `json:"id"`
	Title              string   `json:"title,omitempty"`
	Path               string   `json:"path"`
	Entry              string   `json:"entry"`
	QueryParams        []string `json:"query_params,omitempty"`
	AllowedQueryParams []string `json:"allowed_query_params,omitempty"`
	Keywords           []string `json:"keywords,omitempty"`
}

type Manifest struct {
	ID             string                  `json:"id"`
	Title          string                  `json:"title,omitempty"`
	Icon           string                  `json:"icon,omitempty"`
	IconSVG        string                  `json:"icon_svg,omitempty"`
	Version        string                  `json:"version"`
	CoreDALVersion string                  `json:"core_dal_version"`
	MaxConcurrency int                     `json:"max_concurrency,omitempty"`
	MaxMemoryMB    int                     `json:"max_memory_mb,omitempty"`
	Database       *DatabaseSpec           `json:"database,omitempty"`
	GDPRHooks      *GDPRHooksSpec          `json:"gdpr_hooks,omitempty"`
	Egress         EgressSpec              `json:"egress"`
	Endpoints      map[string]EndpointSpec `json:"endpoints"`
	Workers        map[string]EndpointSpec `json:"workers,omitempty"`
	Views          []ViewSpec              `json:"views,omitempty"`
	Widgets        []WidgetSpec            `json:"widgets,omitempty"`
}

type PluginState string

const (
	PluginStateActive             PluginState = "ACTIVE"
	PluginStateUninstalledDormant PluginState = "UNINSTALLED_DORMANT"
)

type LoadedPlugin struct {
	PlugID           string
	Manifest         Manifest
	LuaSource        string
	DirPath          string
	Folder           string
	State            PluginState
	ManifestModTime  time.Time
	LuaModTime       time.Time
	IsSyntheticClone bool
}

type CallerContext struct {
	Type        string   `json:"type"`
	UserID      string   `json:"user_id"`
	StaffID     string   `json:"staff_id"`
	ActiveRoles []string `json:"active_roles"`
	Scopes      []string `json:"scopes"`
	RequestID   string   `json:"request_id"`
}

type RPCResponse struct {
	OK        bool   `json:"ok"`
	Data      any    `json:"data,omitempty"`
	ErrorCode string `json:"error_code,omitempty"`
	Error     string `json:"error,omitempty"`
	Mutated   bool   `json:"mutated,omitempty"`
}

type Event struct {
	Name    string
	Payload map[string]any
	Hop     uint8
}

type Host struct {
	cfg     Config
	coreDAL *dal.CoreDAL

	mu                sync.RWMutex
	closed            bool
	plugins           map[string]*LoadedPlugin
	serviceLogs       []string
	diagnostics       []string
	diagnosticEntries []DiagnosticEntry
	diagnosticCap     int
	subscribers       map[string][]chan Event
	workerPools       map[string]*pluginEventWorkerPool

	eventsDispatchedTotal  atomic.Uint64
	eventsDroppedTotal     atomic.Uint64
	workerInvocationsTotal atomic.Uint64

	structuredStdout io.Writer
}

func (h *Host) SetStructuredStdout(w io.Writer) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.structuredStdout = w
}

func (h *Host) emitStructuredStdout(line string) {
	h.mu.RLock()
	w := h.structuredStdout
	h.mu.RUnlock()
	if w != nil {
		_, _ = fmt.Fprintln(w, line)
	}
}

func NewHost(cfg Config) (*Host, error) {
	if strings.TrimSpace(cfg.DataDirPath) == "" {
		return nil, errors.New("missing mandatory config: DataDirPath")
	}
	if strings.TrimSpace(cfg.SchoolURN) == "" {
		return nil, errors.New("missing mandatory config: SchoolURN")
	}
	if strings.TrimSpace(cfg.HostKMSKeyHandle) == "" {
		return nil, errors.New("missing mandatory config: HostKMSKeyHandle")
	}

	if err := os.MkdirAll(cfg.DataDirPath, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create DataDirPath: %w", err)
	}

	coreDAL, err := dal.NewCoreDALWithPath(cfg.DataDirPath, cfg.SchoolURN, cfg.CoreDBPath)
	if err != nil {
		return nil, err
	}

	h := &Host{
		cfg:               cfg,
		coreDAL:           coreDAL,
		plugins:           make(map[string]*LoadedPlugin),
		serviceLogs:       make([]string, 0, 128),
		diagnostics:       make([]string, 0, DefaultDiagnosticRingBufferCapacity),
		diagnosticEntries: make([]DiagnosticEntry, 0, DefaultDiagnosticRingBufferCapacity),
		diagnosticCap:     DefaultDiagnosticRingBufferCapacity,
		subscribers:       make(map[string][]chan Event),
		workerPools:       make(map[string]*pluginEventWorkerPool),
	}

	coreDAL.SetPostCommitEventHook(func(evt dal.CoreDomainEvent) {
		h.dispatchEvents([]Event{
			{
				Name:    evt.Name,
				Payload: evt.Payload,
				Hop:     evt.Hop,
			},
		})
	})

	h.logInfo(fmt.Sprintf("Host initialized for school URN %s (core_db=%s)", cfg.SchoolURN, coreDAL.DBPath()))
	return h, nil
}

func (h *Host) DAL() *dal.CoreDAL {
	return h.coreDAL
}

func (h *Host) Close() error {
	h.stopAllPluginWorkerPools()
	return h.coreDAL.Close()
}

func (h *Host) appendLogLineToFileLocked(entry string) {
	if strings.TrimSpace(h.cfg.DataDirPath) == "" {
		return
	}
	logFilePath := filepath.Join(h.cfg.DataDirPath, "mis-server.log")
	f, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = f.WriteString(entry + "\n")
}

func (h *Host) logInfo(msg string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	entry := fmt.Sprintf("[%s] INFO %s", time.Now().UTC().Format(time.RFC3339Nano), msg)
	h.serviceLogs = append(h.serviceLogs, entry)
	h.appendLogLineToFileLocked(entry)
}

func (h *Host) logError(msg string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	entry := fmt.Sprintf("[%s] ERROR %s", time.Now().UTC().Format(time.RFC3339Nano), msg)
	h.serviceLogs = append(h.serviceLogs, entry)
	h.appendLogLineToFileLocked(entry)
}

func (h *Host) recordDiagnostic(pluginID string, method string, errClass string, duration time.Duration) {
	h.recordDiagnosticDetailed(pluginID, method, errClass, nil, duration, nil, nil)
}

func (h *Host) LogCursor() string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return strconv.Itoa(len(h.serviceLogs))
}

func (h *Host) LogsSince(cursor string) []string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	idx, err := strconv.Atoi(cursor)
	if err != nil || idx < 0 {
		idx = 0
	}
	if idx >= len(h.serviceLogs) {
		return []string{}
	}
	out := make([]string, len(h.serviceLogs[idx:]))
	copy(out, h.serviceLogs[idx:])
	return out
}

func (h *Host) Diagnostics() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	out := make([]string, len(h.diagnostics))
	copy(out, h.diagnostics)
	return out
}

func (h *Host) SubscribeEvents(eventName string) <-chan Event {
	h.mu.Lock()
	defer h.mu.Unlock()
	ch := make(chan Event, 16)
	h.subscribers[eventName] = append(h.subscribers[eventName], ch)
	return ch
}

func (h *Host) dispatchEvents(events []Event) {
	for _, evt := range events {
		if evt.Hop == 0 {
			evt.Hop = 1
		}
		if evt.Hop > maxEventHopLimit {
			h.eventsDroppedTotal.Add(1)
			h.logInfo(fmt.Sprintf("Dropped event %s exceeding max hop count (hop=%d > %d)", evt.Name, evt.Hop, maxEventHopLimit))
			continue
		}

		h.eventsDispatchedTotal.Add(1)

		h.mu.RLock()
		rawSubs := h.subscribers[evt.Name]
		subs := make([]chan Event, len(rawSubs))
		copy(subs, rawSubs)

		var matchingPools []*pluginEventWorkerPool
		for pluginID, pool := range h.workerPools {
			if len(pool.subscribedEvents[evt.Name]) == 0 {
				continue
			}
			if p, ok := h.plugins[pluginID]; ok && p.State == PluginStateUninstalledDormant {
				continue
			}
			matchingPools = append(matchingPools, pool)
		}
		h.mu.RUnlock()

		for _, ch := range subs {
			select {
			case ch <- evt:
			default:
			}
		}

		for _, pool := range matchingPools {
			select {
			case pool.queue <- evt:
			default:
				h.eventsDroppedTotal.Add(1)
				h.logError(fmt.Sprintf("Dropped event %s for plugin %s: per-plugin buffered channel full", evt.Name, pool.pluginID))
			}
		}
	}
}

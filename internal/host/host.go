package host

import (
	"context"
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

	lua "github.com/yuin/gopher-lua"

	"optimis/internal/dal"
	"optimis/internal/host/dalbindings"
	"optimis/internal/host/types"
)

type Config = types.Config
type EndpointSpec = types.EndpointSpec
type EgressSpec = types.EgressSpec
type DatabaseSpec = types.DatabaseSpec
type GDPRHooksSpec = types.GDPRHooksSpec
type WidgetSpec = types.WidgetSpec
type ViewSpec = types.ViewSpec
type Manifest = types.Manifest
type PluginState = types.PluginState

const (
	PluginStateActive             = types.PluginStateActive
	PluginStateUninstalledDormant = types.PluginStateUninstalledDormant
)

type LoadedPlugin = types.LoadedPlugin
type CallerContext = types.CallerContext
type RPCResponse = types.RPCResponse
type Event = types.Event

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

func (h *Host) listActivePluginsForDAL() []*types.LoadedPlugin {
	h.mu.RLock()
	defer h.mu.RUnlock()
	out := make([]*types.LoadedPlugin, 0, len(h.plugins))
	for _, p := range h.plugins {
		out = append(out, p)
	}
	return out
}

func (h *Host) buildLuaDALTable(
	L *lua.LState,
	execCtx context.Context,
	endpointSpec EndpointSpec,
	caller CallerContext,
) *lua.LTable {
	return dalbindings.Bind(
		L,
		execCtx,
		h.coreDAL,
		endpointSpec,
		caller,
		h.cfg.HostKMSKeyHandle,
		h.listActivePluginsForDAL,
		h.ensureCloudKMSAvailable,
	)
}

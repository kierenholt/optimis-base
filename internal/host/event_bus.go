package host

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"

	lua "github.com/yuin/gopher-lua"

	"optimis/internal/dal"
)

const (
	defaultPluginEventBufferSize = 64
	defaultWorkerMaxConcurrency  = 1
	maxEventHopLimit             = 3
)

type EventBusMetricsSnapshot struct {
	EventsDispatchedTotal  uint64         `json:"events_dispatched_total"`
	EventsDroppedTotal     uint64         `json:"events_dropped_total"`
	WorkerInvocationsTotal uint64         `json:"worker_invocations_total"`
	PluginChannelDepths    map[string]int `json:"plugin_channel_depths"`
}

type pluginEventWorkerPool struct {
	pluginID         string
	queue            chan Event
	maxConcurrency   int
	subscribedEvents map[string][]string
	stopOnce         sync.Once
	stopCh           chan struct{}
	wg               sync.WaitGroup
}

func (p *pluginEventWorkerPool) stop() {
	p.stopOnce.Do(func() {
		close(p.stopCh)
	})
	p.wg.Wait()
}

func (s EndpointSpec) SubscribedEvents() []string {
	seen := make(map[string]struct{})
	var out []string
	if trimmed := strings.TrimSpace(s.Event); trimmed != "" {
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	for _, raw := range s.Events {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; !exists {
			seen[trimmed] = struct{}{}
			out = append(out, trimmed)
		}
	}
	return out
}

func resolvePluginWorkerConcurrency(manifest Manifest) int {
	if manifest.MaxConcurrency > 0 {
		return manifest.MaxConcurrency
	}
	maxWorkerConc := 0
	for _, wSpec := range manifest.Workers {
		if wSpec.MaxConcurrency > maxWorkerConc {
			maxWorkerConc = wSpec.MaxConcurrency
		}
	}
	if maxWorkerConc > 0 {
		return maxWorkerConc
	}
	return defaultWorkerMaxConcurrency
}

func validateManifestWorkersAndLuaSource(manifest Manifest, luaSource string) error {
	if len(manifest.Workers) == 0 {
		return nil
	}

	L := lua.NewState(lua.Options{SkipOpenLibs: true})
	defer L.Close()
	for _, lib := range []struct {
		name string
		fn   lua.LGFunction
	}{
		{lua.LoadLibName, lua.OpenPackage},
		{lua.BaseLibName, lua.OpenBase},
		{lua.TabLibName, lua.OpenTable},
		{lua.StringLibName, lua.OpenString},
		{lua.MathLibName, lua.OpenMath},
	} {
		L.Push(L.NewFunction(lib.fn))
		L.Push(lua.LString(lib.name))
		L.Call(1, 0)
	}
	L.SetGlobal("dofile", lua.LNil)
	L.SetGlobal("loadfile", lua.LNil)
	L.SetGlobal("package", lua.LNil)
	L.SetGlobal("require", lua.LNil)
	L.SetGlobal("os", lua.LNil)
	L.SetGlobal("io", lua.LNil)
	L.SetGlobal("debug", lua.LNil)

	if err := L.DoString(luaSource); err != nil {
		return fmt.Errorf("failed to parse init.lua for worker verification in plugin %s: %w", manifest.ID, err)
	}

	for workerName, workerSpec := range manifest.Workers {
		trimmedWorker := strings.TrimSpace(workerName)
		if trimmedWorker == "" {
			return fmt.Errorf("plugin %s declares an empty worker name in manifest.json", manifest.ID)
		}
		fn := L.GetGlobal(trimmedWorker)
		if fn.Type() != lua.LTFunction {
			return fmt.Errorf("manifest.json for plugin %s declares worker %q, but function %q is missing in src/init.lua", manifest.ID, trimmedWorker, trimmedWorker)
		}
		for _, evtName := range workerSpec.SubscribedEvents() {
			if err := dal.ValidateEventSubscriptionName(evtName); err != nil {
				return fmt.Errorf("invalid event subscription on worker %s.%s: %w", manifest.ID, trimmedWorker, err)
			}
		}
	}
	return nil
}

func (h *Host) startOrReplacePluginWorkerPool(plugin *LoadedPlugin) {
	subscribedEvents := make(map[string][]string)
	for workerName, workerSpec := range plugin.Manifest.Workers {
		for _, evtName := range workerSpec.SubscribedEvents() {
			subscribedEvents[evtName] = append(subscribedEvents[evtName], workerName)
		}
	}
	for evtName := range subscribedEvents {
		sort.Strings(subscribedEvents[evtName])
	}

	h.mu.Lock()
	oldPool := h.workerPools[plugin.Manifest.ID]
	delete(h.workerPools, plugin.Manifest.ID)
	h.mu.Unlock()

	if oldPool != nil {
		oldPool.stop()
	}

	if len(subscribedEvents) == 0 {
		return
	}

	concurrency := resolvePluginWorkerConcurrency(plugin.Manifest)
	pool := &pluginEventWorkerPool{
		pluginID:         plugin.Manifest.ID,
		queue:            make(chan Event, defaultPluginEventBufferSize),
		maxConcurrency:   concurrency,
		subscribedEvents: subscribedEvents,
		stopCh:           make(chan struct{}),
	}

	for i := 0; i < concurrency; i++ {
		pool.wg.Add(1)
		go func() {
			defer pool.wg.Done()
			for {
				select {
				case <-pool.stopCh:
					return
				case evt, ok := <-pool.queue:
					if !ok {
						return
					}
					workers := pool.subscribedEvents[evt.Name]
					if len(workers) > 0 {
						h.processPluginWorkerEvent(pool.pluginID, workers, evt)
					}
				}
			}
		}()
	}

	h.mu.Lock()
	if h.closed {
		h.mu.Unlock()
		pool.stop()
		return
	}
	h.workerPools[plugin.Manifest.ID] = pool
	h.mu.Unlock()
}

func (h *Host) stopAllPluginWorkerPools() {
	h.mu.Lock()
	h.closed = true
	pools := make([]*pluginEventWorkerPool, 0, len(h.workerPools))
	for id, p := range h.workerPools {
		pools = append(pools, p)
		delete(h.workerPools, id)
	}
	h.mu.Unlock()

	for _, p := range pools {
		p.stop()
	}
}

func (h *Host) processPluginWorkerEvent(pluginID string, workerNames []string, evt Event) {
	plugin, ok := h.GetLoadedPlugin(pluginID)
	if !ok || plugin.State == PluginStateUninstalledDormant {
		return
	}

	for _, workerName := range workerNames {
		args := make(map[string]any, len(evt.Payload)+3)
		for k, v := range evt.Payload {
			args[k] = v
		}
		args["event_name"] = evt.Name
		args["hop"] = int(evt.Hop)
		args["payload"] = evt.Payload

		workerCtx := dal.WithEventHop(context.Background(), evt.Hop)
		h.workerInvocationsTotal.Add(1)
		if _, err := h.ExecuteWorker(workerCtx, pluginID, workerName, args); err != nil {
			h.logError(fmt.Sprintf("Event worker %s.%s failed on event %s (hop=%d): %v", pluginID, workerName, evt.Name, evt.Hop, err))
		}
	}
}

func (h *Host) EventBusMetrics() EventBusMetricsSnapshot {
	h.mu.RLock()
	depths := make(map[string]int, len(h.workerPools))
	for pluginID, pool := range h.workerPools {
		depths[pluginID] = len(pool.queue)
	}
	h.mu.RUnlock()

	return EventBusMetricsSnapshot{
		EventsDispatchedTotal:  h.eventsDispatchedTotal.Load(),
		EventsDroppedTotal:     h.eventsDroppedTotal.Load(),
		WorkerInvocationsTotal: h.workerInvocationsTotal.Load(),
		PluginChannelDepths:    depths,
	}
}

func (h *Host) RegisterEventBusRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/events/catalog", func(w http.ResponseWriter, r *http.Request) {
		h.mu.RLock()
		pluginSubs := make(map[string]any, len(h.plugins))
		for pluginID, p := range h.plugins {
			workersInfo := make(map[string]any)
			for wName, wSpec := range p.Manifest.Workers {
				workersInfo[wName] = map[string]any{
					"events":  wSpec.SubscribedEvents(),
					"scopes":  wSpec.Scopes,
					"mutates": wSpec.Mutates,
				}
			}
			pluginSubs[pluginID] = map[string]any{
				"state":           p.State,
				"max_concurrency": resolvePluginWorkerConcurrency(p.Manifest),
				"workers":         workersInfo,
			}
		}
		h.mu.RUnlock()

		writeJSON(w, http.StatusOK, map[string]any{
			"ok":                   true,
			"core_events":          dal.CoreEventCatalog(),
			"max_hop_limit":        maxEventHopLimit,
			"metrics":              h.EventBusMetrics(),
			"plugin_subscriptions": pluginSubs,
		})
	})
}

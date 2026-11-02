package types

import (
	"strings"
	"time"
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
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Icon        string   `json:"icon,omitempty"`
	Entry       string   `json:"entry"`
	Width       int      `json:"width"`
	Height      int      `json:"height"`
	Keywords    []string `json:"keywords,omitempty"`
}

type ViewSpec struct {
	ID                 string   `json:"id"`
	Title              string   `json:"title,omitempty"`
	Icon               string   `json:"icon,omitempty"`
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
	InstalledAt      string
	UpdatedAt        string
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

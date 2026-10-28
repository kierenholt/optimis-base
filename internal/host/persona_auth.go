package host

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"

	"github.com/google/uuid"

	"optimis/internal/dal"
)

const SessionCookieName = "mis_session"

type hostAuthState struct {
	mu                       sync.RWMutex
	configured               bool
	enableDevPersonaSwitcher bool
	sessions                 map[string]string
	activeDevStaffID         string
}

var (
	authStatesMu sync.Mutex
	authStates   = make(map[*Host]*hostAuthState)
)

func (h *Host) getAuthState() *hostAuthState {
	authStatesMu.Lock()
	defer authStatesMu.Unlock()
	st, ok := authStates[h]
	if !ok {
		enableDev := true
		if raw := strings.TrimSpace(os.Getenv("ENABLE_DEV_PERSONA_SWITCHER")); strings.EqualFold(raw, "false") {
			enableDev = false
		}
		st = &hostAuthState{
			configured:               false,
			enableDevPersonaSwitcher: enableDev,
			sessions:                 make(map[string]string),
		}
		authStates[h] = st
	}
	return st
}

func (h *Host) isDevPersonaSwitcherEnabled() bool {
	st := h.getAuthState()
	st.mu.RLock()
	defer st.mu.RUnlock()
	return st.enableDevPersonaSwitcher
}

func (h *Host) validateRealSSOProvider(ctx context.Context) (*dal.SSOConfig, error) {
	ssoCfg, err := h.coreDAL.ValidateAndLoadSSOConfig(ctx)
	if err != nil {
		h.logError(fmt.Sprintf("REAL_SSO_NOT_CONFIGURED: failed to load real federated SSO provider while ENABLE_DEV_PERSONA_SWITCHER=false: %v", err))
		return nil, fmt.Errorf("REAL_SSO_NOT_CONFIGURED: real federated SSO cannot be loaded when ENABLE_DEV_PERSONA_SWITCHER=false (%w)", err)
	}
	return ssoCfg, nil
}

func (h *Host) loadRealSSOProvider(ctx context.Context) error {
	_, err := h.validateRealSSOProvider(ctx)
	return err
}

func (h *Host) ConfigureAuthMode(ctx context.Context, enableDevPersonaSwitcher bool) error {
	st := h.getAuthState()
	st.mu.Lock()
	st.configured = true
	st.enableDevPersonaSwitcher = enableDevPersonaSwitcher
	st.sessions = make(map[string]string)
	st.activeDevStaffID = ""
	st.mu.Unlock()

	if !enableDevPersonaSwitcher {
		return h.loadRealSSOProvider(ctx)
	}

	h.logInfo("Configured host authentication mode: ENABLE_DEV_PERSONA_SWITCHER=true")
	return nil
}

func (h *Host) ResetDevPersonaSessions() {
	st := h.getAuthState()
	st.mu.Lock()
	defer st.mu.Unlock()
	st.sessions = make(map[string]string)
	st.activeDevStaffID = ""
}

func (h *Host) ResolveRequestCaller(r *http.Request, explicitStaffID string) (CallerContext, error) {
	if !h.isDevPersonaSwitcherEnabled() {
		if err := h.loadRealSSOProvider(r.Context()); err != nil {
			return CallerContext{}, err
		}
	}

	st := h.getAuthState()

	// 1. If an explicit staff_id was provided in request/query, resolve it directly
	if trimmedStaff := strings.TrimSpace(explicitStaffID); trimmedStaff != "" {
		if persona, pErr := h.coreDAL.ResolveStaffPersonaByID(r.Context(), trimmedStaff); pErr == nil {
			return CallerContext{
				Type:        "USER",
				UserID:      persona.UserID,
				StaffID:     persona.StaffID,
				ActiveRoles: persona.ActiveRoles,
				Scopes:      persona.Scopes,
				RequestID:   "req_" + uuid.NewString(),
			}, nil
		}
		return CallerContext{
			Type:        "USER",
			UserID:      "usr_teacher_1",
			StaffID:     trimmedStaff,
			ActiveRoles: []string{"teacher"},
			Scopes: []string{
				"dal.pupils.read:all",
				"dal.pupils.read:own_groups",
				"dal.groups.read:own_groups",
				"dal.attendance.read:own_groups",
				"dal.attendance.write:own_groups",
				"dal.sen.read:own_groups",
				"dal.medical.read:own_groups",
				"dal.guardians.read:own_groups",
				"dal.safeguarding.create",
			},
			RequestID: "req_" + uuid.NewString(),
		}, nil
	}

	// 2. Check explicit mis_session cookie
	if cookie, err := r.Cookie(SessionCookieName); err == nil && strings.TrimSpace(cookie.Value) != "" {
		token := strings.TrimSpace(cookie.Value)
		st.mu.RLock()
		staffID, exists := st.sessions[token]
		st.mu.RUnlock()
		if exists && strings.TrimSpace(staffID) != "" {
			persona, pErr := h.coreDAL.ResolveStaffPersonaByID(r.Context(), staffID)
			if pErr != nil {
				return CallerContext{}, fmt.Errorf("INVALID_SESSION_PERSONA: %w", pErr)
			}
			return CallerContext{
				Type:        "USER",
				UserID:      persona.UserID,
				StaffID:     persona.StaffID,
				ActiveRoles: persona.ActiveRoles,
				Scopes:      persona.Scopes,
				RequestID:   "req_" + uuid.NewString(),
			}, nil
		}
	}

	// 3. Check active switched dev persona on the Host
	st.mu.RLock()
	activeStaffID := st.activeDevStaffID
	st.mu.RUnlock()

	if strings.TrimSpace(activeStaffID) != "" {
		persona, pErr := h.coreDAL.ResolveStaffPersonaByID(r.Context(), activeStaffID)
		if pErr != nil {
			return CallerContext{}, fmt.Errorf("INVALID_ACTIVE_PERSONA: %w", pErr)
		}
		return CallerContext{
			Type:        "USER",
			UserID:      persona.UserID,
			StaffID:     persona.StaffID,
			ActiveRoles: persona.ActiveRoles,
			Scopes:      persona.Scopes,
			RequestID:   "req_" + uuid.NewString(),
		}, nil
	}

	// 4. Default dev testbed caller when no persona has been explicitly switched yet
	return CallerContext{
		Type:        "USER",
		UserID:      "usr_teacher_1",
		StaffID:     "stf_001",
		ActiveRoles: []string{"teacher"},
		Scopes: []string{
			"dal.pupils.read:all",
			"dal.pupils.read:own_groups",
			"dal.groups.read:own_groups",
			"dal.attendance.read:own_groups",
			"dal.attendance.write:own_groups",
			"dal.guardians.read:own_groups",
			"dal.safeguarding.create",
		},
		RequestID: "req_" + uuid.NewString(),
	}, nil
}

func (h *Host) RegisterAuthAndPersonaRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/auth/personas", func(w http.ResponseWriter, r *http.Request) {
		if !h.isDevPersonaSwitcherEnabled() {
			h.logError("Blocked GET /api/auth/personas because ENABLE_DEV_PERSONA_SWITCHER=false")
			writeJSON(w, http.StatusForbidden, map[string]any{
				"ok":         false,
				"error_code": "DEV_PERSONA_SWITCHER_DISABLED",
			})
			return
		}

		personas, err := h.coreDAL.ListActiveStaffPersonas(r.Context())
		if err != nil {
			h.logError(fmt.Sprintf("Failed to list active staff personas: %v", err))
			writeJSON(w, http.StatusInternalServerError, map[string]any{
				"ok":         false,
				"error_code": err.Error(),
			})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"ok":       true,
			"personas": personas,
		})
	})

	mux.HandleFunc("POST /api/auth/persona", func(w http.ResponseWriter, r *http.Request) {
		if !h.isDevPersonaSwitcherEnabled() {
			h.logError("Blocked POST /api/auth/persona because ENABLE_DEV_PERSONA_SWITCHER=false")
			writeJSON(w, http.StatusForbidden, map[string]any{
				"ok":         false,
				"error_code": "DEV_PERSONA_SWITCHER_DISABLED",
			})
			return
		}

		var body struct {
			StaffID string `json:"staff_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"ok":         false,
				"error_code": fmt.Sprintf("invalid JSON body: %v", err),
			})
			return
		}
		if strings.TrimSpace(body.StaffID) == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"ok":         false,
				"error_code": "missing mandatory field: staff_id",
			})
			return
		}

		persona, err := h.coreDAL.ResolveStaffPersonaByID(r.Context(), body.StaffID)
		if err != nil {
			h.logError(fmt.Sprintf("Failed to switch persona to %s: %v", body.StaffID, err))
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"ok":         false,
				"error_code": err.Error(),
			})
			return
		}

		sessionToken := "sess_" + strings.ReplaceAll(uuid.NewString(), "-", "")
		st := h.getAuthState()
		st.mu.Lock()
		st.sessions[sessionToken] = persona.StaffID
		st.activeDevStaffID = persona.StaffID
		st.mu.Unlock()

		http.SetCookie(w, &http.Cookie{
			Name:     SessionCookieName,
			Value:    sessionToken,
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteStrictMode,
		})

		caller := CallerContext{
			Type:        "USER",
			UserID:      persona.UserID,
			StaffID:     persona.StaffID,
			ActiveRoles: persona.ActiveRoles,
			Scopes:      persona.Scopes,
			RequestID:   "req_" + uuid.NewString(),
		}

		h.logInfo(fmt.Sprintf("Switched active dev persona to %s (%v)", persona.StaffID, persona.ActiveRoles))
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":            true,
			"session_token": sessionToken,
			"persona":       persona,
			"caller":        caller,
		})
	})

	mux.HandleFunc("GET /api/auth/session", func(w http.ResponseWriter, r *http.Request) {
		caller, err := h.ResolveRequestCaller(r, strings.TrimSpace(r.URL.Query().Get("staff_id")))
		if err != nil {
			status := http.StatusUnauthorized
			if strings.Contains(err.Error(), "REAL_SSO_NOT_CONFIGURED") {
				status = http.StatusInternalServerError
			}
			writeJSON(w, status, map[string]any{
				"ok":         false,
				"error_code": err.Error(),
			})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"ok":     true,
			"caller": caller,
		})
	})

	mux.HandleFunc("GET /api/auth/sso/login", func(w http.ResponseWriter, r *http.Request) {
		if !h.isDevPersonaSwitcherEnabled() {
			ssoCfg, err := h.validateRealSSOProvider(r.Context())
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]any{
					"ok":         false,
					"error_code": err.Error(),
				})
				return
			}
			authURL := fmt.Sprintf(
				"%s/oauth2/v2.0/authorize?client_id=%s&response_type=code&scope=openid+profile+email",
				strings.TrimRight(ssoCfg.IssuerURL, "/"),
				url.QueryEscape(ssoCfg.ClientID),
			)
			if ssoCfg.RedirectURI != "" {
				authURL += "&redirect_uri=" + url.QueryEscape(ssoCfg.RedirectURI)
			}
			writeJSON(w, http.StatusOK, map[string]any{
				"ok":                true,
				"status":            "SSO_CONFIGURED",
				"provider":          ssoCfg.Provider,
				"issuer_url":        ssoCfg.IssuerURL,
				"client_id":         ssoCfg.ClientID,
				"redirect_uri":      ssoCfg.RedirectURI,
				"authorization_url": authURL,
			})
			return
		}
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"ok":         false,
			"error_code": "SSO_DISABLED_IN_DEV_PERSONA_MODE",
		})
	})

	mux.HandleFunc("DELETE /api/test/system-config/{key}", func(w http.ResponseWriter, r *http.Request) {
		key := strings.TrimSpace(r.PathValue("key"))
		if key == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"ok":    false,
				"error": "missing mandatory path parameter: key",
			})
			return
		}
		if err := h.coreDAL.DeleteSystemConfigKey(r.Context(), key); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"ok":    false,
				"error": err.Error(),
			})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":      true,
			"deleted": true,
		})
	})

	mux.HandleFunc("POST /api/test/auth-mode", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			EnableDevPersonaSwitcher *bool `json:"enable_dev_persona_switcher"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.EnableDevPersonaSwitcher == nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"ok":         false,
				"error_code": "missing mandatory boolean field: enable_dev_persona_switcher",
			})
			return
		}

		if err := h.ConfigureAuthMode(r.Context(), *body.EnableDevPersonaSwitcher); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{
				"ok":         false,
				"error_code": err.Error(),
			})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"ok":                          true,
			"enable_dev_persona_switcher": *body.EnableDevPersonaSwitcher,
		})
	})

	mux.HandleFunc("POST /api/test/seed-personas", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			AcademicYearID string `json:"academic_year_id"`
			EffectiveFrom  string `json:"effective_from"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"error": fmt.Sprintf("invalid JSON body: %v", err),
			})
			return
		}
		seeded, err := h.coreDAL.SeedPersonaTestFixtures(r.Context(), body.AcademicYearID, body.EffectiveFrom)
		if err != nil {
			h.logError(fmt.Sprintf("Failed to seed persona fixtures: %v", err))
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"error": err.Error(),
			})
			return
		}
		h.ResetDevPersonaSessions()
		h.logInfo(fmt.Sprintf("Seeded persona test fixtures %s", seeded.FixtureID))
		writeJSON(w, http.StatusCreated, seeded)
	})

	mux.HandleFunc("DELETE /api/test/seed-personas/{fixture_id}", func(w http.ResponseWriter, r *http.Request) {
		fixtureID := strings.TrimSpace(r.PathValue("fixture_id"))
		if fixtureID == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"error": "missing mandatory path parameter: fixture_id",
			})
			return
		}
		if err := h.coreDAL.DeletePersonaTestFixtures(r.Context(), fixtureID); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"error": err.Error(),
			})
			return
		}
		h.ResetDevPersonaSessions()
		h.logInfo(fmt.Sprintf("Deleted persona test fixtures %s", fixtureID))
		writeJSON(w, http.StatusOK, map[string]any{
			"deleted": true,
		})
	})
}

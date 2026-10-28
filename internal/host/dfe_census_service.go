package host

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"optimis/internal/dal"
)

type MockDfeCensusSubmission struct {
	SubmissionID string `json:"submission_id"`
	SchoolURN    string `json:"school_urn"`
	TermCode     string `json:"term_code"`
	CensusDate   string `json:"census_date"`
	Status       string `json:"status"`
	PupilCount   int    `json:"pupil_count"`
	XMLPayload   string `json:"xml_payload"`
	SubmittedAt  string `json:"submitted_at"`
}

type hostCensusServiceState struct {
	mu                          sync.RWMutex
	enableMockDfeCensusService  bool
	submissions                 []MockDfeCensusSubmission
}

var (
	censusStatesMu sync.Mutex
	censusStates   = make(map[*Host]*hostCensusServiceState)
)

func (h *Host) getCensusServiceState() *hostCensusServiceState {
	censusStatesMu.Lock()
	defer censusStatesMu.Unlock()
	st, ok := censusStates[h]
	if !ok {
		enabled := true
		if raw := strings.TrimSpace(os.Getenv("ENABLE_MOCK_DFE_CENSUS_SERVICE")); strings.EqualFold(raw, "false") {
			enabled = false
		}
		st = &hostCensusServiceState{
			enableMockDfeCensusService: enabled,
			submissions:                make([]MockDfeCensusSubmission, 0),
		}
		censusStates[h] = st
	}
	return st
}

func (h *Host) ConfigureMockDfeCensusService(enabled bool) {
	st := h.getCensusServiceState()
	st.mu.Lock()
	st.enableMockDfeCensusService = enabled
	st.mu.Unlock()
	h.logInfo(fmt.Sprintf("Configured Mock DfE COLLECT census service mode: ENABLE_MOCK_DFE_CENSUS_SERVICE=%v", enabled))
}

func (h *Host) IsMockDfeCensusServiceEnabled() bool {
	st := h.getCensusServiceState()
	st.mu.RLock()
	defer st.mu.RUnlock()
	return st.enableMockDfeCensusService
}

func (h *Host) ListMockDfeCensusSubmissions() []MockDfeCensusSubmission {
	st := h.getCensusServiceState()
	st.mu.RLock()
	defer st.mu.RUnlock()
	out := make([]MockDfeCensusSubmission, len(st.submissions))
	copy(out, st.submissions)
	return out
}

func (h *Host) ClearMockDfeCensusSubmissions() {
	st := h.getCensusServiceState()
	st.mu.Lock()
	st.submissions = make([]MockDfeCensusSubmission, 0)
	st.mu.Unlock()
}

func (h *Host) handleMockDfeCollectRoundTrip(req *http.Request) (*http.Response, error) {
	if !h.IsMockDfeCensusServiceEnabled() {
		return nil, errors.New("MOCK_DFE_CENSUS_SERVICE_DISABLED: mock DfE COLLECT census service is disabled by feature flag ENABLE_MOCK_DFE_CENSUS_SERVICE=false")
	}

	var bodyBytes []byte
	if req.Body != nil {
		var err error
		bodyBytes, err = io.ReadAll(io.LimitReader(req.Body, maxEgressBodyBytes))
		if err != nil {
			return nil, fmt.Errorf("failed reading outbound COLLECT XML payload: %w", err)
		}
	}
	xmlPayload := strings.TrimSpace(string(bodyBytes))
	if xmlPayload == "" {
		return nil, errors.New("MOCK_DFE_CENSUS_EMPTY_PAYLOAD: outbound COLLECT submission requires a non-empty XML body")
	}

	valid, xsdErrs, parsed := ValidateDfeSchoolCensusXML(xmlPayload)
	if !valid || parsed == nil {
		return nil, fmt.Errorf("XSD_VALIDATION_FAILED: mock COLLECT rejected invalid XML: %s", strings.Join(xsdErrs, "; "))
	}

	submissionID := "collect_sub_" + strings.ReplaceAll(uuid.NewString(), "-", "")[:10]
	sub := MockDfeCensusSubmission{
		SubmissionID: submissionID,
		SchoolURN:    strings.TrimSpace(parsed.Header.Source.URN),
		TermCode:     strings.TrimSpace(parsed.Header.Term),
		CensusDate:   strings.TrimSpace(parsed.Header.CensusDate),
		Status:       "COLLECT_ACCEPTED",
		PupilCount:   len(parsed.Pupils.PupilList),
		XMLPayload:   xmlPayload,
		SubmittedAt:  time.Now().UTC().Format(time.RFC3339),
	}

	st := h.getCensusServiceState()
	st.mu.Lock()
	st.submissions = append(st.submissions, sub)
	st.mu.Unlock()

	respBody, _ := json.Marshal(map[string]any{
		"ok":            true,
		"submission_id": sub.SubmissionID,
		"status":        sub.Status,
		"school_urn":    sub.SchoolURN,
		"term_code":     sub.TermCode,
		"census_date":   sub.CensusDate,
		"pupil_count":   sub.PupilCount,
	})

	hdr := make(http.Header)
	hdr.Set("Content-Type", "application/json; charset=utf-8")

	return &http.Response{
		StatusCode: http.StatusCreated,
		Status:     "201 Created",
		Header:     hdr,
		Body:       io.NopCloser(strings.NewReader(string(respBody))),
		Request:    req,
	}, nil
}

func (h *Host) RegisterCensusTestRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/test/census-service-mode", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			EnableMockDfeCensusService *bool `json:"enable_mock_dfe_census_service"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.EnableMockDfeCensusService == nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"ok":         false,
				"error_code": "missing mandatory boolean field: enable_mock_dfe_census_service",
			})
			return
		}
		h.ConfigureMockDfeCensusService(*body.EnableMockDfeCensusService)
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":                             true,
			"enable_mock_dfe_census_service": *body.EnableMockDfeCensusService,
		})
	})

	mux.HandleFunc("GET /api/test/census-submissions", func(w http.ResponseWriter, r *http.Request) {
		if !h.IsMockDfeCensusServiceEnabled() {
			writeJSON(w, http.StatusForbidden, map[string]any{
				"ok":         false,
				"error_code": "MOCK_DFE_CENSUS_SERVICE_DISABLED: mock DfE COLLECT census service is disabled by feature flag ENABLE_MOCK_DFE_CENSUS_SERVICE=false",
			})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":          true,
			"submissions": h.ListMockDfeCensusSubmissions(),
		})
	})

	mux.HandleFunc("DELETE /api/test/census-submissions", func(w http.ResponseWriter, r *http.Request) {
		h.ClearMockDfeCensusSubmissions()
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":      true,
			"cleared": true,
		})
	})

	mux.HandleFunc("POST /api/test/seed-census-fixtures", func(w http.ResponseWriter, r *http.Request) {
		var in dal.CensusTestFixtureInput
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"error": fmt.Sprintf("invalid JSON body: %v", err),
			})
			return
		}
		seeded, err := h.coreDAL.SeedCensusTestFixtures(r.Context(), in)
		if err != nil {
			h.logError(fmt.Sprintf("Failed to seed census test fixtures: %v", err))
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"error": err.Error(),
			})
			return
		}
		h.ResetDevPersonaSessions()
		h.logInfo(fmt.Sprintf("Seeded census test fixtures %s", seeded.FixtureID))
		writeJSON(w, http.StatusCreated, seeded)
	})

	mux.HandleFunc("DELETE /api/test/seed-census-fixtures/{fixture_id}", func(w http.ResponseWriter, r *http.Request) {
		fixtureID := strings.TrimSpace(r.PathValue("fixture_id"))
		if fixtureID == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"error": "missing mandatory path parameter: fixture_id",
			})
			return
		}
		if err := h.coreDAL.DeleteCensusTestFixtures(r.Context(), fixtureID); err != nil {
			h.logError(fmt.Sprintf("Failed to delete census test fixtures %s: %v", fixtureID, err))
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"error": err.Error(),
			})
			return
		}
		h.ResetDevPersonaSessions()
		h.logInfo(fmt.Sprintf("Deleted census test fixtures %s", fixtureID))
		writeJSON(w, http.StatusOK, map[string]any{
			"deleted": true,
		})
	})
}

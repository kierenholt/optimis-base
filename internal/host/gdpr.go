package host

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"optimis/internal/dal"
)

type SARExportBundle struct {
	PupilID         string           `json:"pupil_id"`
	SchoolURN       string           `json:"school_urn"`
	GeneratedAt     string           `json:"generated_at"`
	GeneratedByUser string           `json:"generated_by_user"`
	Sections        []dal.SARSection `json:"sections"`
}

type SARFinalizeInput struct {
	PupilID  string           `json:"pupil_id"`
	Sections []dal.SARSection `json:"sections"`
}

type SARFinalizedPackage struct {
	PupilID           string           `json:"pupil_id"`
	SchoolURN         string           `json:"school_urn"`
	FinalizedAt       string           `json:"finalized_at"`
	RedactedByStaffID string           `json:"redacted_by_staff_id"`
	RedactedByUserID  string           `json:"redacted_by_user_id"`
	Status            string           `json:"status"`
	Sections          []dal.SARSection `json:"sections"`
}

type RetentionPurgeResult struct {
	PupilID          string         `json:"pupil_id"`
	PurgedAt         string         `json:"purged_at"`
	PurgedByStaffID  string         `json:"purged_by_staff_id"`
	CorePurged       bool           `json:"core_purged"`
	PluginPurgedRows map[string]int `json:"plugin_purged_rows"`
	Status           string         `json:"status"`
}

func isCallerAdminOrDPO(caller CallerContext, requireWrite bool) bool {
	for _, r := range caller.ActiveRoles {
		if strings.EqualFold(r, "ADMIN") || strings.EqualFold(r, "DPO") {
			return true
		}
	}
	if requireWrite {
		return hasAllScopes(caller.Scopes, []string{"dal.pupils.write:all", "dal.staff.write:admin"})
	}
	return hasAllScopes(caller.Scopes, []string{"dal.pupils.read:all", "dal.staff.write:admin"})
}

func (h *Host) ExportPupilSAR(ctx context.Context, caller CallerContext, pupilID string) (*SARExportBundle, error) {
	trimmedPupilID := strings.TrimSpace(pupilID)
	if trimmedPupilID == "" {
		return nil, errors.New("missing mandatory parameter: pupil_id")
	}
	if !isCallerAdminOrDPO(caller, false) {
		h.logError(fmt.Sprintf("Unauthorized SAR export attempt by caller %s (staff %s)", caller.UserID, caller.StaffID))
		return nil, errors.New("UNAUTHORIZED_ADMIN_GDPR_SCOPE: SAR export requires System Admin / DPO authority")
	}

	coreSections, err := h.coreDAL.ExportCorePupilSAR(ctx, trimmedPupilID)
	if err != nil {
		return nil, err
	}

	allSections := make([]dal.SARSection, 0, len(coreSections)+4)
	allSections = append(allSections, coreSections...)

	for _, plugin := range h.listAttachedPluginsSorted() {
		if plugin.Manifest.GDPRHooks == nil || strings.TrimSpace(plugin.Manifest.GDPRHooks.ExportPupilData) == "" {
			return nil, fmt.Errorf("plugin %s attaches a database but is missing gdpr_hooks.export_pupil_data", plugin.Manifest.ID)
		}
		rawExport, err := h.invokePluginGDPRHook(
			ctx,
			plugin,
			strings.TrimSpace(plugin.Manifest.GDPRHooks.ExportPupilData),
			false,
			caller,
			trimmedPupilID,
		)
		if err != nil {
			return nil, err
		}
		pluginSecs, err := parsePluginSARSections(plugin, rawExport)
		if err != nil {
			return nil, err
		}
		allSections = append(allSections, pluginSecs...)
	}

	h.logInfo(fmt.Sprintf("Exported UK GDPR SAR package for pupil (%d sections) by staff %s", len(allSections), caller.StaffID))
	return &SARExportBundle{
		PupilID:         trimmedPupilID,
		SchoolURN:       h.cfg.SchoolURN,
		GeneratedAt:     time.Now().UTC().Format(time.RFC3339),
		GeneratedByUser: caller.UserID,
		Sections:        allSections,
	}, nil
}

func (h *Host) FinalizePupilSAR(ctx context.Context, caller CallerContext, in SARFinalizeInput) (*SARFinalizedPackage, error) {
	trimmedPupilID := strings.TrimSpace(in.PupilID)
	if trimmedPupilID == "" {
		return nil, errors.New("missing mandatory parameter: pupil_id")
	}
	if len(in.Sections) == 0 {
		return nil, errors.New("missing mandatory parameter: sections")
	}
	if !isCallerAdminOrDPO(caller, false) {
		return nil, errors.New("UNAUTHORIZED_ADMIN_GDPR_SCOPE: SAR redaction finalize requires System Admin / DPO authority")
	}

	// Verify pupil exists in core.db
	if _, err := h.coreDAL.ExportCorePupilSAR(ctx, trimmedPupilID); err != nil {
		return nil, err
	}

	h.logInfo(fmt.Sprintf("Finalized redacted UK GDPR SAR bundle for pupil by staff %s", caller.StaffID))
	return &SARFinalizedPackage{
		PupilID:           trimmedPupilID,
		SchoolURN:         h.cfg.SchoolURN,
		FinalizedAt:       time.Now().UTC().Format(time.RFC3339),
		RedactedByStaffID: caller.StaffID,
		RedactedByUserID:  caller.UserID,
		Status:            "SAR_EXPORT_FINALIZED",
		Sections:          in.Sections,
	}, nil
}

func (h *Host) PurgePupilData(ctx context.Context, caller CallerContext, pupilID string) (*RetentionPurgeResult, error) {
	trimmedPupilID := strings.TrimSpace(pupilID)
	if trimmedPupilID == "" {
		return nil, errors.New("missing mandatory parameter: pupil_id")
	}
	if !isCallerAdminOrDPO(caller, true) {
		h.logError(fmt.Sprintf("Unauthorized GDPR retention purge attempt by caller %s (staff %s)", caller.UserID, caller.StaffID))
		return nil, errors.New("UNAUTHORIZED_ADMIN_GDPR_SCOPE: statutory retention purge requires System Admin / DPO write authority")
	}

	// 1. Pre-flight check: enforce KCSIE safeguarding retention lock BEFORE touching any plugin or core table
	if err := h.coreDAL.CheckPupilRetentionPurgeEligible(ctx, trimmedPupilID); err != nil {
		h.logError(fmt.Sprintf("Blocked retention purge due to KCSIE retention guard: %v", err))
		return nil, err
	}

	// 2. Invoke gdpr_hooks.purge_pupil_data across all attached plugins (both ACTIVE and UNINSTALLED_DORMANT)
	pluginPurgedRows := make(map[string]int)
	for _, plugin := range h.listAttachedPluginsSorted() {
		if plugin.Manifest.GDPRHooks == nil || strings.TrimSpace(plugin.Manifest.GDPRHooks.PurgePupilData) == "" {
			return nil, fmt.Errorf("plugin %s attaches a database but is missing gdpr_hooks.purge_pupil_data", plugin.Manifest.ID)
		}
		rawPurge, err := h.invokePluginGDPRHook(
			ctx,
			plugin,
			strings.TrimSpace(plugin.Manifest.GDPRHooks.PurgePupilData),
			true,
			caller,
			trimmedPupilID,
		)
		if err != nil {
			return nil, err
		}
		purgedCount := 0
		if m, ok := rawPurge.(map[string]any); ok {
			if f, ok := m["purged_rows"].(float64); ok {
				purgedCount = int(f)
			}
		}
		pluginPurgedRows[plugin.Manifest.ID] = purgedCount
	}

	// 3. Purge pupil records across core.db
	if err := h.coreDAL.PurgeCorePupilData(ctx, trimmedPupilID); err != nil {
		return nil, err
	}

	h.logInfo(fmt.Sprintf("Completed statutory GDPR retention purge across Core and %d attached plugin(s) by staff %s", len(pluginPurgedRows), caller.StaffID))
	return &RetentionPurgeResult{
		PupilID:          trimmedPupilID,
		PurgedAt:         time.Now().UTC().Format(time.RFC3339),
		PurgedByStaffID:  caller.StaffID,
		CorePurged:       true,
		PluginPurgedRows: pluginPurgedRows,
		Status:           "RETENTION_PURGE_COMPLETED",
	}, nil
}

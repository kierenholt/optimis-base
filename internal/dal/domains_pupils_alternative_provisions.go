package dal

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

type PupilAlternativeProvisionSummary struct {
	ID                   string `json:"id"`
	ProvisionID          string `json:"provision_id"`
	PupilID              string `json:"pupil_id"`
	AcademicYearID       string `json:"academic_year_id"`
	ProviderURN          string `json:"provider_urn,omitempty"`
	ProviderUKPRN        string `json:"provider_ukprn,omitempty"`
	CompaniesHouseNumber string `json:"companies_house_number,omitempty"`
	APSettingTypeCode    string `json:"ap_setting_type_code,omitempty"`
	ReasonCode           string `json:"reason_code"`
	SENProvision         string `json:"sen_provision,omitempty"`
	SessionsPerWeek      int    `json:"sessions_per_week"`
	EffectiveFrom        string `json:"effective_from"`
	EffectiveTo          string `json:"effective_to,omitempty"`
}

func (d *CoreDAL) ListPupilAlternativeProvisionsScoped(
	ctx context.Context,
	scope DALQueryScope,
	pupilID string,
	academicYearID string,
) ([]PupilAlternativeProvisionSummary, error) {
	if err := validateScope(scope); err != nil {
		return nil, err
	}
	trimmedPupil := strings.TrimSpace(pupilID)
	if trimmedPupil != "" && scope.Mode == ScopeOwnGroups {
		if err := d.ensurePupilInOwnGroups(ctx, scope, trimmedPupil); err != nil {
			return nil, err
		}
	}
	trimmedAY := strings.TrimSpace(academicYearID)

	query := `
		SELECT id, pupil_id, academic_year_id,
		       COALESCE(provider_urn, ''), COALESCE(provider_ukprn, ''),
		       COALESCE(companies_house_number, ''), COALESCE(ap_setting_type_code, ''),
		       ap_reason_code, COALESCE(sen_provision_during_ap, 'N'),
		       sessions_per_week, effective_from, COALESCE(effective_to, '')
		FROM pupil_alternative_provisions WHERE 1=1`
	var args []any
	if trimmedPupil != "" {
		query += ` AND pupil_id = ?`
		args = append(args, trimmedPupil)
	}
	if trimmedAY != "" {
		query += ` AND academic_year_id = ?`
		args = append(args, trimmedAY)
	}
	query += ` ORDER BY effective_from DESC, id ASC`

	rows, err := d.readDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list pupil_alternative_provisions: %w", err)
	}
	defer rows.Close()

	result := make([]PupilAlternativeProvisionSummary, 0)
	for rows.Next() {
		var r PupilAlternativeProvisionSummary
		if err := rows.Scan(
			&r.ID, &r.PupilID, &r.AcademicYearID, &r.ProviderURN, &r.ProviderUKPRN,
			&r.CompaniesHouseNumber, &r.APSettingTypeCode,
			&r.ReasonCode, &r.SENProvision, &r.SessionsPerWeek, &r.EffectiveFrom, &r.EffectiveTo,
		); err != nil {
			return nil, fmt.Errorf("failed to scan pupil_alternative_provisions row: %w", err)
		}
		r.ProvisionID = r.ID
		result = append(result, r)
	}
	return result, rows.Err()
}

func (d *CoreDAL) UpsertPupilAlternativeProvisionScoped(
	ctx context.Context,
	scope DALQueryScope,
	in PupilAlternativeProvisionSummary,
) (*PupilAlternativeProvisionSummary, error) {
	if err := validateScope(scope); err != nil {
		return nil, err
	}
	if scope.Mode != ScopeAll {
		return nil, errors.New("access denied: upserting alternative provisions requires dal.pupils.write:all scope")
	}

	pupilID := strings.TrimSpace(in.PupilID)
	ayID := strings.TrimSpace(in.AcademicYearID)
	if ayID == "" {
		ayID = scope.AcademicYearID
	}
	reasonCode := strings.ToUpper(strings.TrimSpace(in.ReasonCode))
	effFrom := strings.TrimSpace(in.EffectiveFrom)
	if effFrom == "" {
		effFrom = scope.ReferenceDate
	}
	if pupilID == "" || reasonCode == "" || effFrom == "" {
		return nil, errors.New("missing mandatory parameter: pupil_id, reason_code, and effective_from are required")
	}
	provID := strings.TrimSpace(in.ProvisionID)
	if provID == "" {
		provID = strings.TrimSpace(in.ID)
	}
	if provID == "" {
		provID = "ap_" + strings.ReplaceAll(uuid.NewString(), "-", "")[:12]
	}
	sessPerWeek := in.SessionsPerWeek
	if sessPerWeek <= 0 {
		sessPerWeek = 1
	}
	senProv := strings.ToUpper(strings.TrimSpace(in.SENProvision))
	if senProv != "K" && senProv != "E" {
		senProv = "N"
	}

	if _, err := d.writeDB.ExecContext(
		ctx,
		`INSERT INTO pupil_alternative_provisions (
			id, pupil_id, academic_year_id, provider_urn, provider_ukprn,
			companies_house_number, ap_setting_type_code, ap_reason_code,
			sen_provision_during_ap, sessions_per_week, effective_from, effective_to
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			provider_urn = excluded.provider_urn,
			provider_ukprn = excluded.provider_ukprn,
			companies_house_number = excluded.companies_house_number,
			ap_setting_type_code = excluded.ap_setting_type_code,
			ap_reason_code = excluded.ap_reason_code,
			sen_provision_during_ap = excluded.sen_provision_during_ap,
			sessions_per_week = excluded.sessions_per_week,
			effective_from = excluded.effective_from,
			effective_to = excluded.effective_to`,
		provID, pupilID, ayID, optionalTrimmedString(in.ProviderURN), optionalTrimmedString(in.ProviderUKPRN),
		optionalTrimmedString(in.CompaniesHouseNumber), optionalTrimmedString(in.APSettingTypeCode),
		reasonCode, senProv, sessPerWeek, effFrom, optionalTrimmedString(in.EffectiveTo),
	); err != nil {
		return nil, fmt.Errorf("failed to upsert pupil_alternative_provisions row: %w", err)
	}

	out := in
	out.ID = provID
	out.ProvisionID = provID
	out.PupilID = pupilID
	out.AcademicYearID = ayID
	out.ReasonCode = reasonCode
	out.SENProvision = senProv
	out.SessionsPerWeek = sessPerWeek
	out.EffectiveFrom = effFrom
	return &out, nil
}

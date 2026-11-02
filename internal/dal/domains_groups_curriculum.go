package dal

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// ============================================================================
// Rooms
// ============================================================================

type RoomSummary struct {
	ID             string `json:"id"`
	Code           string `json:"code"`
	Name           string `json:"name"`
	RoomType       string `json:"room_type"`
	Capacity       int    `json:"capacity"`
	IsActive       bool   `json:"is_active"`
	PhoneExtension string `json:"phone_extension"`
}

func (d *CoreDAL) ListRooms(ctx context.Context) ([]RoomSummary, error) {
	rows, err := d.readDB.QueryContext(
		ctx,
		`SELECT id, code, name, room_type, capacity, is_active, COALESCE(phone_extension, '')
		 FROM rooms
		 ORDER BY code ASC, id ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list rooms: %w", err)
	}
	defer rows.Close()

	var result []RoomSummary
	for rows.Next() {
		var r RoomSummary
		var activeInt int
		if err := rows.Scan(&r.ID, &r.Code, &r.Name, &r.RoomType, &r.Capacity, &activeInt, &r.PhoneExtension); err != nil {
			return nil, fmt.Errorf("failed to scan room row: %w", err)
		}
		r.IsActive = activeInt == 1
		result = append(result, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rooms: %w", err)
	}
	return result, nil
}

func (d *CoreDAL) UpsertRoomScoped(
	ctx context.Context,
	scope DALQueryScope,
	in RoomSummary,
) (*RoomSummary, error) {
	if err := validateScope(scope); err != nil {
		return nil, err
	}
	if scope.Mode != ScopeAll {
		return nil, errors.New("access denied: upserting rooms requires dal.groups.write:all scope")
	}

	roomID := strings.TrimSpace(in.ID)
	code := strings.ToUpper(strings.TrimSpace(in.Code))
	name := strings.TrimSpace(in.Name)
	if code == "" || name == "" {
		return nil, errors.New("missing mandatory parameter: code and name are required")
	}
	if roomID == "" {
		roomID = "rm_" + strings.ToLower(code)
	}
	roomType := strings.ToUpper(strings.TrimSpace(in.RoomType))
	if roomType == "" {
		roomType = "CLASSROOM"
	}
	capVal := in.Capacity
	if capVal <= 0 {
		capVal = 30
	}

	if _, err := d.writeDB.ExecContext(
		ctx,
		`INSERT INTO rooms (id, code, name, room_type, capacity, is_active, phone_extension)
		 VALUES (?, ?, ?, ?, ?, 1, ?)
		 ON CONFLICT(id) DO UPDATE SET
		     code = excluded.code,
		     name = excluded.name,
		     room_type = excluded.room_type,
		     capacity = excluded.capacity,
		     phone_extension = excluded.phone_extension`,
		roomID, code, name, roomType, capVal, strings.TrimSpace(in.PhoneExtension),
	); err != nil {
		return nil, fmt.Errorf("failed to upsert rooms row: %w", err)
	}

	return &RoomSummary{
		ID:             roomID,
		Code:           code,
		Name:           name,
		RoomType:       roomType,
		Capacity:       capVal,
		IsActive:       true,
		PhoneExtension: strings.TrimSpace(in.PhoneExtension),
	}, nil
}

// ============================================================================
// Courses
// ============================================================================

type CourseSummary struct {
	ID             string `json:"id"`
	AcademicYearID string `json:"academic_year_id"`
	DepartmentID   string `json:"department_id"`
	Code           string `json:"code"`
	Name           string `json:"name"`
	DfESubjectCode string `json:"dfe_subject_code"`
}

func (d *CoreDAL) ListCourses(ctx context.Context, academicYearID string) ([]CourseSummary, error) {
	ayID := strings.TrimSpace(academicYearID)
	query := `SELECT id, academic_year_id, COALESCE(department_id, ''), code, name, COALESCE(dfe_subject_code, '')
	          FROM courses`
	var args []any
	if ayID != "" {
		query += ` WHERE academic_year_id = ?`
		args = append(args, ayID)
	}
	query += ` ORDER BY code ASC, id ASC`

	rows, err := d.readDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list courses: %w", err)
	}
	defer rows.Close()

	var result []CourseSummary
	for rows.Next() {
		var c CourseSummary
		if err := rows.Scan(&c.ID, &c.AcademicYearID, &c.DepartmentID, &c.Code, &c.Name, &c.DfESubjectCode); err != nil {
			return nil, fmt.Errorf("failed to scan course row: %w", err)
		}
		result = append(result, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating courses: %w", err)
	}
	return result, nil
}

func (d *CoreDAL) UpsertCourseScoped(
	ctx context.Context,
	scope DALQueryScope,
	in CourseSummary,
) (*CourseSummary, error) {
	if err := validateScope(scope); err != nil {
		return nil, err
	}
	if scope.Mode != ScopeAll {
		return nil, errors.New("access denied: upserting courses requires dal.groups.write:all scope")
	}

	courseID := strings.TrimSpace(in.ID)
	ayID := strings.TrimSpace(in.AcademicYearID)
	if ayID == "" {
		ayID = scope.AcademicYearID
	}
	code := strings.ToUpper(strings.TrimSpace(in.Code))
	name := strings.TrimSpace(in.Name)
	if code == "" || name == "" {
		return nil, errors.New("missing mandatory parameter: code and name are required")
	}
	if courseID == "" {
		courseID = "crs_" + strings.ReplaceAll(uuid.NewString(), "-", "")[:12]
	}
	var depArg any
	if strings.TrimSpace(in.DepartmentID) != "" {
		depArg = strings.TrimSpace(in.DepartmentID)
	}

	if _, err := d.writeDB.ExecContext(
		ctx,
		`INSERT INTO courses (id, academic_year_id, department_id, code, name, dfe_subject_code)
		 VALUES (?, ?, ?, ?, ?, ?)
		 ON CONFLICT(academic_year_id, code) DO UPDATE SET
		     department_id = COALESCE(excluded.department_id, courses.department_id),
		     name = excluded.name,
		     dfe_subject_code = excluded.dfe_subject_code`,
		courseID, ayID, depArg, code, name, strings.TrimSpace(in.DfESubjectCode),
	); err != nil {
		return nil, fmt.Errorf("failed to upsert courses row: %w", err)
	}

	var persistedID string
	if err := d.readDB.QueryRowContext(ctx, `SELECT id FROM courses WHERE academic_year_id = ? AND code = ?`, ayID, code).Scan(&persistedID); err == nil && persistedID != "" {
		courseID = persistedID
	}

	return &CourseSummary{
		ID:             courseID,
		AcademicYearID: ayID,
		DepartmentID:   strings.TrimSpace(in.DepartmentID),
		Code:           code,
		Name:           name,
		DfESubjectCode: strings.TrimSpace(in.DfESubjectCode),
	}, nil
}

// ============================================================================
// Departments
// ============================================================================

type DepartmentSummary struct {
	ID                      string `json:"id"`
	Name                    string `json:"name"`
	NameShort               string `json:"name_short"`
	Type                    string `json:"type"`
	HeadOfDepartmentStaffID string `json:"head_of_department_staff_id"`
}

func (d *CoreDAL) ListDepartments(ctx context.Context) ([]DepartmentSummary, error) {
	rows, err := d.readDB.QueryContext(
		ctx,
		`SELECT id, name, name_short, type, COALESCE(head_of_department_staff_id, '')
		 FROM departments
		 ORDER BY name ASC, id ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list departments: %w", err)
	}
	defer rows.Close()

	var result []DepartmentSummary
	for rows.Next() {
		var dep DepartmentSummary
		if err := rows.Scan(&dep.ID, &dep.Name, &dep.NameShort, &dep.Type, &dep.HeadOfDepartmentStaffID); err != nil {
			return nil, fmt.Errorf("failed to scan department row: %w", err)
		}
		result = append(result, dep)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating departments: %w", err)
	}
	return result, nil
}

func (d *CoreDAL) UpsertDepartmentScoped(
	ctx context.Context,
	scope DALQueryScope,
	in DepartmentSummary,
) (*DepartmentSummary, error) {
	if err := validateScope(scope); err != nil {
		return nil, err
	}
	if scope.Mode != ScopeAll {
		return nil, errors.New("access denied: upserting departments requires dal.groups.write:all scope")
	}

	depID := strings.TrimSpace(in.ID)
	name := strings.TrimSpace(in.Name)
	nameShort := strings.ToUpper(strings.TrimSpace(in.NameShort))
	if nameShort == "" && len(name) >= 3 {
		nameShort = strings.ToUpper(name[:3])
	}
	if name == "" || nameShort == "" {
		return nil, errors.New("missing mandatory parameter: name and name_short/code are required")
	}
	if depID == "" {
		depID = "dep_" + strings.ToLower(nameShort)
	}
	depType := strings.ToUpper(strings.TrimSpace(in.Type))
	switch depType {
	case "", "ACADEMIC", "LEARNING_AREA":
		depType = "LEARNING_AREA"
	case "ADMINISTRATION", "PASTORAL":
	default:
		return nil, fmt.Errorf("invalid department type %q", in.Type)
	}
	var hodArg any
	if strings.TrimSpace(in.HeadOfDepartmentStaffID) != "" {
		hodArg = strings.TrimSpace(in.HeadOfDepartmentStaffID)
	}

	if _, err := d.writeDB.ExecContext(
		ctx,
		`INSERT INTO departments (id, name, name_short, type, head_of_department_staff_id)
		 VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET
		     name = excluded.name,
		     name_short = excluded.name_short,
		     type = excluded.type,
		     head_of_department_staff_id = COALESCE(excluded.head_of_department_staff_id, departments.head_of_department_staff_id)`,
		depID, name, nameShort, depType, hodArg,
	); err != nil {
		return nil, fmt.Errorf("failed to upsert departments row: %w", err)
	}

	return &DepartmentSummary{
		ID:                      depID,
		Name:                    name,
		NameShort:               nameShort,
		Type:                    depType,
		HeadOfDepartmentStaffID: strings.TrimSpace(in.HeadOfDepartmentStaffID),
	}, nil
}

// ============================================================================
// Year Groups
// ============================================================================

type YearGroupSummary struct {
	ID                string `json:"id"`
	NCCode            string `json:"nc_code"`
	Name              string `json:"name"`
	Phase             string `json:"phase"`
	KeyStage          string `json:"key_stage"`
	SequenceNumber    int    `json:"sequence_number"`
	HeadOfYearStaffID string `json:"head_of_year_staff_id"`
}

func (d *CoreDAL) ListYearGroups(ctx context.Context) ([]YearGroupSummary, error) {
	rows, err := d.readDB.QueryContext(
		ctx,
		`SELECT id, nc_code, name, phase, key_stage, sequence_number, COALESCE(head_of_year_staff_id, '')
		 FROM year_groups
		 ORDER BY sequence_number ASC, id ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list year_groups: %w", err)
	}
	defer rows.Close()

	var result []YearGroupSummary
	for rows.Next() {
		var yg YearGroupSummary
		if err := rows.Scan(&yg.ID, &yg.NCCode, &yg.Name, &yg.Phase, &yg.KeyStage, &yg.SequenceNumber, &yg.HeadOfYearStaffID); err != nil {
			return nil, fmt.Errorf("failed to scan year_group row: %w", err)
		}
		result = append(result, yg)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating year_groups: %w", err)
	}
	return result, nil
}

func (d *CoreDAL) UpsertYearGroupScoped(
	ctx context.Context,
	scope DALQueryScope,
	in YearGroupSummary,
) (*YearGroupSummary, error) {
	if err := validateScope(scope); err != nil {
		return nil, err
	}
	if scope.Mode != ScopeAll {
		return nil, errors.New("access denied: upserting year groups requires dal.groups.write:all scope")
	}

	ygID := strings.TrimSpace(in.ID)
	ncCode := strings.ToUpper(strings.TrimSpace(in.NCCode))
	name := strings.TrimSpace(in.Name)
	if ncCode == "" || name == "" {
		return nil, errors.New("missing mandatory parameter: nc_code and name are required")
	}
	if ygID == "" {
		ygID = "yg_" + strings.ToLower(ncCode)
	}
	phase := strings.ToUpper(strings.TrimSpace(in.Phase))
	if phase == "" {
		phase = "SECONDARY"
	}
	keyStage := strings.ToUpper(strings.TrimSpace(in.KeyStage))
	if keyStage == "" {
		keyStage = "KS3"
	}
	seq := in.SequenceNumber
	if seq <= 0 {
		_ = d.readDB.QueryRowContext(ctx, `SELECT COALESCE(MAX(sequence_number), 0) + 1 FROM year_groups`).Scan(&seq)
		if seq <= 0 {
			seq = 1
		}
	}
	var hoyArg any
	if strings.TrimSpace(in.HeadOfYearStaffID) != "" {
		hoyArg = strings.TrimSpace(in.HeadOfYearStaffID)
	}

	if _, err := d.writeDB.ExecContext(
		ctx,
		`INSERT INTO year_groups (id, nc_code, name, phase, key_stage, sequence_number, head_of_year_staff_id)
		 VALUES (?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET
		     nc_code = excluded.nc_code,
		     name = excluded.name,
		     phase = excluded.phase,
		     key_stage = excluded.key_stage,
		     sequence_number = excluded.sequence_number,
		     head_of_year_staff_id = COALESCE(excluded.head_of_year_staff_id, year_groups.head_of_year_staff_id)`,
		ygID, ncCode, name, phase, keyStage, seq, hoyArg,
	); err != nil {
		return nil, fmt.Errorf("failed to upsert year_groups row: %w", err)
	}

	return &YearGroupSummary{
		ID:                ygID,
		NCCode:            ncCode,
		Name:              name,
		Phase:             phase,
		KeyStage:          keyStage,
		SequenceNumber:    seq,
		HeadOfYearStaffID: strings.TrimSpace(in.HeadOfYearStaffID),
	}, nil
}

// ============================================================================
// Houses
// ============================================================================

type HouseSummary struct {
	ID             string `json:"id"`
	HouseID        string `json:"house_id"`
	Code           string `json:"code"`
	NameShort      string `json:"name_short"`
	Name           string `json:"name"`
	ColorHex       string `json:"color_hex"`
	SequenceNumber int    `json:"sequence_number,omitempty"`
}

func (d *CoreDAL) ListHouses(ctx context.Context) ([]HouseSummary, error) {
	rows, err := d.readDB.QueryContext(
		ctx,
		`SELECT id, name, name_short, color_hex
		 FROM houses
		 ORDER BY name ASC, id ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list houses: %w", err)
	}
	defer rows.Close()

	result := make([]HouseSummary, 0)
	seq := 1
	for rows.Next() {
		var h HouseSummary
		if err := rows.Scan(&h.ID, &h.Name, &h.NameShort, &h.ColorHex); err != nil {
			return nil, fmt.Errorf("failed to scan house row: %w", err)
		}
		h.HouseID = h.ID
		h.Code = h.NameShort
		h.SequenceNumber = seq
		seq++
		result = append(result, h)
	}
	return result, rows.Err()
}

func (d *CoreDAL) UpsertHouseScoped(
	ctx context.Context,
	scope DALQueryScope,
	in HouseSummary,
) (*HouseSummary, error) {
	if err := validateScope(scope); err != nil {
		return nil, err
	}
	if scope.Mode != ScopeAll {
		return nil, errors.New("access denied: upserting houses requires dal.groups.write:all scope")
	}

	houseID := strings.TrimSpace(in.HouseID)
	if houseID == "" {
		houseID = strings.TrimSpace(in.ID)
	}
	if houseID == "" {
		houseID = "hse_" + strings.ReplaceAll(uuid.NewString(), "-", "")[:12]
	}
	name := strings.TrimSpace(in.Name)
	nameShort := strings.ToUpper(strings.TrimSpace(in.NameShort))
	if nameShort == "" {
		nameShort = strings.ToUpper(strings.TrimSpace(in.Code))
	}
	if nameShort == "" && len(name) >= 3 {
		nameShort = strings.ToUpper(name[:3])
	}
	if name == "" || nameShort == "" {
		return nil, errors.New("missing mandatory parameter: name and code/name_short are required")
	}
	colorHex := strings.TrimSpace(in.ColorHex)
	if colorHex == "" {
		colorHex = "#1E40AF"
	}

	if _, err := d.writeDB.ExecContext(
		ctx,
		`INSERT INTO houses (id, name, name_short, color_hex)
		 VALUES (?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET
		     name = excluded.name,
		     name_short = excluded.name_short,
		     color_hex = excluded.color_hex`,
		houseID, name, nameShort, colorHex,
	); err != nil {
		return nil, fmt.Errorf("failed to upsert houses row: %w", err)
	}

	seq := in.SequenceNumber
	if seq <= 0 {
		seq = 1
	}
	return &HouseSummary{
		ID:             houseID,
		HouseID:        houseID,
		Code:           nameShort,
		NameShort:      nameShort,
		Name:           name,
		ColorHex:       colorHex,
		SequenceNumber: seq,
	}, nil
}

CREATE TABLE IF NOT EXISTS markbook_columns (
    id TEXT PRIMARY KEY,
    group_id TEXT NOT NULL,
    academic_year_id TEXT NOT NULL,
    title TEXT NOT NULL,
    grading_scale TEXT NOT NULL CHECK (grading_scale IN ('9-1', 'WT,EXS,GDS')),
    mat_schema_version TEXT NOT NULL,
    assessment_date TEXT NOT NULL,
    created_by_staff_id TEXT NOT NULL,
    created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_markbook_columns_group
    ON markbook_columns(group_id, academic_year_id);

CREATE TABLE IF NOT EXISTS markbook_grades (
    id TEXT PRIMARY KEY,
    column_id TEXT NOT NULL REFERENCES markbook_columns(id),
    group_id TEXT NOT NULL,
    pupil_id TEXT NOT NULL,
    academic_year_id TEXT NOT NULL,
    grade TEXT NOT NULL,
    grade_points REAL NOT NULL,
    teacher_comment TEXT NOT NULL DEFAULT '',
    recorded_by_staff_id TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE(column_id, pupil_id)
);

CREATE INDEX IF NOT EXISTS idx_markbook_grades_pupil
    ON markbook_grades(pupil_id);

CREATE INDEX IF NOT EXISTS idx_markbook_grades_group_column
    ON markbook_grades(group_id, column_id);

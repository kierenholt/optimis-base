CREATE TABLE IF NOT EXISTS pupil_notes (
  id TEXT PRIMARY KEY,
  pupil_id TEXT NOT NULL,
  note TEXT NOT NULL,
  created_by TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_pupil_notes_pupil_id ON pupil_notes(pupil_id);

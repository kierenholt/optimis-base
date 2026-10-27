# Test Templates & Reference Patterns

## 1. Playwright E2E Test Template (Sequential + Log Tapping + `data-testid` + Setup/Teardown)

```typescript
import { test, expect } from '@playwright/test';

// All tests must run sequentially
test.describe.configure({ mode: 'serial' });

test.describe('Pupil Attendance Register', () => {
  let seededPupilId: string;
  let browserLogs: string[] = [];
  let pageErrors: string[] = [];
  let logCursor: string;

  test.beforeEach(async ({ page, request }) => {
    browserLogs = [];
    pageErrors = [];

    // 1. Tap Browser Console & Uncaught Page Errors
    page.on('console', (msg) => {
      browserLogs.push(`[browser:${msg.type()}] ${msg.text()}`);
    });
    page.on('pageerror', (err) => {
      pageErrors.push(`[pageerror] ${err.stack || err.message}`);
    });

    // 2. Tap Backend Service Logs (record cursor before test execution)
    const cursorRes = await request.get('/api/test/logs/cursor');
    if (!cursorRes.ok()) {
      throw new Error(`Failed to obtain service log cursor: ${cursorRes.status()}`);
    }
    logCursor = (await cursorRes.json()).cursor;

    // 3. Seed required test data explicitly during setup
    const res = await request.post('/api/test/seed-pupil', {
      data: { upn: 'A123456789012', first_name: 'Ada', last_name: 'Lovelace' },
    });
    if (!res.ok()) {
      throw new Error(`Failed to seed pupil fixture: ${res.status()}`);
    }
    const body = await res.json();
    seededPupilId = body.pupil_id;
  });

  test.afterEach(async ({ request }, testInfo) => {
    // 1. Fetch Backend Service Logs emitted during this test
    if (logCursor) {
      const logsRes = await request.get(`/api/test/logs?since=${encodeURIComponent(logCursor)}`);
      if (logsRes.ok()) {
        const serviceLogs: string[] = (await logsRes.json()).entries;
        if (testInfo.status !== testInfo.expectedStatus || serviceLogs.some((l) => l.includes('ERROR'))) {
          console.log('--- TAPPED SERVICE LOGS ---\n' + serviceLogs.join('\n'));
        }
      }
    }

    // 2. Dump tapped Browser Console logs on failure or error
    if (testInfo.status !== testInfo.expectedStatus || pageErrors.length > 0) {
      console.log('--- TAPPED BROWSER CONSOLE ---\n' + browserLogs.join('\n'));
      console.log('--- TAPPED PAGE ERRORS ---\n' + pageErrors.join('\n'));
    }

    // 3. Always clean up seeded mock data at the end of the test
    if (seededPupilId) {
      await request.delete(`/api/test/seed-pupil/${seededPupilId}`);
    }

    expect(pageErrors).toEqual([]);
  });

  test('markAttendance_unmarkedAMSession_savesPresentMark_test', async ({ page }) => {
    await page.goto(`/registers/session/AM?pupil=${seededPupilId}`);

    // Temporary layout inspection pattern (never use the browser tool directly):
    // const html = await page.content();
    // console.log(html);

    const row = page.getByTestId(`attendance-row-${seededPupilId}`);
    await expect(row).toBeVisible();

    await row.getByTestId('mark-present-btn').click();
    await page.getByTestId('save-register-btn').click();

    await expect(row.getByTestId('attendance-status-badge')).toHaveText('Present (/)');
  });
});
```

---

## 2. Go Host / Core DAL Test Template (With Service Log & Ring Buffer Tap)

```go
package dal_test

import (
	"context"
	"testing"
)

func TestGetGuardiansForPupil_CourtOrderBlockPresent_ExcludesRestrictedGuardian_test(t *testing.T) {
	ctx := context.Background()
	db, logTap := setupIsolatedTestDBWithLogTap(t)
	t.Cleanup(func() {
		if t.Failed() {
			t.Logf("--- TAPPED SERVICE & DIAGNOSTIC LOGS ---\n%s", logTap.Dump())
		}
		db.Close()
	})

	// Seed pupil and two guardians (one with a court-order block)
	pupilID := seedTestPupil(t, db, "U987654321000")
	allowedGuardianID := seedTestGuardian(t, db, pupilID, true, false)
	_ = seedTestGuardian(t, db, pupilID, true, true) // court_order_block = true

	guardians, err := db.DAL().Guardians().ListAuthorizedForPupil(ctx, pupilID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(guardians) != 1 {
		t.Fatalf("expected exactly 1 authorized guardian, got %d", len(guardians))
	}
	if guardians[0].ID != allowedGuardianID {
		t.Errorf("expected guardian %s, got %s", allowedGuardianID, guardians[0].ID)
	}
}
```

---

## 3. Lua Plugin GDPR Hook Verification Pattern

```lua
-- Inside plugin test suite verified by `mis-sdk verify`
function exportAndPurgePupilData_withSeededPluginRows_leavesZeroOrphanedRecords_test(ctx)
  local pupil_id = "pupil_test_001"

  -- 1. Seed plugin-specific data
  ctx.plugin_db.exec(
    "INSERT INTO merit_points (id, pupil_id, points, note) VALUES (?, ?, ?, ?)",
    { "m1", pupil_id, 5, "Helped peer in maths" }
  )

  -- 2. Verify SAR export hook returns redactable sections
  local exported = gdpr_hooks.export_pupil_data(ctx, pupil_id)
  assert(exported.ok == true, "export_pupil_data must return ok = true")
  assert(#exported.data == 1, "expected 1 exported section")
  assert(#exported.data[1].records == 1, "expected 1 exported record")

  -- 3. Verify purge hook deletes all pupil records with zero orphans
  local purged = gdpr_hooks.purge_pupil_data(ctx, pupil_id)
  assert(purged.ok == true, "purge_pupil_data must return ok = true")

  local remaining = ctx.plugin_db.query(
    "SELECT COUNT(*) AS cnt FROM merit_points WHERE pupil_id = ?",
    { pupil_id }
  )
  assert(remaining[1].cnt == 0, "expected 0 remaining rows after purge_pupil_data")
end
```

import React, { useEffect, useState } from 'react';
import { checkCallerAdminOrDpo } from '../components/routeUtils';
import { InitContextOverrides } from '../components/iframeMessageHandlers';
import {
  ActiveCallerIdentity,
  DecryptedSafeguardingNote,
  PinnedRecentEntry,
  PluginNavEntry,
  SafeguardingNoteEnvelope,
  SARSection,
  StaffPersonaSummary,
  ThemeMode,
} from '../themeAndTypes';

export interface UseHostAdminAndDslStateParams {
  activePupilId: string;
  activeStaffId: string;
  setActiveStaffId: React.Dispatch<React.SetStateAction<string>>;
  activeCaller: ActiveCallerIdentity | null;
  setActiveCaller: React.Dispatch<React.SetStateAction<ActiveCallerIdentity | null>>;
  setPinnedRecents?: React.Dispatch<React.SetStateAction<PinnedRecentEntry[]>>;
  applyInstalledPluginsState: (
    plugins: PluginNavEntry[],
    serverEntityHandlers?: Record<string, string>
  ) => void;
  refreshWorkspaceForStaff: (staffIdOverride?: string) => Promise<PluginNavEntry[] | undefined>;
  broadcastInitToMountedIframes: (
    overrideContext?: Partial<InitContextOverrides>,
    overrideThemeMode?: ThemeMode,
    overridePlugins?: PluginNavEntry[],
    overrideCaller?: ActiveCallerIdentity | null
  ) => void;
}

export function useHostAdminAndDslState(params: UseHostAdminAndDslStateParams) {
  const {
    activePupilId,
    activeStaffId,
    setActiveStaffId,
    activeCaller,
    setActiveCaller,
    broadcastInitToMountedIframes,
  } = params;

  // DSL Safeguarding Vault state (Section 5B)
  const [hasSafeguardingRead, setHasSafeguardingRead] = useState<boolean>(false);
  const [safeguardingEnvelopes, setSafeguardingEnvelopes] = useState<SafeguardingNoteEnvelope[]>([]);
  const [cloudKeyToken, setCloudKeyToken] = useState<string>('');
  const [decryptedNotes, setDecryptedNotes] = useState<Record<string, DecryptedSafeguardingNote>>({});
  const [dslVaultError, setDslVaultError] = useState<string>('');

  // UK GDPR SAR Redaction & Retention Purge state (Sections 5C, 7B & 9.3)
  const [hasAdminOrDpoAccess, setHasAdminOrDpoAccess] = useState<boolean>(false);
  const [sarSections, setSarSections] = useState<SARSection[]>([]);
  const [sarExportStatus, setSarExportStatus] = useState<string>('');
  const [sarFinalizedJson, setSarFinalizedJson] = useState<string>('');
  const [sarPurgeStatus, setSarPurgeStatus] = useState<string>('');

  // Day-2 Operations: Admin-Exported PII-Scrubbed Diagnostic Bundle state (Section 8 Gap 1)
  const [diagnosticBundleStatus, setDiagnosticBundleStatus] = useState<string>('');
  const [diagnosticBundleJson, setDiagnosticBundleJson] = useState<string>('');
  const [diagnosticBundleLog, setDiagnosticBundleLog] = useState<string>('');

  useEffect(() => {
    const loadSessionAndSafeguarding = async () => {
      try {
        const sessRes = await fetch('/api/auth/session');
        if (!sessRes.ok) {
          return;
        }
        const sessBody = await sessRes.json();
        const caller = sessBody?.caller;
        const scopes: string[] = Array.isArray(caller?.scopes) ? caller.scopes : [];
        const roles: string[] = Array.isArray(caller?.active_roles) ? caller.active_roles : [];
        if (caller && typeof caller.staff_id === 'string') {
          const nextCaller: ActiveCallerIdentity = {
            staff_id: caller.staff_id,
            user_id: caller.user_id || '',
            active_roles: roles,
            scopes,
          };
          setActiveCaller(nextCaller);
          setActiveStaffId((prev) => prev || caller.staff_id);
          broadcastInitToMountedIframes(
            { staff_id: caller.staff_id },
            undefined,
            undefined,
            nextCaller
          );
        }
        setHasAdminOrDpoAccess(checkCallerAdminOrDpo(roles, scopes));

        if (scopes.includes('dal.safeguarding.read')) {
          setHasSafeguardingRead(true);
          const notesParams = new URLSearchParams();
          if (activePupilId) {
            notesParams.set('pupil_id', activePupilId);
          }
          const query = notesParams.toString() ? `?${notesParams.toString()}` : '';
          const notesRes = await fetch(`/api/safeguarding/notes${query}`);
          if (notesRes.ok) {
            const notesBody = await notesRes.json();
            if (notesBody.ok && Array.isArray(notesBody.notes)) {
              setSafeguardingEnvelopes(notesBody.notes);
            }
          }
        } else {
          setHasSafeguardingRead(false);
          setSafeguardingEnvelopes([]);
          setDecryptedNotes({});
        }
      } catch {
        // Ignore session fetch errors if not in DSL flow
      }
    };
    void loadSessionAndSafeguarding();
  }, [activePupilId, activeStaffId]);

  const handleRenewCloudKey = async () => {
    setDslVaultError('');
    const res = await fetch('/api/safeguarding/cloud-key/renew', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ ttl_seconds: 3600 }),
    });
    const body = await res.json();
    if (!res.ok || !body.ok) {
      setDslVaultError(body.error_code || 'Failed to renew DSL Cloud Key');
      return;
    }
    setCloudKeyToken(body.cloud_key_token);
  };

  const handleDecryptSafeguardingNote = async (noteId: string) => {
    setDslVaultError('');
    const res = await fetch(`/api/safeguarding/notes/${encodeURIComponent(noteId)}/decrypt`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ cloud_key_token: cloudKeyToken }),
    });
    const body = await res.json();
    if (!res.ok || !body.ok) {
      setDslVaultError(body.error_code || 'Failed to decrypt safeguarding note');
      return;
    }
    setDecryptedNotes((prev) => ({
      ...prev,
      [noteId]: body.note,
    }));
  };

  const handleLoadSarExport = async () => {
    if (!activePupilId) {
      throw new Error('Missing mandatory pupil_id for SAR export');
    }
    const res = await fetch('/api/admin/gdpr/sar/export', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ pupil_id: activePupilId }),
    });
    const body = await res.json();
    if (!res.ok || !body.ok) {
      setSarExportStatus(`ERROR:${body.error_code || res.status}`);
      return;
    }
    setSarSections(body.bundle.sections || []);
    setSarExportStatus('SAR_LOADED');
    setSarFinalizedJson('');
  };

  const handleUpdateRedactableField = (
    secIdx: number,
    recIdx: number,
    fieldKey: string,
    newValue: string
  ) => {
    setSarSections((prev) =>
      prev.map((sec, sIdx) => {
        if (sIdx !== secIdx) {
          return sec;
        }
        return {
          ...sec,
          records: sec.records.map((rec, rIdx) => {
            if (rIdx !== recIdx) {
              return rec;
            }
            return {
              ...rec,
              [fieldKey]: newValue,
            };
          }),
        };
      })
    );
  };

  const handleFinalizeSarExport = async () => {
    if (!activePupilId) {
      throw new Error('Missing mandatory pupil_id for SAR finalization');
    }
    const res = await fetch('/api/admin/gdpr/sar/finalize', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        pupil_id: activePupilId,
        sections: sarSections,
      }),
    });
    const body = await res.json();
    if (!res.ok || !body.ok) {
      setSarExportStatus(`ERROR:${body.error_code || res.status}`);
      return;
    }
    setSarExportStatus(body.package.status || 'SAR_EXPORT_FINALIZED');
    setSarFinalizedJson(JSON.stringify(body.package, null, 2));
  };

  const handlePurgePupilData = async () => {
    if (!activePupilId) {
      throw new Error('Missing mandatory pupil_id for retention purge');
    }
    const res = await fetch('/api/admin/gdpr/purge', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        pupil_id: activePupilId,
      }),
    });
    const body = await res.json();
    if (!res.ok || !body.ok) {
      setSarPurgeStatus(`ERROR:${body.error_code || res.status}`);
      return;
    }
    setSarPurgeStatus(body.result.status || 'RETENTION_PURGE_COMPLETED');
    setSarSections([]);
  };

  const handleDownloadDiagnosticBundle = async () => {
    const res = await fetch('/api/admin/diagnostics/bundle');
    const body = await res.json();
    if (!res.ok || !body.ok || !body.bundle) {
      setDiagnosticBundleStatus(`ERROR:${body?.error_code || res.status}`);
      return;
    }
    const formattedJson = JSON.stringify(body.bundle, null, 2);
    setDiagnosticBundleJson(formattedJson);
    setDiagnosticBundleLog(typeof body.bundle.log_archive === 'string' ? body.bundle.log_archive : '');
    setDiagnosticBundleStatus('DIAGNOSTIC_BUNDLE_EXPORTED');
  };

  return {
    activeCaller,
    hasSafeguardingRead,
    safeguardingEnvelopes,
    cloudKeyToken,
    setCloudKeyToken,
    decryptedNotes,
    dslVaultError,
    handleRenewCloudKey,
    handleDecryptSafeguardingNote,
    hasAdminOrDpoAccess,
    sarSections,
    sarExportStatus,
    sarFinalizedJson,
    sarPurgeStatus,
    handleLoadSarExport,
    handleUpdateRedactableField,
    handleFinalizeSarExport,
    handlePurgePupilData,
    diagnosticBundleStatus,
    diagnosticBundleJson,
    diagnosticBundleLog,
    handleDownloadDiagnosticBundle,
  };
}

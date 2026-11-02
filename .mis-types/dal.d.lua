---@meta
-- Optimis School MIS — Lua 5.1 Plugin Runtime Type Annotations (LuaLS / EmmyLua)
-- Architecture Blueprint: Sections 9.2 & 10B (Zero-Config .vscode/ Workspace & .mis-types/dal.d.lua)
-- Pre-bound via .vscode/settings.json for instant IDE autocomplete and inline linting.

---@class OptimisCallerContext
---@field type "USER"|"SYSTEM"
---@field user_id string
---@field staff_id string
---@field request_id string
---@field active_roles string[]

---@class OptimisSchoolContext
---@field urn string
---@field academic_year_id string
---@field reference_date string
---@field mat_id string
---@field mat_grading_schema_version string
---@field mat_grading_schema table|nil


---@class OptimisAcademicDAL
---@field list_years fun(): OptimisAcademicYearRecord[]
---@field upsert_year fun(params: table): OptimisAcademicYearRecord
---@field list_terms fun(academic_year_id_or_opts?: string|table): OptimisAcademicTermRecord[]
---@field upsert_term fun(params: table): OptimisAcademicTermRecord
---@field list_days_of_week fun(): OptimisDayOfWeekRecord[]
---@field upsert_day_of_week fun(params: table): OptimisDayOfWeekRecord
---@field list_special_days fun(term_id_or_opts?: string|table): OptimisAcademicSpecialDayRecord[]
---@field upsert_special_day fun(params: table): OptimisAcademicSpecialDayRecord
---@field list_alert_levels fun(): OptimisAlertLevelRecord[]

---@class OptimisPupilsDAL
---@field get_by_id fun(pupil_id: string|table, opts?: table): OptimisPupilRecord
---@field get_by_ids_batch fun(pupil_ids: string[]|table, opts?: table): OptimisPupilRecord[]
---@field get_batch fun(pupil_ids: string[]|table, opts?: table): OptimisPupilRecord[]
---@field list_all fun(opts?: table): OptimisPupilRecord[]
---@field list_by_group fun(group_id: string, opts?: table): OptimisPupilRecord[]
---@field list_fsm_periods_batch fun(pupil_ids: string[]|table, opts?: table): table<string, OptimisFSMPeriodSummary[]>
---@field upsert_fsm_period fun(params: table, opts?: table): OptimisFSMPeriodSummary
---@field list_exclusions fun(pupil_id_or_opts?: string|table, opts?: table): OptimisPupilExclusionRecord[]
---@field upsert_exclusion fun(params: table, opts?: table): OptimisPupilExclusionRecord
---@field list_alternative_provisions fun(pupil_id_or_opts?: string|table, opts?: table): OptimisPupilAlternativeProvisionRecord[]
---@field upsert_alternative_provision fun(params: table, opts?: table): OptimisPupilAlternativeProvisionRecord
---@field list_post16_learning_aims fun(pupil_id_or_opts?: string|table, opts?: table): OptimisPupilPost16LearningAimRecord[]
---@field upsert_post16_learning_aim fun(params: table, opts?: table): OptimisPupilPost16LearningAimRecord
---@field list_pastoral_notes fun(pupil_id_or_opts?: string|table, opts?: table): OptimisPupilPastoralNoteRecord[]
---@field upsert_pastoral_note fun(params: table, opts?: table): OptimisPupilPastoralNoteRecord
---@field list_sen_and_medical_batch fun(pupil_ids: string[]|table, opts?: table): table<string, OptimisSenAndMedicalSummary>
---@field list_sen_register fun(opts?: table): OptimisSenAndMedicalSummary[]
---@field upsert_pupil_admission fun(params: table, opts?: table): OptimisPupilRecord
---@field upsert_year_enrolment fun(params: table, opts?: table): OptimisPupilRecord
---@field promote_pupil_enrolment fun(params: table, opts?: table): OptimisPupilRecord

---@class OptimisGroupsDAL
---@field assert_caller_assigned fun(group_id: string|table, opts?: table): boolean
---@field can_mutate_group fun(group_id: string|table, opts?: table): boolean
---@field is_caller_assigned fun(group_id: string|table, opts?: table): boolean
---@field list_groups fun(opts?: table): OptimisGroupSummary[]
---@field list_my_groups fun(opts?: table): OptimisGroupSummary[]
---@field list_all_groups fun(opts?: table): OptimisGroupSummary[]
---@field list_group_pupils fun(group_id: string|table, opts?: table): OptimisPupilRecord[]
---@field list_pupil_memberships fun(pupil_id_or_opts?: string|table, opts?: table): table[]
---@field list_staff fun(): OptimisStaffRecord[]
---@field get_staff_timetable_slots fun(staff_id?: string, cycle_week?: integer, day_of_week_id?: integer, opts?: table): OptimisTimetableSlot[]
---@field get_staff_timetable_slots_batch fun(staff_ids: string[], cycle_week?: integer, day_of_week_id?: integer, opts?: table): table<string, OptimisTimetableSlot[]>
---@field get_pupil_timetable_slots fun(pupil_id_or_opts: string|table, cycle_week?: integer|table, day_of_week_id?: integer, opts?: table): OptimisTimetableSlot[]
---@field list_free_staff_for_period fun(cycle_week_or_opts: integer|table, day_of_week_id?: integer, period_identifier?: string, opts?: table): OptimisStaffRecord[]
---@field list_houses fun(): OptimisHouseSummary[]
---@field list_rooms fun(): OptimisRoomSummary[]
---@field list_courses fun(academic_year_id_or_opts?: string|table): OptimisCourseSummary[]
---@field list_departments fun(): OptimisDepartmentSummary[]
---@field list_year_groups fun(): OptimisYearGroupSummary[]
---@field list_timetable_periods fun(opts?: table): OptimisTimetablePeriodSummary[]
---@field list_timetable_slots fun(opts?: table): OptimisTimetableSlot[]
---@field upsert_pupil_membership fun(params: table, opts?: table): table
---@field upsert_staff_assignment fun(params: table, opts?: table): table
---@field revoke_staff_assignment fun(params: table, opts?: table): { revoked: boolean, rows_affected: integer }
---@field delete_staff_assignment fun(params: table, opts?: table): { revoked: boolean, rows_affected: integer }
---@field upsert_group fun(params: table, opts?: table): OptimisGroupSummary
---@field upsert_house fun(params: table, opts?: table): OptimisHouseSummary
---@field upsert_room fun(params: table, opts?: table): OptimisRoomSummary
---@field upsert_department fun(params: table, opts?: table): OptimisDepartmentSummary
---@field upsert_course fun(params: table, opts?: table): OptimisCourseSummary
---@field upsert_year_group fun(params: table, opts?: table): OptimisYearGroupSummary
---@field upsert_timetable_period fun(params: table, opts?: table): OptimisTimetablePeriodSummary
---@field upsert_timetable_slot fun(params: table, opts?: table): table

---@class OptimisGuardiansDAL
---@field list_for_pupil fun(pupil_id: string, opts?: table): OptimisGuardianRecord[]
---@field list_for_pupils_batch fun(pupil_ids: string[]|table, opts?: table): table<string, OptimisGuardianRecord[]>
---@field list_batch fun(pupil_ids: string[]|table, opts?: table): table<string, OptimisGuardianRecord[]>
---@field list_siblings_for_pupil fun(pupil_id_or_opts: string|table, opts?: table): OptimisSiblingSummary[]
---@field list_households fun(pupil_id_or_opts?: string|table, opts?: table): OptimisHouseholdSummary[]
---@field upsert_guardian_contact fun(params: table, opts?: table): OptimisGuardianRecord

---@class OptimisAttendanceDAL
---@field list_codes fun(): OptimisAttendanceCodeRecord[]
---@field upsert_code fun(params: table, opts?: table): OptimisAttendanceCodeRecord
---@field get_group_session fun(group_id: string, session_date: string, session_type: "AM"|"PM", opts?: table): table[]
---@field get_group_lesson fun(group_id_or_opts: string|table, timetable_slot_id?: string|table, date?: string, opts?: table): table[]
---@field list_session_marks fun(opts?: table): table[]
---@field list_register_submissions fun(opts?: table): table[]
---@field get_pupil_summary fun(pupil_id: string, opts?: table): table
---@field get_pupil_summaries_batch fun(pupil_ids: string[]|table, opts?: table): table<string, table>
---@field get_summaries_batch fun(pupil_ids: string[]|table, opts?: table): table<string, table>
---@field upsert_session_batch fun(params: table, opts?: table): table
---@field record_session_batch fun(params: table, opts?: table): table
---@field upsert_lesson_batch fun(params: table, opts?: table): table
---@field record_lesson_batch fun(params: table, opts?: table): table

---@class OptimisSafeguardingDAL
---@field create_postbox fun(params: table): { note_id: string, status: string }

---@class OptimisSenDAL
---@field list_batch fun(pupil_ids: string[]|table, opts?: table): table<string, OptimisSenAndMedicalSummary>
---@field list_register fun(opts?: table): OptimisSenAndMedicalSummary[]
---@field list_need_types fun(): OptimisSenNeedTypeRecord[]
---@field upsert_need_type fun(params: table): OptimisSenNeedTypeRecord
---@field upsert_sen_record fun(params: table, opts?: table): table
---@field upsert_sen_need fun(params: table, opts?: table): OptimisSenNeedSummary
---@field upsert_sen_assistant fun(params: table, opts?: table): OptimisSenAssistantSummary

---@class OptimisMedicalDAL
---@field list_batch fun(pupil_ids: string[]|table, opts?: table): table<string, OptimisSenAndMedicalSummary>
---@field list_conditions fun(pupil_id_or_opts?: string|table, opts?: table): OptimisMedicalConditionAlert[]
---@field upsert_condition fun(params: table, opts?: table): OptimisMedicalConditionAlert

---@class OptimisStaffDAL
---@field list_active fun(): OptimisStaffRecord[]
---@field list_all fun(opts?: table): OptimisStaffRecord[]
---@field list_roles fun(): OptimisStaffRoleRecord[]
---@field list_cover_assignments fun(opts?: table): OptimisStaffCoverAssignmentRecord[]
---@field get_by_id fun(staff_id: string): OptimisStaffRecord
---@field list_scr_register fun(opts?: table): OptimisStaffSCRRecord[]
---@field list_contracts fun(staff_id_or_opts?: string|table): OptimisStaffContractRecord[]
---@field list_absences fun(staff_id_or_opts?: string|table, academic_year_id?: string): OptimisStaffAbsenceRecord[]
---@field upsert_staff_member fun(params: table): OptimisStaffRecord
---@field upsert_cover_assignment fun(params: table): OptimisStaffCoverAssignmentRecord
---@field upsert_scr_record fun(params: table): OptimisStaffSCRRecord
---@field upsert_contract fun(params: table): OptimisStaffContractRecord
---@field upsert_absence fun(params: table): OptimisStaffAbsenceRecord

---@class OptimisCensusDAL
---@field get_statutory_rollup fun(opts: { census_date: string, term_code: "AUTUMN"|"SPRING"|"SUMMER", academic_year_id?: string, attendance_start_date?: string, attendance_end_date?: string }): table

---@class OptimisDashboardDAL
---@field list_available_widgets fun(): table[]
---@field list_role_defaults fun(): table<string, OptimisDashboardWidgetPlacement[]>
---@field get_role_default fun(role_id: string|table): OptimisDashboardWidgetPlacement[]
---@field save_role_default fun(params: { role_id: string, layout: OptimisDashboardWidgetPlacement[], append_to_existing_staff?: boolean }): { role_id: string, layout: OptimisDashboardWidgetPlacement[], appended_staff_count: integer }
---@field append_widget_to_role fun(params: { role_id: string, plug_id: string, widget_id: string }): { role_id: string, layout: OptimisDashboardWidgetPlacement[], appended_staff_count: integer }
---@field get_staff_layout fun(staff_id?: string|table): { staff_id: string, has_override: boolean, column_count: integer, layout: OptimisDashboardWidgetPlacement[] }
---@field save_staff_layout fun(params: { staff_id?: string, column_count?: integer, layout: OptimisDashboardWidgetPlacement[] }): { staff_id: string, column_count: integer, layout: OptimisDashboardWidgetPlacement[] }
---@field reset_staff_layout fun(staff_id?: string|table): { staff_id: string, reset: boolean }

---@class OptimisDAL
---@field academic OptimisAcademicDAL
---@field pupils OptimisPupilsDAL
---@field groups OptimisGroupsDAL
---@field guardians OptimisGuardiansDAL
---@field attendance OptimisAttendanceDAL
---@field safeguarding OptimisSafeguardingDAL
---@field sen OptimisSenDAL
---@field medical OptimisMedicalDAL
---@field staff OptimisStaffDAL
---@field census OptimisCensusDAL
---@field dashboard OptimisDashboardDAL

---@class OptimisPluginDB
---@field query fun(sql: string, params?: table): table[]
---@field exec fun(sql: string, params?: table): { rows_affected: integer, last_insert_id: integer }
---@field exec_batch fun(sql: string, batch_params: table[]): { rows_affected: integer, batch_size: integer }

---@class OptimisCrypto
---@field uuid fun(): string
---@field seal_secret fun(plaintext: string): string
---@field open_secret fun(ciphertext: string): string

---@class OptimisClock
---@field now_utc fun(): string

---@class OptimisJSON
---@field encode fun(val: any): string
---@field decode fun(str: string): any

---@class OptimisHTTPResponse
---@field ok boolean
---@field status integer
---@field body string
---@field headers table<string, string>
---@field error_code string|nil
---@field error string|nil

---@class OptimisHTTP
---@field get fun(url: string, headers?: table<string, string>): OptimisHTTPResponse
---@field post fun(url: string, headers?: table<string, string>, body?: string): OptimisHTTPResponse

---@class OptimisEvents
---@field emit fun(event_name: string, payload?: table)

---@class OptimisXML
---@field build fun(root_node_spec: table): string
---@field validate_xsd fun(xml_string: string, xsd_schema_name: string): { valid: boolean, errors: string[] }

---@class OptimisPluginContext
---@field caller OptimisCallerContext
---@field school OptimisSchoolContext
---@field dal OptimisDAL
---@field plugin_db OptimisPluginDB
---@field events OptimisEvents
---@field http OptimisHTTP
---@field xml OptimisXML
---@field json OptimisJSON
---@field clock OptimisClock
---@field crypto OptimisCrypto

---@alias OptimisContext OptimisPluginContext

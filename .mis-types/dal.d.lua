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

---@class OptimisPupilRecord
---@field id string
---@field upn string
---@field first_name string
---@field last_name string
---@field preferred_first_name string
---@field preferred_last_name string
---@field legal_first_name string
---@field legal_last_name string
---@field nc_year_group string
---@field enrolment_status "C"|"M"|"S"|"F"|"O"
---@field academic_year_id string
---@field effective_from string
---@field effective_to string|nil

---@class OptimisSenNeedSummary
---@field id string
---@field sen_need_code string
---@field ranking integer
---@field alert_level_id string
---@field effective_from string

---@class OptimisSenAssistantSummary
---@field id string
---@field staff_id string
---@field display_name string
---@field allocation_notes string
---@field effective_from string

---@class OptimisMedicalConditionAlert
---@field id string
---@field condition_name string
---@field condition_category "MEDICAL"|"ALLERGY"|"DIETARY"
---@field alert_level_id string
---@field has_ihcp boolean
---@field triggers string
---@field reaction_symptoms string
---@field emergency_response_plan string
---@field medication_name_and_dose string
---@field medication_location string
---@field parental_consent_given boolean
---@field effective_from string
---@field effective_to string
---@field medication_expiry_date string

---@class OptimisSenAndMedicalSummary
---@field pupil_id string
---@field sen_provision_code "N"|"K"|"E"
---@field primary_sen_need_code string
---@field secondary_sen_need_code string
---@field teaching_strategies string
---@field pupil_targets string
---@field senco_notes string
---@field ehcp_start_date string
---@field ehcp_review_due_date string
---@field next_annual_review_date string
---@field sen_needs OptimisSenNeedSummary[]
---@field sen_assistants OptimisSenAssistantSummary[]
---@field medical_conditions OptimisMedicalConditionAlert[]

---@class OptimisGuardianRecord
---@field relationship_id string
---@field pupil_id string
---@field guardian_person_id string
---@field first_name string
---@field last_name string
---@field email string
---@field phone_primary string
---@field relationship_type string
---@field has_parental_responsibility boolean
---@field court_order_restricted boolean
---@field portal_access_enabled boolean
---@field authorised_to_collect boolean
---@field is_emergency_contact boolean
---@field receives_reports boolean
---@field contact_priority integer
---@field allow_sms boolean
---@field allow_email boolean
---@field allow_phone boolean
---@field household_id string
---@field address_line_1 string
---@field address_line_2 string
---@field town_city string
---@field postcode string
---@field is_confidential_address_redacted boolean

---@class OptimisStaffRecord
---@field id string
---@field staff_id string
---@field person_id string
---@field staff_code string
---@field first_name string
---@field last_name string
---@field preferred_first_name string
---@field preferred_last_name string
---@field legal_first_name string
---@field legal_last_name string
---@field title string
---@field display_name string
---@field email string
---@field staff_type string
---@field job_title string
---@field is_dsl boolean

---@class OptimisGroupSummary
---@field id string
---@field academic_year_id string
---@field group_type string
---@field code string
---@field name string
---@field default_room_id string
---@field takes_attendance boolean
---@field effective_from string

---@class OptimisTimetableSlot
---@field slot_id string
---@field period_id string
---@field period_code string
---@field period_name string
---@field cycle_week integer
---@field day_of_week_id integer
---@field start_time string
---@field end_time string
---@field period_type string
---@field maps_to_statutory_session "AM"|"PM"|""
---@field group_id string
---@field group_code string
---@field staff_id string
---@field room_id string

---@class OptimisPupilsDAL
---@field get_by_id fun(pupil_id: string, opts?: table): OptimisPupilRecord
---@field list_by_group fun(group_id: string, opts?: table): OptimisPupilRecord[]
---@field list_active fun(opts?: table): OptimisPupilRecord[]
---@field list_sen_and_medical_batch fun(pupil_ids: string[], opts?: table): table<string, OptimisSenAndMedicalSummary>
---@field assert_caller_has_access fun(pupil_id: string, opts?: table): boolean

---@class OptimisGroupsDAL
---@field list_for_staff fun(staff_id: string, opts?: table): OptimisGroupSummary[]
---@field list_by_type fun(group_type: string, opts?: table): OptimisGroupSummary[]
---@field get_staff_timetable_slots fun(staff_id: string, opts?: table): OptimisTimetableSlot[]
---@field assert_caller_assigned fun(group_id: string, opts?: table): boolean
---@field upsert_pupil_membership fun(params: table): table
---@field upsert_staff_assignment fun(params: table): table

---@class OptimisGuardiansDAL
---@field list_for_pupil fun(pupil_id: string, opts?: table): OptimisGuardianRecord[]

---@class OptimisAttendanceDAL
---@field list_sessions_for_pupil fun(pupil_id: string, opts?: table): table[]
---@field list_sessions_for_group fun(group_id: string, session_date: string, session_type: "AM"|"PM"): table[]
---@field upsert_session_batch fun(params: table): table

---@class OptimisSafeguardingDAL
---@field create_postbox_concern fun(params: table): table

---@class OptimisSenDAL
---@field list_register fun(opts?: table): OptimisSenAndMedicalSummary[]

---@class OptimisMedicalDAL
---@field list_pupil_conditions fun(pupil_id: string, opts?: table): OptimisMedicalConditionAlert[]

---@class OptimisStaffDAL
---@field get_by_id fun(staff_id: string, opts?: table): OptimisStaffRecord
---@field list_active fun(opts?: table): OptimisStaffRecord[]

---@class OptimisCensusDAL
---@field get_statutory_rollup fun(opts: { census_date: string, term: "AUTUMN"|"SPRING"|"SUMMER", academic_year_id?: string }): table

---@class OptimisDAL
---@field pupils OptimisPupilsDAL
---@field groups OptimisGroupsDAL
---@field guardians OptimisGuardiansDAL
---@field attendance OptimisAttendanceDAL
---@field safeguarding OptimisSafeguardingDAL
---@field sen OptimisSenDAL
---@field medical OptimisMedicalDAL
---@field staff OptimisStaffDAL
---@field census OptimisCensusDAL

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

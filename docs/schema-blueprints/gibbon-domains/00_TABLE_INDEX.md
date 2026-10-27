# Gibbon Schema Blueprint — Agent Triage & Domain Index

> **Purpose:** This index maps all **199 tables** from [`gibbon.sql`](../gibbon.sql) into small, domain-clustered SQL files so agents never need to read the 8,943-line monolithic dump.

## 1. Domain File Directory (Read Only What You Need)

| Domain File | Tables | Lines | Size | Triage Focus |
| :--- | ---: | ---: | ---: | :--- |
| [`01_identity_and_people.sql`](./01_identity_and_people.sql) | 22 | 538 | 24.8 KB | See table list below |
| [`02_family_and_guardians.sql`](./02_family_and_guardians.sql) | 5 | 82 | 3.9 KB | See table list below |
| [`03_school_structure_and_timetable.sql`](./03_school_structure_and_timetable.sql) | 28 | 390 | 17.6 KB | See table list below |
| [`04_attendance.sql`](./04_attendance.sql) | 4 | 73 | 3.6 KB | See table list below |
| [`05_sen_medical_and_pastoral.sql`](./05_sen_medical_and_pastoral.sql) | 17 | 251 | 11.2 KB | See table list below |
| [`06_plugin_assessment_and_reporting.sql`](./06_plugin_assessment_and_reporting.sql) | 34 | 566 | 27.5 KB | See table list below |
| [`07_plugin_behaviour_and_planner.sql`](./07_plugin_behaviour_and_planner.sql) | 28 | 439 | 21.1 KB | See table list below |
| [`08_plugin_finance_ops_and_comms.sql`](./08_plugin_finance_ops_and_comms.sql) | 40 | 787 | 40.0 KB | See table list below |
| [`09_system_lookups_and_legacy.sql`](./09_system_lookups_and_legacy.sql) | 21 | 292 | 13.5 KB | See table list below |

## 2. Optimis Triage Summary (`core.db` vs `plugin_*.db` vs `DROP`)

* **`CORE` (67 tables) + `CORE_LOOKUP` (4 tables):** Reconstructed into Optimis `core.db` with UK DfE CBDS fields and effective-dated temporal columns.
* **`PLUGIN` (110 tables):** Excluded from `core.db`; kept as reference blueprints for isolated `plugin_<name>.db` schemas.
* **`DROP` (18 tables):** Legacy Gibbon PHP framework/session/UI internals not needed in Optimis.

## 3. Complete 199-Table Blueprint Map

| Gibbon Table | Triage | Domain File | Cols | Primary Key | Key Foreign/Link Columns |
| :--- | :--- | :--- | ---: | :--- | :--- |
| `gibbonAction` | **DROP** | [`09_system_lookups_and_legacy.sql`](./09_system_lookups_and_legacy.sql) | 20 | `gibbonActionID` | `gibbonModuleID` |
| `gibbonActivity` | **PLUGIN** | [`07_plugin_behaviour_and_planner.sql`](./07_plugin_behaviour_and_planner.sql) | 19 | `gibbonActivityID` | `gibbonSchoolYearID, gibbonActivityCategoryID, gibbonSchoolYearTermIDList, gibbonYearGroupIDList` |
| `gibbonActivityAttendance` | **PLUGIN** | [`07_plugin_behaviour_and_planner.sql`](./07_plugin_behaviour_and_planner.sql) | 6 | `gibbonActivityAttendanceID` | `gibbonActivityID, gibbonPersonIDTaker` |
| `gibbonActivityCategory` | **PLUGIN** | [`07_plugin_behaviour_and_planner.sql`](./07_plugin_behaviour_and_planner.sql) | 13 | `gibbonActivityCategoryID` | `gibbonSchoolYearID` |
| `gibbonActivityChoice` | **PLUGIN** | [`07_plugin_behaviour_and_planner.sql`](./07_plugin_behaviour_and_planner.sql) | 9 | `gibbonActivityChoiceID` | `gibbonActivityCategoryID, gibbonActivityID, gibbonPersonID, gibbonPersonIDCreated, gibbonPersonIDModified` |
| `gibbonActivityPhoto` | **PLUGIN** | [`07_plugin_behaviour_and_planner.sql`](./07_plugin_behaviour_and_planner.sql) | 5 | `gibbonActivityPhotoID` | `gibbonActivityID` |
| `gibbonActivitySlot` | **PLUGIN** | [`07_plugin_behaviour_and_planner.sql`](./07_plugin_behaviour_and_planner.sql) | 7 | `gibbonActivitySlotID` | `gibbonActivityID, gibbonSpaceID, gibbonDaysOfWeekID` |
| `gibbonActivityStaff` | **PLUGIN** | [`07_plugin_behaviour_and_planner.sql`](./07_plugin_behaviour_and_planner.sql) | 4 | `gibbonActivityStaffID` | `gibbonActivityID, gibbonPersonID` |
| `gibbonActivityStudent` | **PLUGIN** | [`07_plugin_behaviour_and_planner.sql`](./07_plugin_behaviour_and_planner.sql) | 9 | `gibbonActivityStudentID` | `gibbonActivityID, gibbonPersonID, gibbonFinanceInvoiceID, gibbonActivityChoiceID` |
| `gibbonActivityType` | **PLUGIN** | [`07_plugin_behaviour_and_planner.sql`](./07_plugin_behaviour_and_planner.sql) | 8 | `gibbonActivityTypeID` | `-` |
| `gibbonAdmissionsAccount` | **PLUGIN** | [`08_plugin_finance_ops_and_comms.sql`](./08_plugin_finance_ops_and_comms.sql) | 10 | `gibbonAdmissionsAccountID` | `gibbonPersonID, gibbonFamilyID, accessID` |
| `gibbonAdmissionsApplication` | **PLUGIN** | [`08_plugin_finance_ops_and_comms.sql`](./08_plugin_finance_ops_and_comms.sql) | 20 | `gibbonAdmissionsApplicationID` | `gibbonFormID, gibbonFormPageID, gibbonSchoolYearID, gibbonYearGroupID, gibbonFormGroupID...` |
| `gibbonAlarm` | **PLUGIN** | [`08_plugin_finance_ops_and_comms.sql`](./08_plugin_finance_ops_and_comms.sql) | 6 | `gibbonAlarmID` | `gibbonPersonID` |
| `gibbonAlarmConfirm` | **PLUGIN** | [`08_plugin_finance_ops_and_comms.sql`](./08_plugin_finance_ops_and_comms.sql) | 4 | `gibbonAlarmConfirmID` | `gibbonAlarmID, gibbonPersonID` |
| `gibbonAlertLevel` | **CORE** | [`05_sen_medical_and_pastoral.sql`](./05_sen_medical_and_pastoral.sql) | 7 | `gibbonAlertLevelID` | `-` |
| `gibbonApplicationForm` | **PLUGIN** | [`08_plugin_finance_ops_and_comms.sql`](./08_plugin_finance_ops_and_comms.sql) | 127 | `gibbonApplicationFormID` | `gibbonFamilyID, gibbonFinanceFeeCategoryIDList, parent1gibbonPersonID, gibbonFormGroupID, gibbonPaymentID...` |
| `gibbonApplicationFormFile` | **PLUGIN** | [`08_plugin_finance_ops_and_comms.sql`](./08_plugin_finance_ops_and_comms.sql) | 4 | `gibbonApplicationFormFileID` | `gibbonApplicationFormID` |
| `gibbonApplicationFormLink` | **PLUGIN** | [`08_plugin_finance_ops_and_comms.sql`](./08_plugin_finance_ops_and_comms.sql) | 4 | `gibbonApplicationFormLinkID` | `-` |
| `gibbonApplicationFormRelationship` | **PLUGIN** | [`08_plugin_finance_ops_and_comms.sql`](./08_plugin_finance_ops_and_comms.sql) | 4 | `gibbonApplicationFormRelationshipID` | `gibbonApplicationFormID, gibbonPersonID` |
| `gibbonAttendanceCode` | **CORE** | [`04_attendance.sql`](./04_attendance.sql) | 12 | `gibbonAttendanceCodeID` | `-` |
| `gibbonAttendanceLogCourseClass` | **CORE** | [`04_attendance.sql`](./04_attendance.sql) | 6 | `gibbonAttendanceLogCourseClassID` | `gibbonCourseClassID, gibbonTTDayRowClassID, gibbonPersonIDTaker` |
| `gibbonAttendanceLogFormGroup` | **CORE** | [`04_attendance.sql`](./04_attendance.sql) | 5 | `gibbonAttendanceLogFormGroupID` | `gibbonFormGroupID, gibbonPersonIDTaker` |
| `gibbonAttendanceLogPerson` | **CORE** | [`04_attendance.sql`](./04_attendance.sql) | 14 | `gibbonAttendanceLogPersonID` | `gibbonAttendanceCodeID, gibbonPersonID, gibbonPersonIDTaker, gibbonFormGroupID, gibbonCourseClassID...` |
| `gibbonBehaviour` | **PLUGIN** | [`07_plugin_behaviour_and_planner.sql`](./07_plugin_behaviour_and_planner.sql) | 14 | `gibbonBehaviourID` | `gibbonSchoolYearID, gibbonPersonID, gibbonPlannerEntryID, gibbonPersonIDCreator, gibbonMultiIncidentID` |
| `gibbonBehaviourFollowUp` | **PLUGIN** | [`07_plugin_behaviour_and_planner.sql`](./07_plugin_behaviour_and_planner.sql) | 5 | `gibbonBehaviourFollowUpID` | `gibbonBehaviourID, gibbonPersonID` |
| `gibbonBehaviourLetter` | **PLUGIN** | [`07_plugin_behaviour_and_planner.sql`](./07_plugin_behaviour_and_planner.sql) | 10 | `gibbonBehaviourLetterID` | `gibbonSchoolYearID, gibbonPersonID` |
| `gibbonCountry` | **CORE_LOOKUP** | [`09_system_lookups_and_legacy.sql`](./09_system_lookups_and_legacy.sql) | 2 | `printable_name` | `-` |
| `gibbonCourse` | **CORE** | [`03_school_structure_and_timetable.sql`](./03_school_structure_and_timetable.sql) | 10 | `gibbonCourseID` | `gibbonSchoolYearID, gibbonDepartmentID, gibbonYearGroupIDList` |
| `gibbonCourseClass` | **CORE** | [`03_school_structure_and_timetable.sql`](./03_school_structure_and_timetable.sql) | 10 | `gibbonCourseClassID` | `gibbonCourseID` |
| `gibbonCourseClassMap` | **CORE** | [`03_school_structure_and_timetable.sql`](./03_school_structure_and_timetable.sql) | 4 | `gibbonCourseClassMapID` | `gibbonCourseClassID, gibbonFormGroupID, gibbonYearGroupID` |
| `gibbonCourseClassPerson` | **CORE** | [`03_school_structure_and_timetable.sql`](./03_school_structure_and_timetable.sql) | 7 | `gibbonCourseClassPersonID` | `gibbonCourseClassID, gibbonPersonID` |
| `gibbonCrowdAssessDiscuss` | **PLUGIN** | [`06_plugin_assessment_and_reporting.sql`](./06_plugin_assessment_and_reporting.sql) | 6 | `gibbonCrowdAssessDiscussID` | `gibbonPlannerEntryHomeworkID, gibbonPersonID` |
| `gibbonCustomField` | **DROP** | [`09_system_lookups_and_legacy.sql`](./09_system_lookups_and_legacy.sql) | 18 | `gibbonCustomFieldID` | `-` |
| `gibbonDataRetention` | **CORE_LOOKUP** | [`09_system_lookups_and_legacy.sql`](./09_system_lookups_and_legacy.sql) | 6 | `gibbonDataRetentionID` | `gibbonPersonID, gibbonPersonIDOperator` |
| `gibbonDaysOfWeek` | **CORE** | [`03_school_structure_and_timetable.sql`](./03_school_structure_and_timetable.sql) | 9 | `gibbonDaysOfWeekID` | `-` |
| `gibbonDepartment` | **CORE** | [`03_school_structure_and_timetable.sql`](./03_school_structure_and_timetable.sql) | 9 | `gibbonDepartmentID` | `-` |
| `gibbonDepartmentResource` | **PLUGIN** | [`03_school_structure_and_timetable.sql`](./03_school_structure_and_timetable.sql) | 5 | `gibbonDepartmentResourceID` | `gibbonDepartmentID` |
| `gibbonDepartmentStaff` | **CORE** | [`03_school_structure_and_timetable.sql`](./03_school_structure_and_timetable.sql) | 4 | `gibbonDepartmentStaffID` | `gibbonDepartmentID, gibbonPersonID` |
| `gibbonDiscussion` | **PLUGIN** | [`07_plugin_behaviour_and_planner.sql`](./07_plugin_behaviour_and_planner.sql) | 13 | `gibbonDiscussionID` | `foreignTableID, gibbonModuleID, gibbonPersonID, gibbonPersonIDTarget` |
| `gibbonDistrict` | **CORE_LOOKUP** | [`09_system_lookups_and_legacy.sql`](./09_system_lookups_and_legacy.sql) | 2 | `gibbonDistrictID` | `-` |
| `gibbonEmailTemplate` | **PLUGIN** | [`08_plugin_finance_ops_and_comms.sql`](./08_plugin_finance_ops_and_comms.sql) | 9 | `gibbonEmailTemplateID` | `-` |
| `gibbonExternalAssessment` | **PLUGIN** | [`06_plugin_assessment_and_reporting.sql`](./06_plugin_assessment_and_reporting.sql) | 7 | `gibbonExternalAssessmentID` | `-` |
| `gibbonExternalAssessmentField` | **PLUGIN** | [`06_plugin_assessment_and_reporting.sql`](./06_plugin_assessment_and_reporting.sql) | 7 | `gibbonExternalAssessmentFieldID` | `gibbonExternalAssessmentID, gibbonScaleID, gibbonYearGroupIDList` |
| `gibbonExternalAssessmentStudent` | **PLUGIN** | [`06_plugin_assessment_and_reporting.sql`](./06_plugin_assessment_and_reporting.sql) | 5 | `gibbonExternalAssessmentStudentID` | `gibbonExternalAssessmentID, gibbonPersonID` |
| `gibbonExternalAssessmentStudentEntry` | **PLUGIN** | [`06_plugin_assessment_and_reporting.sql`](./06_plugin_assessment_and_reporting.sql) | 4 | `gibbonExternalAssessmentStudentEntryID` | `gibbonExternalAssessmentStudentID, gibbonExternalAssessmentFieldID, gibbonScaleGradeID` |
| `gibbonFamily` | **CORE** | [`02_family_and_guardians.sql`](./02_family_and_guardians.sql) | 10 | `gibbonFamilyID` | `-` |
| `gibbonFamilyAdult` | **CORE** | [`02_family_and_guardians.sql`](./02_family_and_guardians.sql) | 10 | `gibbonFamilyAdultID` | `gibbonFamilyID, gibbonPersonID` |
| `gibbonFamilyChild` | **CORE** | [`02_family_and_guardians.sql`](./02_family_and_guardians.sql) | 4 | `gibbonFamilyChildID` | `gibbonFamilyID, gibbonPersonID` |
| `gibbonFamilyRelationship` | **CORE** | [`02_family_and_guardians.sql`](./02_family_and_guardians.sql) | 5 | `gibbonFamilyRelationshipID` | `gibbonFamilyID, gibbonPersonID1, gibbonPersonID2` |
| `gibbonFamilyUpdate` | **CORE** | [`02_family_and_guardians.sql`](./02_family_and_guardians.sql) | 12 | `gibbonFamilyUpdateID` | `gibbonSchoolYearID, gibbonFamilyID, gibbonPersonIDUpdater` |
| `gibbonFileExtension` | **DROP** | [`09_system_lookups_and_legacy.sql`](./09_system_lookups_and_legacy.sql) | 4 | `gibbonFileExtensionID` | `-` |
| `gibbonFinanceBillingSchedule` | **PLUGIN** | [`08_plugin_finance_ops_and_comms.sql`](./08_plugin_finance_ops_and_comms.sql) | 11 | `gibbonFinanceBillingScheduleID` | `gibbonSchoolYearID, gibbonPersonIDCreator, gibbonPersonIDUpdate` |
| `gibbonFinanceBudget` | **PLUGIN** | [`08_plugin_finance_ops_and_comms.sql`](./08_plugin_finance_ops_and_comms.sql) | 9 | `gibbonFinanceBudgetID` | `gibbonPersonIDCreator, gibbonPersonIDUpdate` |
| `gibbonFinanceBudgetCycle` | **PLUGIN** | [`08_plugin_finance_ops_and_comms.sql`](./08_plugin_finance_ops_and_comms.sql) | 10 | `gibbonFinanceBudgetCycleID` | `gibbonPersonIDCreator, gibbonPersonIDUpdate` |
| `gibbonFinanceBudgetCycleAllocation` | **PLUGIN** | [`08_plugin_finance_ops_and_comms.sql`](./08_plugin_finance_ops_and_comms.sql) | 4 | `gibbonFinanceBudgetCycleAllocationID` | `gibbonFinanceBudgetID, gibbonFinanceBudgetCycleID` |
| `gibbonFinanceBudgetPerson` | **PLUGIN** | [`08_plugin_finance_ops_and_comms.sql`](./08_plugin_finance_ops_and_comms.sql) | 4 | `gibbonFinanceBudgetPersonID` | `gibbonFinanceBudgetID, gibbonPersonID` |
| `gibbonFinanceExpense` | **PLUGIN** | [`08_plugin_finance_ops_and_comms.sql`](./08_plugin_finance_ops_and_comms.sql) | 20 | `gibbonFinanceExpenseID` | `gibbonFinanceBudgetID, gibbonFinanceBudgetCycleID, gibbonPersonIDPayment, paymentID, gibbonPersonIDCreator` |
| `gibbonFinanceExpenseApprover` | **PLUGIN** | [`08_plugin_finance_ops_and_comms.sql`](./08_plugin_finance_ops_and_comms.sql) | 7 | `gibbonFinanceExpenseApproverID` | `gibbonPersonID, gibbonPersonIDCreator, gibbonPersonIDUpdate` |
| `gibbonFinanceExpenseLog` | **PLUGIN** | [`08_plugin_finance_ops_and_comms.sql`](./08_plugin_finance_ops_and_comms.sql) | 6 | `gibbonFinanceExpenseLogID` | `gibbonFinanceExpenseID, gibbonPersonID` |
| `gibbonFinanceFee` | **PLUGIN** | [`08_plugin_finance_ops_and_comms.sql`](./08_plugin_finance_ops_and_comms.sql) | 12 | `gibbonFinanceFeeID` | `gibbonSchoolYearID, gibbonFinanceFeeCategoryID, gibbonPersonIDCreator, gibbonPersonIDUpdate` |
| `gibbonFinanceFeeCategory` | **PLUGIN** | [`08_plugin_finance_ops_and_comms.sql`](./08_plugin_finance_ops_and_comms.sql) | 9 | `gibbonFinanceFeeCategoryID` | `gibbonPersonIDCreator, gibbonPersonIDUpdate` |
| `gibbonFinanceInvoice` | **PLUGIN** | [`08_plugin_finance_ops_and_comms.sql`](./08_plugin_finance_ops_and_comms.sql) | 21 | `gibbonFinanceInvoiceID` | `gibbonSchoolYearID, gibbonFinanceInvoiceeID, gibbonFinanceBillingScheduleID, gibbonFinanceFeeCategoryIDList, gibbonPaymentID...` |
| `gibbonFinanceInvoiceFee` | **PLUGIN** | [`08_plugin_finance_ops_and_comms.sql`](./08_plugin_finance_ops_and_comms.sql) | 10 | `gibbonFinanceInvoiceFeeID` | `gibbonFinanceInvoiceID, gibbonFinanceFeeID, gibbonFinanceFeeCategoryID` |
| `gibbonFinanceInvoicee` | **PLUGIN** | [`08_plugin_finance_ops_and_comms.sql`](./08_plugin_finance_ops_and_comms.sql) | 11 | `gibbonFinanceInvoiceeID` | `gibbonPersonID, gibbonFinanceFeeCategoryIDList` |
| `gibbonFinanceInvoiceeUpdate` | **PLUGIN** | [`08_plugin_finance_ops_and_comms.sql`](./08_plugin_finance_ops_and_comms.sql) | 15 | `gibbonFinanceInvoiceeUpdateID` | `gibbonSchoolYearID, gibbonFinanceInvoiceeID, gibbonFinanceFeeCategoryIDList, gibbonPersonIDUpdater` |
| `gibbonFirstAid` | **PLUGIN** | [`05_sen_medical_and_pastoral.sql`](./05_sen_medical_and_pastoral.sql) | 14 | `gibbonFirstAidID` | `gibbonPersonIDPatient, gibbonCourseClassID, gibbonPersonIDFirstAider, gibbonPersonIDFollowUp, gibbonSchoolYearID` |
| `gibbonFirstAidFollowUp` | **PLUGIN** | [`05_sen_medical_and_pastoral.sql`](./05_sen_medical_and_pastoral.sql) | 5 | `gibbonFirstAidFollowUpID` | `gibbonFirstAidID, gibbonPersonID` |
| `gibbonForm` | **PLUGIN** | [`08_plugin_finance_ops_and_comms.sql`](./08_plugin_finance_ops_and_comms.sql) | 8 | `gibbonFormID` | `gibbonYearGroupIDList` |
| `gibbonFormField` | **PLUGIN** | [`08_plugin_finance_ops_and_comms.sql`](./08_plugin_finance_ops_and_comms.sql) | 14 | `gibbonFormFieldID` | `gibbonFormPageID` |
| `gibbonFormGroup` | **CORE** | [`03_school_structure_and_timetable.sql`](./03_school_structure_and_timetable.sql) | 14 | `gibbonFormGroupID` | `gibbonSchoolYearID, gibbonPersonIDTutor, gibbonPersonIDTutor2, gibbonPersonIDTutor3, gibbonPersonIDEA...` |
| `gibbonFormPage` | **PLUGIN** | [`08_plugin_finance_ops_and_comms.sql`](./08_plugin_finance_ops_and_comms.sql) | 6 | `gibbonFormPageID` | `gibbonFormID` |
| `gibbonFormSubmission` | **PLUGIN** | [`08_plugin_finance_ops_and_comms.sql`](./08_plugin_finance_ops_and_comms.sql) | 13 | `gibbonFormSubmissionID` | `gibbonFormID, gibbonFormPageID, foreignTableID, gibbonPersonIDModified` |
| `gibbonFormUpload` | **PLUGIN** | [`08_plugin_finance_ops_and_comms.sql`](./08_plugin_finance_ops_and_comms.sql) | 9 | `gibbonFormUploadID` | `gibbonFormID, gibbonFormFieldID, foreignTableID` |
| `gibbonGroup` | **CORE** | [`03_school_structure_and_timetable.sql`](./03_school_structure_and_timetable.sql) | 6 | `gibbonGroupID` | `gibbonPersonIDOwner, gibbonSchoolYearID` |
| `gibbonGroupPerson` | **CORE** | [`03_school_structure_and_timetable.sql`](./03_school_structure_and_timetable.sql) | 3 | `gibbonGroupPersonID` | `gibbonGroupID, gibbonPersonID` |
| `gibbonHook` | **DROP** | [`09_system_lookups_and_legacy.sql`](./09_system_lookups_and_legacy.sql) | 5 | `gibbonHookID` | `gibbonModuleID` |
| `gibbonHouse` | **CORE** | [`03_school_structure_and_timetable.sql`](./03_school_structure_and_timetable.sql) | 4 | `gibbonHouseID` | `-` |
| `gibbonIN` | **CORE** | [`05_sen_medical_and_pastoral.sql`](./05_sen_medical_and_pastoral.sql) | 6 | `gibbonINID` | `gibbonPersonID` |
| `gibbonINArchive` | **CORE** | [`05_sen_medical_and_pastoral.sql`](./05_sen_medical_and_pastoral.sql) | 9 | `gibbonINArchiveID` | `gibbonPersonID` |
| `gibbonINAssistant` | **CORE** | [`05_sen_medical_and_pastoral.sql`](./05_sen_medical_and_pastoral.sql) | 4 | `gibbonINAssistantID` | `gibbonPersonIDStudent, gibbonPersonIDAssistant` |
| `gibbonINDescriptor` | **CORE** | [`05_sen_medical_and_pastoral.sql`](./05_sen_medical_and_pastoral.sql) | 5 | `gibbonINDescriptorID` | `-` |
| `gibbonINInvestigation` | **CORE** | [`05_sen_medical_and_pastoral.sql`](./05_sen_medical_and_pastoral.sql) | 11 | `gibbonINInvestigationID` | `gibbonSchoolYearID, gibbonPersonIDCreator, gibbonPersonIDStudent` |
| `gibbonINInvestigationContribution` | **CORE** | [`05_sen_medical_and_pastoral.sql`](./05_sen_medical_and_pastoral.sql) | 13 | `gibbonINInvestigationContributionID` | `gibbonINInvestigationID, gibbonPersonID, gibbonCourseClassPersonID` |
| `gibbonINPersonDescriptor` | **CORE** | [`05_sen_medical_and_pastoral.sql`](./05_sen_medical_and_pastoral.sql) | 4 | `gibbonINPersonDescriptorID` | `gibbonPersonID, gibbonINDescriptorID, gibbonAlertLevelID` |
| `gibbonInternalAssessmentColumn` | **PLUGIN** | [`06_plugin_assessment_and_reporting.sql`](./06_plugin_assessment_and_reporting.sql) | 19 | `gibbonInternalAssessmentColumnID` | `gibbonCourseClassID, groupingID, gibbonPersonIDCreator, gibbonPersonIDLastEdit` |
| `gibbonInternalAssessmentEntry` | **PLUGIN** | [`06_plugin_assessment_and_reporting.sql`](./06_plugin_assessment_and_reporting.sql) | 10 | `gibbonInternalAssessmentEntryID` | `gibbonInternalAssessmentColumnID, gibbonPersonIDStudent, gibbonPersonIDLastEdit` |
| `gibbonLanguage` | **CORE_LOOKUP** | [`09_system_lookups_and_legacy.sql`](./09_system_lookups_and_legacy.sql) | 2 | `gibbonLanguageID` | `-` |
| `gibbonLibraryItem` | **PLUGIN** | [`08_plugin_finance_ops_and_comms.sql`](./08_plugin_finance_ops_and_comms.sql) | 36 | `gibbonLibraryItemID` | `gibbonLibraryTypeID, gibbonSpaceID, gibbonPersonIDOwnership, gibbonDepartmentID, gibbonPersonIDStatusResponsible...` |
| `gibbonLibraryItemEvent` | **PLUGIN** | [`08_plugin_finance_ops_and_comms.sql`](./08_plugin_finance_ops_and_comms.sql) | 12 | `gibbonLibraryItemEventID` | `gibbonLibraryItemID, gibbonPersonIDStatusResponsible, gibbonPersonIDOut, gibbonPersonIDReturnAction, gibbonPersonIDIn` |
| `gibbonLibraryShelf` | **PLUGIN** | [`08_plugin_finance_ops_and_comms.sql`](./08_plugin_finance_ops_and_comms.sql) | 8 | `gibbonLibraryShelfID` | `-` |
| `gibbonLibraryShelfItem` | **PLUGIN** | [`08_plugin_finance_ops_and_comms.sql`](./08_plugin_finance_ops_and_comms.sql) | 3 | `gibbonLibraryShelfItemID` | `gibbonLibraryItemID, gibbonLibraryShelfID` |
| `gibbonLibraryType` | **PLUGIN** | [`08_plugin_finance_ops_and_comms.sql`](./08_plugin_finance_ops_and_comms.sql) | 4 | `gibbonLibraryTypeID` | `-` |
| `gibbonLog` | **DROP** | [`09_system_lookups_and_legacy.sql`](./09_system_lookups_and_legacy.sql) | 8 | `gibbonLogID` | `gibbonModuleID, gibbonPersonID, gibbonSchoolYearID` |
| `gibbonMarkbookColumn` | **PLUGIN** | [`06_plugin_assessment_and_reporting.sql`](./06_plugin_assessment_and_reporting.sql) | 30 | `gibbonMarkbookColumnID` | `gibbonCourseClassID, gibbonHookID, gibbonUnitID, gibbonPlannerEntryID, gibbonSchoolYearTermID...` |
| `gibbonMarkbookEntry` | **PLUGIN** | [`06_plugin_assessment_and_reporting.sql`](./06_plugin_assessment_and_reporting.sql) | 14 | `gibbonMarkbookEntryID` | `gibbonMarkbookColumnID, gibbonPersonIDStudent, gibbonPersonIDLastEdit` |
| `gibbonMarkbookTarget` | **PLUGIN** | [`06_plugin_assessment_and_reporting.sql`](./06_plugin_assessment_and_reporting.sql) | 4 | `gibbonMarkbookTargetID` | `gibbonCourseClassID, gibbonPersonIDStudent, gibbonScaleGradeID` |
| `gibbonMarkbookWeight` | **PLUGIN** | [`06_plugin_assessment_and_reporting.sql`](./06_plugin_assessment_and_reporting.sql) | 7 | `gibbonMarkbookWeightID` | `gibbonCourseClassID` |
| `gibbonMedicalCondition` | **CORE** | [`05_sen_medical_and_pastoral.sql`](./05_sen_medical_and_pastoral.sql) | 3 | `gibbonMedicalConditionID` | `-` |
| `gibbonMessenger` | **PLUGIN** | [`08_plugin_finance_ops_and_comms.sql`](./08_plugin_finance_ops_and_comms.sql) | 23 | `gibbonMessengerID` | `gibbonSchoolYearID, gibbonPersonID` |
| `gibbonMessengerCannedResponse` | **PLUGIN** | [`08_plugin_finance_ops_and_comms.sql`](./08_plugin_finance_ops_and_comms.sql) | 5 | `gibbonMessengerCannedResponseID` | `gibbonPersonIDCreator` |
| `gibbonMessengerMailingList` | **PLUGIN** | [`08_plugin_finance_ops_and_comms.sql`](./08_plugin_finance_ops_and_comms.sql) | 4 | `gibbonMessengerMailingListID` | `-` |
| `gibbonMessengerMailingListRecipient` | **PLUGIN** | [`08_plugin_finance_ops_and_comms.sql`](./08_plugin_finance_ops_and_comms.sql) | 8 | `gibbonMessengerMailingListRecipientID` | `gibbonMessengerMailingListIDList` |
| `gibbonMessengerReceipt` | **PLUGIN** | [`08_plugin_finance_ops_and_comms.sql`](./08_plugin_finance_ops_and_comms.sql) | 14 | `gibbonMessengerReceiptID` | `gibbonMessengerID, gibbonPersonID, targetID, gibbonPersonIDListStudent` |
| `gibbonMessengerTarget` | **PLUGIN** | [`08_plugin_finance_ops_and_comms.sql`](./08_plugin_finance_ops_and_comms.sql) | 7 | `gibbonMessengerTargetID` | `gibbonMessengerID` |
| `gibbonMigration` | **DROP** | [`09_system_lookups_and_legacy.sql`](./09_system_lookups_and_legacy.sql) | 4 | `gibbonMigrationID` | `-` |
| `gibbonModule` | **DROP** | [`09_system_lookups_and_legacy.sql`](./09_system_lookups_and_legacy.sql) | 10 | `gibbonModuleID` | `-` |
| `gibbonNotification` | **DROP** | [`09_system_lookups_and_legacy.sql`](./09_system_lookups_and_legacy.sql) | 8 | `gibbonNotificationID` | `gibbonPersonID, gibbonModuleID` |
| `gibbonNotificationEvent` | **DROP** | [`09_system_lookups_and_legacy.sql`](./09_system_lookups_and_legacy.sql) | 7 | `gibbonNotificationEventID` | `-` |
| `gibbonNotificationListener` | **DROP** | [`09_system_lookups_and_legacy.sql`](./09_system_lookups_and_legacy.sql) | 5 | `gibbonNotificationListenerID` | `gibbonNotificationEventID, gibbonPersonID, scopeID` |
| `gibbonOutcome` | **PLUGIN** | [`06_plugin_assessment_and_reporting.sql`](./06_plugin_assessment_and_reporting.sql) | 10 | `gibbonOutcomeID` | `gibbonDepartmentID, gibbonYearGroupIDList, gibbonPersonIDCreator` |
| `gibbonPayment` | **PLUGIN** | [`08_plugin_finance_ops_and_comms.sql`](./08_plugin_finance_ops_and_comms.sql) | 14 | `gibbonPaymentID` | `foreignTableID, gibbonPersonID, paymentPayerID, paymentTransactionID, paymentReceiptID` |
| `gibbonPermission` | **DROP** | [`09_system_lookups_and_legacy.sql`](./09_system_lookups_and_legacy.sql) | 3 | `permissionID` | `gibbonRoleID, gibbonActionID` |
| `gibbonPerson` | **CORE** | [`01_identity_and_people.sql`](./01_identity_and_people.sql) | 94 | `gibbonPersonID` | `gibbonHouseID, studentID, gibbonApplicationFormID` |
| `gibbonPersonMedical` | **CORE** | [`05_sen_medical_and_pastoral.sql`](./05_sen_medical_and_pastoral.sql) | 6 | `gibbonPersonMedicalID` | `gibbonPersonID` |
| `gibbonPersonMedicalCondition` | **CORE** | [`05_sen_medical_and_pastoral.sql`](./05_sen_medical_and_pastoral.sql) | 12 | `gibbonPersonMedicalConditionID` | `gibbonPersonMedicalID, gibbonAlertLevelID` |
| `gibbonPersonMedicalConditionUpdate` | **CORE** | [`05_sen_medical_and_pastoral.sql`](./05_sen_medical_and_pastoral.sql) | 16 | `gibbonPersonMedicalConditionUpdateID` | `gibbonPersonMedicalUpdateID, gibbonPersonMedicalConditionID, gibbonPersonMedicalID, gibbonAlertLevelID, gibbonPersonIDUpdater` |
| `gibbonPersonMedicalUpdate` | **CORE** | [`05_sen_medical_and_pastoral.sql`](./05_sen_medical_and_pastoral.sql) | 11 | `gibbonPersonMedicalUpdateID` | `gibbonSchoolYearID, gibbonPersonMedicalID, gibbonPersonID, gibbonPersonIDUpdater` |
| `gibbonPersonReset` | **DROP** | [`01_identity_and_people.sql`](./01_identity_and_people.sql) | 4 | `gibbonPersonResetID` | `gibbonPersonID` |
| `gibbonPersonStatusLog` | **CORE** | [`01_identity_and_people.sql`](./01_identity_and_people.sql) | 7 | `gibbonPersonStatusLogID` | `gibbonPersonID, gibbonPersonIDModified` |
| `gibbonPersonUpdate` | **CORE** | [`01_identity_and_people.sql`](./01_identity_and_people.sql) | 53 | `gibbonPersonUpdateID` | `gibbonSchoolYearID, gibbonPersonID, gibbonPersonIDUpdater` |
| `gibbonPersonalDocument` | **CORE** | [`01_identity_and_people.sql`](./01_identity_and_people.sql) | 14 | `gibbonPersonalDocumentID` | `gibbonPersonalDocumentTypeID, foreignTableID, gibbonPersonIDUpdater` |
| `gibbonPersonalDocumentType` | **CORE** | [`01_identity_and_people.sql`](./01_identity_and_people.sql) | 15 | `gibbonPersonalDocumentTypeID` | `-` |
| `gibbonPlannerEntry` | **PLUGIN** | [`07_plugin_behaviour_and_planner.sql`](./07_plugin_behaviour_and_planner.sql) | 32 | `gibbonPlannerEntryID` | `gibbonCourseClassID, gibbonUnitID, gibbonPersonIDCreator, gibbonPersonIDLastEdit` |
| `gibbonPlannerEntryDiscuss` | **PLUGIN** | [`07_plugin_behaviour_and_planner.sql`](./07_plugin_behaviour_and_planner.sql) | 6 | `gibbonPlannerEntryDiscussID` | `gibbonPlannerEntryID, gibbonPersonID` |
| `gibbonPlannerEntryGuest` | **PLUGIN** | [`07_plugin_behaviour_and_planner.sql`](./07_plugin_behaviour_and_planner.sql) | 4 | `gibbonPlannerEntryGuestID` | `gibbonPlannerEntryID, gibbonPersonID` |
| `gibbonPlannerEntryHomework` | **PLUGIN** | [`07_plugin_behaviour_and_planner.sql`](./07_plugin_behaviour_and_planner.sql) | 9 | `gibbonPlannerEntryHomeworkID` | `gibbonPlannerEntryID, gibbonPersonID` |
| `gibbonPlannerEntryOutcome` | **PLUGIN** | [`07_plugin_behaviour_and_planner.sql`](./07_plugin_behaviour_and_planner.sql) | 5 | `gibbonPlannerEntryOutcomeID` | `gibbonPlannerEntryID, gibbonOutcomeID` |
| `gibbonPlannerEntryStudentHomework` | **PLUGIN** | [`07_plugin_behaviour_and_planner.sql`](./07_plugin_behaviour_and_planner.sql) | 6 | `gibbonPlannerEntryStudentHomeworkID` | `gibbonPlannerEntryID, gibbonPersonID` |
| `gibbonPlannerEntryStudentTracker` | **PLUGIN** | [`07_plugin_behaviour_and_planner.sql`](./07_plugin_behaviour_and_planner.sql) | 4 | `gibbonPlannerEntryStudentTrackerID` | `gibbonPlannerEntryID, gibbonPersonID` |
| `gibbonPlannerParentWeeklyEmailSummary` | **PLUGIN** | [`07_plugin_behaviour_and_planner.sql`](./07_plugin_behaviour_and_planner.sql) | 7 | `gibbonPlannerParentWeeklyEmailSummaryID` | `gibbonSchoolYearID, gibbonPersonIDParent, gibbonPersonIDStudent` |
| `gibbonReport` | **PLUGIN** | [`06_plugin_assessment_and_reporting.sql`](./06_plugin_assessment_and_reporting.sql) | 13 | `gibbonReportID` | `gibbonReportArchiveID, gibbonReportTemplateID, gibbonSchoolYearID, gibbonReportingCycleID, gibbonYearGroupIDList...` |
| `gibbonReportArchive` | **PLUGIN** | [`06_plugin_assessment_and_reporting.sql`](./06_plugin_assessment_and_reporting.sql) | 8 | `gibbonReportArchiveID` | `-` |
| `gibbonReportArchiveEntry` | **PLUGIN** | [`06_plugin_assessment_and_reporting.sql`](./06_plugin_assessment_and_reporting.sql) | 18 | `gibbonReportArchiveEntryID` | `gibbonReportArchiveID, gibbonReportID, gibbonSchoolYearID, gibbonYearGroupID, gibbonFormGroupID...` |
| `gibbonReportPrototypeSection` | **PLUGIN** | [`06_plugin_assessment_and_reporting.sql`](./06_plugin_assessment_and_reporting.sql) | 11 | `gibbonReportPrototypeSectionID` | `-` |
| `gibbonReportTemplate` | **PLUGIN** | [`06_plugin_assessment_and_reporting.sql`](./06_plugin_assessment_and_reporting.sql) | 12 | `gibbonReportTemplateID` | `-` |
| `gibbonReportTemplateFont` | **PLUGIN** | [`06_plugin_assessment_and_reporting.sql`](./06_plugin_assessment_and_reporting.sql) | 6 | `gibbonReportTemplateFontID` | `-` |
| `gibbonReportTemplateSection` | **PLUGIN** | [`06_plugin_assessment_and_reporting.sql`](./06_plugin_assessment_and_reporting.sql) | 11 | `gibbonReportTemplateSectionID` | `gibbonReportTemplateID, gibbonReportPrototypeSectionID` |
| `gibbonReportingAccess` | **PLUGIN** | [`06_plugin_assessment_and_reporting.sql`](./06_plugin_assessment_and_reporting.sql) | 10 | `gibbonReportingAccessID` | `gibbonReportingCycleID, gibbonReportingScopeIDList, gibbonRoleIDList, gibbonPersonIDList` |
| `gibbonReportingCriteria` | **PLUGIN** | [`06_plugin_assessment_and_reporting.sql`](./06_plugin_assessment_and_reporting.sql) | 14 | `gibbonReportingCriteriaID` | `gibbonReportingCycleID, gibbonReportingScopeID, gibbonReportingCriteriaTypeID, gibbonYearGroupID, gibbonFormGroupID...` |
| `gibbonReportingCriteriaType` | **PLUGIN** | [`06_plugin_assessment_and_reporting.sql`](./06_plugin_assessment_and_reporting.sql) | 8 | `gibbonReportingCriteriaTypeID` | `gibbonScaleID` |
| `gibbonReportingCycle` | **PLUGIN** | [`06_plugin_assessment_and_reporting.sql`](./06_plugin_assessment_and_reporting.sql) | 12 | `gibbonReportingCycleID` | `gibbonSchoolYearID, gibbonYearGroupIDList` |
| `gibbonReportingProgress` | **PLUGIN** | [`06_plugin_assessment_and_reporting.sql`](./06_plugin_assessment_and_reporting.sql) | 7 | `gibbonReportingProgressID` | `gibbonReportingScopeID, gibbonYearGroupID, gibbonFormGroupID, gibbonCourseClassID, gibbonPersonIDStudent` |
| `gibbonReportingProof` | **PLUGIN** | [`06_plugin_assessment_and_reporting.sql`](./06_plugin_assessment_and_reporting.sql) | 9 | `gibbonReportingProofID` | `gibbonReportingValueID, gibbonPersonIDProofed, gibbonPersonIDActioned` |
| `gibbonReportingScope` | **PLUGIN** | [`06_plugin_assessment_and_reporting.sql`](./06_plugin_assessment_and_reporting.sql) | 5 | `gibbonReportingScopeID` | `gibbonReportingCycleID` |
| `gibbonReportingValue` | **PLUGIN** | [`06_plugin_assessment_and_reporting.sql`](./06_plugin_assessment_and_reporting.sql) | 13 | `gibbonReportingValueID` | `gibbonReportingCycleID, gibbonReportingCriteriaID, gibbonSchoolYearID, gibbonCourseClassID, gibbonPersonIDStudent...` |
| `gibbonResource` | **PLUGIN** | [`07_plugin_behaviour_and_planner.sql`](./07_plugin_behaviour_and_planner.sql) | 11 | `gibbonResourceID` | `gibbonYearGroupIDList, gibbonPersonID` |
| `gibbonResourceTag` | **PLUGIN** | [`07_plugin_behaviour_and_planner.sql`](./07_plugin_behaviour_and_planner.sql) | 3 | `gibbonResourceTagID` | `-` |
| `gibbonRole` | **CORE** | [`01_identity_and_people.sql`](./01_identity_and_people.sql) | 10 | `gibbonRoleID` | `-` |
| `gibbonRubric` | **PLUGIN** | [`06_plugin_assessment_and_reporting.sql`](./06_plugin_assessment_and_reporting.sql) | 10 | `gibbonRubricID` | `gibbonDepartmentID, gibbonYearGroupIDList, gibbonScaleID, gibbonPersonIDCreator` |
| `gibbonRubricCell` | **PLUGIN** | [`06_plugin_assessment_and_reporting.sql`](./06_plugin_assessment_and_reporting.sql) | 5 | `gibbonRubricCellID` | `gibbonRubricID, gibbonRubricColumnID, gibbonRubricRowID` |
| `gibbonRubricColumn` | **PLUGIN** | [`06_plugin_assessment_and_reporting.sql`](./06_plugin_assessment_and_reporting.sql) | 7 | `gibbonRubricColumnID` | `gibbonRubricID, gibbonScaleGradeID` |
| `gibbonRubricEntry` | **PLUGIN** | [`06_plugin_assessment_and_reporting.sql`](./06_plugin_assessment_and_reporting.sql) | 6 | `gibbonRubricEntry` | `gibbonRubricID, gibbonPersonID, gibbonRubricCellID, contextDBTableID` |
| `gibbonRubricRow` | **PLUGIN** | [`06_plugin_assessment_and_reporting.sql`](./06_plugin_assessment_and_reporting.sql) | 6 | `gibbonRubricRowID` | `gibbonRubricID, gibbonOutcomeID` |
| `gibbonScale` | **PLUGIN** | [`06_plugin_assessment_and_reporting.sql`](./06_plugin_assessment_and_reporting.sql) | 7 | `gibbonScaleID` | `-` |
| `gibbonScaleGrade` | **PLUGIN** | [`06_plugin_assessment_and_reporting.sql`](./06_plugin_assessment_and_reporting.sql) | 6 | `gibbonScaleGradeID` | `gibbonScaleID` |
| `gibbonSchoolYear` | **CORE** | [`03_school_structure_and_timetable.sql`](./03_school_structure_and_timetable.sql) | 6 | `gibbonSchoolYearID` | `-` |
| `gibbonSchoolYearSpecialDay` | **CORE** | [`03_school_structure_and_timetable.sql`](./03_school_structure_and_timetable.sql) | 14 | `gibbonSchoolYearSpecialDayID` | `gibbonSchoolYearTermID, gibbonYearGroupIDList, gibbonFormGroupIDList` |
| `gibbonSchoolYearTerm` | **CORE** | [`03_school_structure_and_timetable.sql`](./03_school_structure_and_timetable.sql) | 7 | `gibbonSchoolYearTermID` | `gibbonSchoolYearID` |
| `gibbonSession` | **DROP** | [`09_system_lookups_and_legacy.sql`](./09_system_lookups_and_legacy.sql) | 7 | `gibbonSessionID` | `gibbonPersonID, gibbonActionID` |
| `gibbonSetting` | **DROP** | [`09_system_lookups_and_legacy.sql`](./09_system_lookups_and_legacy.sql) | 6 | `gibbonSettingID` | `-` |
| `gibbonSpace` | **CORE** | [`03_school_structure_and_timetable.sql`](./03_school_structure_and_timetable.sql) | 16 | `gibbonSpaceID` | `-` |
| `gibbonSpacePerson` | **CORE** | [`03_school_structure_and_timetable.sql`](./03_school_structure_and_timetable.sql) | 4 | `gibbonSpacePersonID` | `gibbonSpaceID, gibbonPersonID` |
| `gibbonStaff` | **CORE** | [`01_identity_and_people.sql`](./01_identity_and_people.sql) | 16 | `gibbonStaffID` | `gibbonPersonID` |
| `gibbonStaffAbsence` | **CORE** | [`01_identity_and_people.sql`](./01_identity_and_people.sql) | 18 | `gibbonStaffAbsenceID` | `gibbonStaffAbsenceTypeID, gibbonSchoolYearID, gibbonPersonID, gibbonPersonIDApproval, gibbonPersonIDCreator...` |
| `gibbonStaffAbsenceDate` | **CORE** | [`01_identity_and_people.sql`](./01_identity_and_people.sql) | 7 | `gibbonStaffAbsenceDateID` | `gibbonStaffAbsenceID` |
| `gibbonStaffAbsenceType` | **CORE** | [`01_identity_and_people.sql`](./01_identity_and_people.sql) | 7 | `gibbonStaffAbsenceTypeID` | `-` |
| `gibbonStaffApplicationForm` | **PLUGIN** | [`01_identity_and_people.sql`](./01_identity_and_people.sql) | 33 | `gibbonStaffApplicationFormID` | `gibbonStaffJobOpeningID, gibbonPersonID` |
| `gibbonStaffApplicationFormFile` | **PLUGIN** | [`01_identity_and_people.sql`](./01_identity_and_people.sql) | 4 | `gibbonStaffApplicationFormFileID` | `gibbonStaffApplicationFormID` |
| `gibbonStaffContract` | **CORE** | [`01_identity_and_people.sql`](./01_identity_and_people.sql) | 25 | `gibbonStaffContractID` | `gibbonStaffID, gibbonPersonIDCreator` |
| `gibbonStaffCoverage` | **CORE** | [`01_identity_and_people.sql`](./01_identity_and_people.sql) | 17 | `gibbonStaffCoverageID` | `gibbonStaffAbsenceID, gibbonSchoolYearID, gibbonPersonID, gibbonPersonIDStatus, gibbonPersonIDCoverage` |
| `gibbonStaffCoverageDate` | **CORE** | [`01_identity_and_people.sql`](./01_identity_and_people.sql) | 12 | `gibbonStaffCoverageDateID` | `gibbonStaffCoverageID, gibbonStaffAbsenceDateID, foreignTableID, gibbonPersonIDUnavailable` |
| `gibbonStaffDuty` | **CORE** | [`01_identity_and_people.sql`](./01_identity_and_people.sql) | 7 | `gibbonStaffDutyID` | `gibbonDaysOfWeekIDList` |
| `gibbonStaffDutyPerson` | **CORE** | [`01_identity_and_people.sql`](./01_identity_and_people.sql) | 4 | `gibbonStaffDutyPersonID` | `gibbonStaffDutyID, gibbonDaysOfWeekID, gibbonPersonID` |
| `gibbonStaffJobOpening` | **PLUGIN** | [`01_identity_and_people.sql`](./01_identity_and_people.sql) | 8 | `gibbonStaffJobOpeningID` | `gibbonPersonIDCreator` |
| `gibbonStaffUpdate` | **CORE** | [`01_identity_and_people.sql`](./01_identity_and_people.sql) | 16 | `gibbonStaffUpdateID` | `gibbonSchoolYearID, gibbonStaffID, gibbonPersonIDUpdater` |
| `gibbonString` | **DROP** | [`09_system_lookups_and_legacy.sql`](./09_system_lookups_and_legacy.sql) | 6 | `gibbonStringID` | `-` |
| `gibbonStudentEnrolment` | **CORE** | [`01_identity_and_people.sql`](./01_identity_and_people.sql) | 7 | `gibbonStudentEnrolmentID` | `gibbonPersonID, gibbonSchoolYearID, gibbonYearGroupID, gibbonFormGroupID` |
| `gibbonStudentNote` | **CORE** | [`05_sen_medical_and_pastoral.sql`](./05_sen_medical_and_pastoral.sql) | 7 | `gibbonStudentNoteID` | `gibbonPersonID, gibbonStudentNoteCategoryID, gibbonPersonIDCreator` |
| `gibbonStudentNoteCategory` | **CORE** | [`05_sen_medical_and_pastoral.sql`](./05_sen_medical_and_pastoral.sql) | 4 | `gibbonStudentNoteCategoryID` | `-` |
| `gibbonSubstitute` | **CORE** | [`01_identity_and_people.sql`](./01_identity_and_people.sql) | 6 | `gibbonSubstituteID` | `gibbonPersonID` |
| `gibbonTT` | **CORE** | [`03_school_structure_and_timetable.sql`](./03_school_structure_and_timetable.sql) | 7 | `gibbonTTID` | `gibbonSchoolYearID, gibbonYearGroupIDList` |
| `gibbonTTColumn` | **CORE** | [`03_school_structure_and_timetable.sql`](./03_school_structure_and_timetable.sql) | 3 | `gibbonTTColumnID` | `-` |
| `gibbonTTColumnRow` | **CORE** | [`03_school_structure_and_timetable.sql`](./03_school_structure_and_timetable.sql) | 7 | `gibbonTTColumnRowID` | `gibbonTTColumnID` |
| `gibbonTTDay` | **CORE** | [`03_school_structure_and_timetable.sql`](./03_school_structure_and_timetable.sql) | 7 | `gibbonTTDayID` | `gibbonTTID, gibbonTTColumnID` |
| `gibbonTTDayDate` | **CORE** | [`03_school_structure_and_timetable.sql`](./03_school_structure_and_timetable.sql) | 3 | `gibbonTTDayDateID` | `gibbonTTDayID` |
| `gibbonTTDayRowClass` | **CORE** | [`03_school_structure_and_timetable.sql`](./03_school_structure_and_timetable.sql) | 5 | `gibbonTTDayRowClassID` | `gibbonTTColumnRowID, gibbonTTDayID, gibbonCourseClassID, gibbonSpaceID` |
| `gibbonTTDayRowClassException` | **CORE** | [`03_school_structure_and_timetable.sql`](./03_school_structure_and_timetable.sql) | 3 | `gibbonTTDayRowClassExceptionID` | `gibbonTTDayRowClassID, gibbonPersonID` |
| `gibbonTTImport` | **PLUGIN** | [`03_school_structure_and_timetable.sql`](./03_school_structure_and_timetable.sql) | 7 | `gibbonTTImportID` | `-` |
| `gibbonTTSpaceBooking` | **PLUGIN** | [`03_school_structure_and_timetable.sql`](./03_school_structure_and_timetable.sql) | 8 | `gibbonTTSpaceBookingID` | `foreignKeyID, gibbonPersonID` |
| `gibbonTTSpaceChange` | **CORE** | [`03_school_structure_and_timetable.sql`](./03_school_structure_and_timetable.sql) | 5 | `gibbonTTSpaceChangeID` | `gibbonTTDayRowClassID, gibbonSpaceID, gibbonPersonID` |
| `gibbonTheme` | **DROP** | [`09_system_lookups_and_legacy.sql`](./09_system_lookups_and_legacy.sql) | 7 | `gibbonThemeID` | `-` |
| `gibbonUnit` | **PLUGIN** | [`07_plugin_behaviour_and_planner.sql`](./07_plugin_behaviour_and_planner.sql) | 14 | `gibbonUnitID` | `gibbonCourseID, gibbonPersonIDCreator, gibbonPersonIDLastEdit` |
| `gibbonUnitBlock` | **PLUGIN** | [`07_plugin_behaviour_and_planner.sql`](./07_plugin_behaviour_and_planner.sql) | 8 | `gibbonUnitBlockID` | `gibbonUnitID` |
| `gibbonUnitClass` | **PLUGIN** | [`07_plugin_behaviour_and_planner.sql`](./07_plugin_behaviour_and_planner.sql) | 4 | `gibbonUnitClassID` | `gibbonUnitID, gibbonCourseClassID` |
| `gibbonUnitClassBlock` | **PLUGIN** | [`07_plugin_behaviour_and_planner.sql`](./07_plugin_behaviour_and_planner.sql) | 11 | `gibbonUnitClassBlockID` | `gibbonUnitClassID, gibbonPlannerEntryID, gibbonUnitBlockID` |
| `gibbonUnitOutcome` | **PLUGIN** | [`07_plugin_behaviour_and_planner.sql`](./07_plugin_behaviour_and_planner.sql) | 5 | `gibbonUnitOutcomeID` | `gibbonUnitID, gibbonOutcomeID` |
| `gibbonUsernameFormat` | **DROP** | [`09_system_lookups_and_legacy.sql`](./09_system_lookups_and_legacy.sql) | 8 | `gibbonUsernameFormatID` | `gibbonRoleIDList` |
| `gibbonYearGroup` | **CORE** | [`03_school_structure_and_timetable.sql`](./03_school_structure_and_timetable.sql) | 5 | `gibbonYearGroupID` | `gibbonPersonIDHOY` |
| `gibboni18n` | **DROP** | [`09_system_lookups_and_legacy.sql`](./09_system_lookups_and_legacy.sql) | 11 | `gibboni18nID` | `-` |

-- ============================================================================
-- Gibbon Blueprint Domain File: 06_plugin_assessment_and_reporting.sql
-- Tables in this file: 34
-- Note: Stripped of INSERT seed data for fast AI agent reading.
-- ============================================================================

-- [TABLE] gibbonCrowdAssessDiscuss | [TRIAGE] PLUGIN | [CATEGORY] Assessment, Markbook, Grades & Reports
CREATE TABLE `gibbonCrowdAssessDiscuss` (
  `gibbonCrowdAssessDiscussID` int(16) UNSIGNED ZEROFILL NOT NULL,
  `gibbonPlannerEntryHomeworkID` int(16) UNSIGNED ZEROFILL NOT NULL,
  `gibbonPersonID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `timestamp` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `comment` text NOT NULL,
  `gibbonCrowdAssessDiscussIDReplyTo` int(16) UNSIGNED ZEROFILL DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonCrowdAssessDiscuss`
  ADD PRIMARY KEY (`gibbonCrowdAssessDiscussID`);

-- [TABLE] gibbonExternalAssessment | [TRIAGE] PLUGIN | [CATEGORY] Assessment, Markbook, Grades & Reports
CREATE TABLE `gibbonExternalAssessment` (
  `gibbonExternalAssessmentID` int(4) UNSIGNED ZEROFILL NOT NULL,
  `name` varchar(50) NOT NULL,
  `nameShort` varchar(10) NOT NULL,
  `description` varchar(255) NOT NULL,
  `website` text NOT NULL,
  `active` enum('Y','N') NOT NULL,
  `allowFileUpload` enum('Y','N') NOT NULL DEFAULT 'N'
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonExternalAssessment`
  ADD PRIMARY KEY (`gibbonExternalAssessmentID`);

-- [TABLE] gibbonExternalAssessmentField | [TRIAGE] PLUGIN | [CATEGORY] Assessment, Markbook, Grades & Reports
CREATE TABLE `gibbonExternalAssessmentField` (
  `gibbonExternalAssessmentFieldID` int(6) UNSIGNED ZEROFILL NOT NULL,
  `gibbonExternalAssessmentID` int(4) UNSIGNED ZEROFILL NOT NULL,
  `name` varchar(50) NOT NULL,
  `category` varchar(50) NOT NULL,
  `order` int(4) NOT NULL,
  `gibbonScaleID` int(5) UNSIGNED ZEROFILL NOT NULL,
  `gibbonYearGroupIDList` varchar(255) DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonExternalAssessmentField`
  ADD PRIMARY KEY (`gibbonExternalAssessmentFieldID`),
  ADD KEY `gibbonExternalAssessmentID` (`gibbonExternalAssessmentID`);

-- [TABLE] gibbonExternalAssessmentStudent | [TRIAGE] PLUGIN | [CATEGORY] Assessment, Markbook, Grades & Reports
CREATE TABLE `gibbonExternalAssessmentStudent` (
  `gibbonExternalAssessmentStudentID` int(12) UNSIGNED ZEROFILL NOT NULL,
  `gibbonExternalAssessmentID` int(4) UNSIGNED ZEROFILL NOT NULL,
  `gibbonPersonID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `date` date NOT NULL,
  `attachment` varchar(255) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonExternalAssessmentStudent`
  ADD PRIMARY KEY (`gibbonExternalAssessmentStudentID`),
  ADD KEY `gibbonExternalAssessmentID` (`gibbonExternalAssessmentID`),
  ADD KEY `gibbonPersonID` (`gibbonPersonID`);

-- [TABLE] gibbonExternalAssessmentStudentEntry | [TRIAGE] PLUGIN | [CATEGORY] Assessment, Markbook, Grades & Reports
CREATE TABLE `gibbonExternalAssessmentStudentEntry` (
  `gibbonExternalAssessmentStudentEntryID` int(14) UNSIGNED ZEROFILL NOT NULL,
  `gibbonExternalAssessmentStudentID` int(12) UNSIGNED ZEROFILL NOT NULL,
  `gibbonExternalAssessmentFieldID` int(6) UNSIGNED ZEROFILL NOT NULL,
  `gibbonScaleGradeID` int(7) UNSIGNED ZEROFILL DEFAULT NULL COMMENT 'Key for the actual grade achieved'
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonExternalAssessmentStudentEntry`
  ADD PRIMARY KEY (`gibbonExternalAssessmentStudentEntryID`),
  ADD KEY `gibbonExternalAssessmentStudentID` (`gibbonExternalAssessmentStudentID`),
  ADD KEY `gibbonExternalAssessmentFieldID` (`gibbonExternalAssessmentFieldID`),
  ADD KEY `gibbonScaleGradeID` (`gibbonScaleGradeID`);

-- [TABLE] gibbonInternalAssessmentColumn | [TRIAGE] PLUGIN | [CATEGORY] Assessment, Markbook, Grades & Reports
CREATE TABLE `gibbonInternalAssessmentColumn` (
  `gibbonInternalAssessmentColumnID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `gibbonCourseClassID` int(8) UNSIGNED ZEROFILL NOT NULL,
  `groupingID` int(8) UNSIGNED ZEROFILL DEFAULT NULL COMMENT 'A value used to group multiple columns.',
  `name` varchar(30) NOT NULL,
  `description` text NOT NULL,
  `type` varchar(50) NOT NULL,
  `attachment` varchar(255) NOT NULL,
  `attainment` enum('Y','N') NOT NULL DEFAULT 'Y',
  `gibbonScaleIDAttainment` int(5) UNSIGNED ZEROFILL DEFAULT NULL,
  `effort` enum('Y','N') NOT NULL DEFAULT 'Y',
  `gibbonScaleIDEffort` int(5) UNSIGNED ZEROFILL DEFAULT NULL,
  `comment` enum('Y','N') NOT NULL DEFAULT 'Y',
  `uploadedResponse` enum('N','Y') NOT NULL DEFAULT 'N',
  `complete` enum('N','Y') NOT NULL,
  `completeDate` date DEFAULT NULL,
  `viewableStudents` enum('N','Y') NOT NULL,
  `viewableParents` enum('N','Y') NOT NULL,
  `gibbonPersonIDCreator` int(10) UNSIGNED ZEROFILL NOT NULL,
  `gibbonPersonIDLastEdit` int(10) UNSIGNED ZEROFILL NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonInternalAssessmentColumn`
  ADD PRIMARY KEY (`gibbonInternalAssessmentColumnID`);

-- [TABLE] gibbonInternalAssessmentEntry | [TRIAGE] PLUGIN | [CATEGORY] Assessment, Markbook, Grades & Reports
CREATE TABLE `gibbonInternalAssessmentEntry` (
  `gibbonInternalAssessmentEntryID` int(12) UNSIGNED ZEROFILL NOT NULL,
  `gibbonInternalAssessmentColumnID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `gibbonPersonIDStudent` int(10) UNSIGNED ZEROFILL NOT NULL,
  `attainmentValue` varchar(10) DEFAULT NULL,
  `attainmentDescriptor` varchar(100) DEFAULT NULL,
  `effortValue` varchar(10) DEFAULT NULL,
  `effortDescriptor` varchar(100) DEFAULT NULL,
  `comment` text,
  `response` text,
  `gibbonPersonIDLastEdit` int(10) UNSIGNED ZEROFILL NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonInternalAssessmentEntry`
  ADD PRIMARY KEY (`gibbonInternalAssessmentEntryID`);

-- [TABLE] gibbonMarkbookColumn | [TRIAGE] PLUGIN | [CATEGORY] Assessment, Markbook, Grades & Reports
CREATE TABLE `gibbonMarkbookColumn` (
  `gibbonMarkbookColumnID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `gibbonCourseClassID` int(8) UNSIGNED ZEROFILL NOT NULL,
  `gibbonHookID` int(4) UNSIGNED ZEROFILL DEFAULT NULL,
  `gibbonUnitID` int(10) UNSIGNED ZEROFILL DEFAULT NULL,
  `gibbonPlannerEntryID` int(14) UNSIGNED ZEROFILL DEFAULT NULL,
  `gibbonSchoolYearTermID` int(5) UNSIGNED ZEROFILL DEFAULT NULL,
  `groupingID` int(8) UNSIGNED ZEROFILL DEFAULT NULL COMMENT 'A value used to group multiple markbook columns.',
  `type` varchar(50) NOT NULL,
  `name` varchar(40) DEFAULT NULL,
  `description` text NOT NULL,
  `date` date DEFAULT NULL,
  `sequenceNumber` int(3) UNSIGNED NOT NULL DEFAULT '0',
  `attachment` varchar(255) NOT NULL,
  `attainment` enum('Y','N') NOT NULL DEFAULT 'Y',
  `gibbonScaleIDAttainment` int(5) UNSIGNED ZEROFILL DEFAULT NULL,
  `attainmentWeighting` decimal(5,2) DEFAULT NULL,
  `attainmentRaw` enum('Y','N') NOT NULL DEFAULT 'N',
  `attainmentRawMax` decimal(8,2) DEFAULT NULL,
  `effort` enum('Y','N') NOT NULL DEFAULT 'Y',
  `gibbonScaleIDEffort` int(5) UNSIGNED ZEROFILL DEFAULT NULL,
  `gibbonRubricIDAttainment` int(8) UNSIGNED ZEROFILL DEFAULT NULL,
  `gibbonRubricIDEffort` int(8) UNSIGNED ZEROFILL DEFAULT NULL,
  `comment` enum('Y','N') NOT NULL DEFAULT 'Y',
  `uploadedResponse` enum('Y','N') NOT NULL DEFAULT 'Y',
  `complete` enum('N','Y') NOT NULL,
  `completeDate` date DEFAULT NULL,
  `viewableStudents` enum('N','Y') NOT NULL,
  `viewableParents` enum('N','Y') NOT NULL,
  `gibbonPersonIDCreator` int(10) UNSIGNED ZEROFILL NOT NULL,
  `gibbonPersonIDLastEdit` int(10) UNSIGNED ZEROFILL NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonMarkbookColumn`
  ADD PRIMARY KEY (`gibbonMarkbookColumnID`),
  ADD KEY `gibbonCourseClassID` (`gibbonCourseClassID`),
  ADD KEY `completeDate` (`completeDate`),
  ADD KEY `complete` (`complete`);

-- [TABLE] gibbonMarkbookEntry | [TRIAGE] PLUGIN | [CATEGORY] Assessment, Markbook, Grades & Reports
CREATE TABLE `gibbonMarkbookEntry` (
  `gibbonMarkbookEntryID` int(12) UNSIGNED ZEROFILL NOT NULL,
  `gibbonMarkbookColumnID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `gibbonPersonIDStudent` int(10) UNSIGNED ZEROFILL NOT NULL,
  `modifiedAssessment` enum('N','Y') DEFAULT NULL,
  `attainmentValue` varchar(10) DEFAULT NULL,
  `attainmentValueRaw` varchar(10) DEFAULT NULL,
  `attainmentDescriptor` varchar(100) DEFAULT NULL,
  `attainmentConcern` enum('N','Y','P') DEFAULT NULL COMMENT '''P'' denotes that student has exceed their personal target',
  `effortValue` varchar(10) DEFAULT NULL,
  `effortDescriptor` varchar(100) DEFAULT NULL,
  `effortConcern` enum('N','Y') DEFAULT NULL,
  `comment` text,
  `response` varchar(255) DEFAULT NULL,
  `gibbonPersonIDLastEdit` int(10) UNSIGNED ZEROFILL NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonMarkbookEntry`
  ADD PRIMARY KEY (`gibbonMarkbookEntryID`),
  ADD KEY `gibbonPersonIDStudent` (`gibbonPersonIDStudent`),
  ADD KEY `gibbonMarkbookColumnID` (`gibbonMarkbookColumnID`);

-- [TABLE] gibbonMarkbookTarget | [TRIAGE] PLUGIN | [CATEGORY] Assessment, Markbook, Grades & Reports
CREATE TABLE `gibbonMarkbookTarget` (
  `gibbonMarkbookTargetID` int(14) UNSIGNED ZEROFILL NOT NULL,
  `gibbonCourseClassID` int(8) UNSIGNED ZEROFILL NOT NULL,
  `gibbonPersonIDStudent` int(10) UNSIGNED ZEROFILL NOT NULL,
  `gibbonScaleGradeID` int(7) UNSIGNED ZEROFILL DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonMarkbookTarget`
  ADD PRIMARY KEY (`gibbonMarkbookTargetID`),
  ADD UNIQUE KEY `coursePerson` (`gibbonCourseClassID`,`gibbonPersonIDStudent`);

-- [TABLE] gibbonMarkbookWeight | [TRIAGE] PLUGIN | [CATEGORY] Assessment, Markbook, Grades & Reports
CREATE TABLE `gibbonMarkbookWeight` (
  `gibbonMarkbookWeightID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `gibbonCourseClassID` int(8) UNSIGNED ZEROFILL NOT NULL,
  `type` varchar(50) NOT NULL,
  `description` varchar(50) NOT NULL,
  `reportable` enum('Y','N') NOT NULL DEFAULT 'Y',
  `calculate` enum('term','year') NOT NULL DEFAULT 'year',
  `weighting` decimal(5,2) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonMarkbookWeight`
  ADD PRIMARY KEY (`gibbonMarkbookWeightID`);

-- [TABLE] gibbonOutcome | [TRIAGE] PLUGIN | [CATEGORY] Assessment, Markbook, Grades & Reports
CREATE TABLE `gibbonOutcome` (
  `gibbonOutcomeID` int(8) UNSIGNED ZEROFILL NOT NULL,
  `name` varchar(100) NOT NULL,
  `nameShort` varchar(14) NOT NULL,
  `category` varchar(50) NOT NULL,
  `description` text NOT NULL,
  `active` enum('Y','N') NOT NULL,
  `scope` enum('School','Learning Area') NOT NULL,
  `gibbonDepartmentID` int(4) UNSIGNED ZEROFILL DEFAULT NULL,
  `gibbonYearGroupIDList` varchar(255) NOT NULL,
  `gibbonPersonIDCreator` int(10) UNSIGNED ZEROFILL NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonOutcome`
  ADD PRIMARY KEY (`gibbonOutcomeID`);

-- [TABLE] gibbonReport | [TRIAGE] PLUGIN | [CATEGORY] Assessment, Markbook, Grades & Reports
CREATE TABLE `gibbonReport` (
  `gibbonReportID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `gibbonReportArchiveID` int(5) UNSIGNED ZEROFILL DEFAULT NULL,
  `gibbonReportTemplateID` int(10) UNSIGNED ZEROFILL DEFAULT NULL,
  `gibbonSchoolYearID` int(3) UNSIGNED ZEROFILL DEFAULT NULL,
  `name` varchar(90) DEFAULT NULL,
  `active` enum('Y','N') NOT NULL DEFAULT 'Y',
  `status` enum('None','Draft','Published') NOT NULL DEFAULT 'None',
  `gibbonReportingCycleID` int(10) UNSIGNED ZEROFILL DEFAULT NULL,
  `gibbonYearGroupIDList` varchar(255) DEFAULT NULL,
  `queryBuilderQueryID` int(10) UNSIGNED ZEROFILL DEFAULT NULL,
  `accessDate` datetime DEFAULT NULL,
  `timestampModified` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `timestampGenerated` timestamp NULL DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonReport`
  ADD PRIMARY KEY (`gibbonReportID`);

-- [TABLE] gibbonReportArchive | [TRIAGE] PLUGIN | [CATEGORY] Assessment, Markbook, Grades & Reports
CREATE TABLE `gibbonReportArchive` (
  `gibbonReportArchiveID` int(5) UNSIGNED ZEROFILL NOT NULL,
  `name` varchar(90) DEFAULT NULL,
  `path` varchar(255) DEFAULT NULL,
  `readonly` enum('Y','N') NOT NULL DEFAULT 'N',
  `viewableStaff` enum('Y','N') NOT NULL DEFAULT 'N',
  `viewableStudents` enum('Y','N') NOT NULL DEFAULT 'N',
  `viewableParents` enum('Y','N') NOT NULL DEFAULT 'N',
  `viewableOther` enum('Y','N') NOT NULL DEFAULT 'N'
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonReportArchive`
  ADD PRIMARY KEY (`gibbonReportArchiveID`);

-- [TABLE] gibbonReportArchiveEntry | [TRIAGE] PLUGIN | [CATEGORY] Assessment, Markbook, Grades & Reports
CREATE TABLE `gibbonReportArchiveEntry` (
  `gibbonReportArchiveEntryID` int(12) UNSIGNED ZEROFILL NOT NULL,
  `gibbonReportArchiveID` int(5) UNSIGNED ZEROFILL DEFAULT NULL,
  `gibbonReportID` int(10) UNSIGNED ZEROFILL DEFAULT NULL,
  `gibbonSchoolYearID` int(3) UNSIGNED ZEROFILL DEFAULT NULL,
  `gibbonYearGroupID` int(3) UNSIGNED ZEROFILL DEFAULT NULL,
  `gibbonFormGroupID` int(5) UNSIGNED ZEROFILL DEFAULT NULL,
  `gibbonPersonID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `type` enum('Single','Batch') NOT NULL DEFAULT 'Single',
  `status` enum('Draft','Final') NOT NULL DEFAULT 'Draft',
  `reportIdentifier` varchar(255) DEFAULT NULL,
  `filePath` varchar(255) DEFAULT NULL,
  `timestampCreated` timestamp NULL DEFAULT NULL,
  `timestampModified` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `timestampSent` timestamp NULL DEFAULT NULL,
  `gibbonPersonIDAccessed` int(10) UNSIGNED ZEROFILL DEFAULT NULL,
  `timestampAccessed` timestamp NULL DEFAULT NULL,
  `timestampAccessExpiry` timestamp NULL DEFAULT NULL,
  `accessToken` varchar(60) DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonReportArchiveEntry`
  ADD PRIMARY KEY (`gibbonReportArchiveEntryID`),
  ADD UNIQUE KEY `archiveEntry` (`gibbonReportArchiveID`,`gibbonSchoolYearID`,`reportIdentifier`,`type`,`gibbonYearGroupID`,`gibbonPersonID`);

-- [TABLE] gibbonReportPrototypeSection | [TRIAGE] PLUGIN | [CATEGORY] Assessment, Markbook, Grades & Reports
CREATE TABLE `gibbonReportPrototypeSection` (
  `gibbonReportPrototypeSectionID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `name` varchar(90) DEFAULT NULL,
  `type` enum('Core','Additional') NOT NULL DEFAULT 'Additional',
  `category` varchar(90) DEFAULT NULL,
  `types` varchar(60) DEFAULT NULL,
  `config` text,
  `templateParams` text,
  `templateFile` varchar(255) DEFAULT NULL,
  `dataSources` text,
  `icon` varchar(120) DEFAULT NULL,
  `fonts` varchar(255) DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonReportPrototypeSection`
  ADD PRIMARY KEY (`gibbonReportPrototypeSectionID`),
  ADD UNIQUE KEY `type` (`type`,`templateFile`);

-- [TABLE] gibbonReportTemplate | [TRIAGE] PLUGIN | [CATEGORY] Assessment, Markbook, Grades & Reports
CREATE TABLE `gibbonReportTemplate` (
  `gibbonReportTemplateID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `context` enum('Student Enrolment','Reporting Cycle','Custom Query') NOT NULL DEFAULT 'Student Enrolment',
  `name` varchar(90) DEFAULT NULL,
  `active` enum('Y','N') NOT NULL DEFAULT 'Y',
  `orientation` char(1) NOT NULL DEFAULT 'P',
  `pageSize` varchar(20) NOT NULL DEFAULT 'A4',
  `marginX` decimal(4,2) NOT NULL DEFAULT '10.00',
  `marginY` decimal(4,2) NOT NULL DEFAULT '10.00',
  `flags` int(10) UNSIGNED ZEROFILL DEFAULT NULL,
  `config` text,
  `stylesheet` varchar(120) DEFAULT NULL,
  `timestamp` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonReportTemplate`
  ADD PRIMARY KEY (`gibbonReportTemplateID`);

-- [TABLE] gibbonReportTemplateFont | [TRIAGE] PLUGIN | [CATEGORY] Assessment, Markbook, Grades & Reports
CREATE TABLE `gibbonReportTemplateFont` (
  `gibbonReportTemplateFontID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `fontName` varchar(90) NOT NULL,
  `fontTCPDF` varchar(90) DEFAULT NULL,
  `fontPath` text,
  `fontType` enum('R','B','I','BI') NOT NULL DEFAULT 'R',
  `fontFamily` varchar(60) DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonReportTemplateFont`
  ADD PRIMARY KEY (`gibbonReportTemplateFontID`),
  ADD UNIQUE KEY `fontTCPDF` (`fontTCPDF`);

-- [TABLE] gibbonReportTemplateSection | [TRIAGE] PLUGIN | [CATEGORY] Assessment, Markbook, Grades & Reports
CREATE TABLE `gibbonReportTemplateSection` (
  `gibbonReportTemplateSectionID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `gibbonReportTemplateID` int(10) UNSIGNED ZEROFILL DEFAULT NULL,
  `gibbonReportPrototypeSectionID` int(10) UNSIGNED ZEROFILL DEFAULT NULL,
  `name` varchar(90) DEFAULT NULL,
  `type` enum('Body','Header','Footer') NOT NULL DEFAULT 'Body',
  `flags` int(10) UNSIGNED ZEROFILL DEFAULT NULL,
  `page` int(3) NOT NULL DEFAULT '0',
  `config` text,
  `sequenceNumber` int(3) UNSIGNED DEFAULT NULL,
  `templateParams` text,
  `templateContent` text
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonReportTemplateSection`
  ADD PRIMARY KEY (`gibbonReportTemplateSectionID`);

-- [TABLE] gibbonReportingAccess | [TRIAGE] PLUGIN | [CATEGORY] Assessment, Markbook, Grades & Reports
CREATE TABLE `gibbonReportingAccess` (
  `gibbonReportingAccessID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `gibbonReportingCycleID` int(10) UNSIGNED ZEROFILL DEFAULT NULL,
  `gibbonReportingScopeIDList` varchar(255) DEFAULT NULL,
  `gibbonRoleIDList` varchar(255) DEFAULT NULL,
  `gibbonPersonIDList` varchar(255) DEFAULT NULL,
  `accessType` enum('Role','Person') NOT NULL DEFAULT 'Role',
  `dateStart` date DEFAULT NULL,
  `dateEnd` date DEFAULT NULL,
  `canWrite` enum('Y','N') NOT NULL DEFAULT 'N',
  `canProofRead` enum('Y','N') NOT NULL DEFAULT 'N'
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonReportingAccess`
  ADD PRIMARY KEY (`gibbonReportingAccessID`);

-- [TABLE] gibbonReportingCriteria | [TRIAGE] PLUGIN | [CATEGORY] Assessment, Markbook, Grades & Reports
CREATE TABLE `gibbonReportingCriteria` (
  `gibbonReportingCriteriaID` int(12) UNSIGNED ZEROFILL NOT NULL,
  `gibbonReportingCycleID` int(10) UNSIGNED ZEROFILL DEFAULT NULL,
  `gibbonReportingScopeID` int(10) UNSIGNED ZEROFILL DEFAULT NULL,
  `gibbonReportingCriteriaTypeID` int(10) UNSIGNED ZEROFILL DEFAULT NULL,
  `gibbonYearGroupID` int(3) UNSIGNED ZEROFILL DEFAULT NULL,
  `gibbonFormGroupID` int(5) UNSIGNED ZEROFILL DEFAULT NULL,
  `gibbonCourseID` int(8) UNSIGNED ZEROFILL DEFAULT NULL,
  `target` enum('Per Student','Per Group') NOT NULL DEFAULT 'Per Student',
  `name` varchar(255) DEFAULT NULL,
  `description` text,
  `category` varchar(255) DEFAULT NULL,
  `gibbonScaleID` int(5) UNSIGNED ZEROFILL DEFAULT NULL,
  `sequenceNumber` int(3) DEFAULT NULL,
  `groupID` varchar(255) DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonReportingCriteria`
  ADD PRIMARY KEY (`gibbonReportingCriteriaID`);

-- [TABLE] gibbonReportingCriteriaType | [TRIAGE] PLUGIN | [CATEGORY] Assessment, Markbook, Grades & Reports
CREATE TABLE `gibbonReportingCriteriaType` (
  `gibbonReportingCriteriaTypeID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `name` varchar(90) DEFAULT NULL,
  `valueType` varchar(30) DEFAULT NULL,
  `defaultValue` varchar(255) DEFAULT NULL,
  `active` enum('Y','N') NOT NULL DEFAULT 'Y',
  `characterLimit` int(6) DEFAULT NULL,
  `gibbonScaleID` int(5) UNSIGNED ZEROFILL DEFAULT NULL,
  `options` text
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonReportingCriteriaType`
  ADD PRIMARY KEY (`gibbonReportingCriteriaTypeID`),
  ADD UNIQUE KEY `name` (`name`);

-- [TABLE] gibbonReportingCycle | [TRIAGE] PLUGIN | [CATEGORY] Assessment, Markbook, Grades & Reports
CREATE TABLE `gibbonReportingCycle` (
  `gibbonReportingCycleID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `gibbonSchoolYearID` int(3) UNSIGNED ZEROFILL DEFAULT NULL,
  `gibbonYearGroupIDList` varchar(255) DEFAULT NULL,
  `name` varchar(90) DEFAULT NULL,
  `nameShort` varchar(20) DEFAULT NULL,
  `sequenceNumber` int(3) DEFAULT NULL,
  `cycleNumber` int(3) DEFAULT NULL,
  `cycleTotal` int(3) DEFAULT NULL,
  `dateStart` date DEFAULT NULL,
  `dateEnd` date DEFAULT NULL,
  `milestones` text,
  `notes` text
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonReportingCycle`
  ADD PRIMARY KEY (`gibbonReportingCycleID`);

-- [TABLE] gibbonReportingProgress | [TRIAGE] PLUGIN | [CATEGORY] Assessment, Markbook, Grades & Reports
CREATE TABLE `gibbonReportingProgress` (
  `gibbonReportingProgressID` int(12) UNSIGNED ZEROFILL NOT NULL,
  `gibbonReportingScopeID` int(10) UNSIGNED ZEROFILL DEFAULT NULL,
  `gibbonYearGroupID` int(3) UNSIGNED ZEROFILL DEFAULT NULL,
  `gibbonFormGroupID` int(5) UNSIGNED ZEROFILL DEFAULT NULL,
  `gibbonCourseClassID` int(8) UNSIGNED ZEROFILL NOT NULL,
  `gibbonPersonIDStudent` int(10) UNSIGNED ZEROFILL NOT NULL,
  `status` enum('In Progress','Complete') NOT NULL DEFAULT 'In Progress'
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonReportingProgress`
  ADD PRIMARY KEY (`gibbonReportingProgressID`),
  ADD UNIQUE KEY `gibbonReportingScopeID` (`gibbonReportingScopeID`,`gibbonCourseClassID`,`gibbonPersonIDStudent`);

-- [TABLE] gibbonReportingProof | [TRIAGE] PLUGIN | [CATEGORY] Assessment, Markbook, Grades & Reports
CREATE TABLE `gibbonReportingProof` (
  `gibbonReportingProofID` int(12) UNSIGNED ZEROFILL NOT NULL,
  `gibbonReportingValueID` int(12) UNSIGNED ZEROFILL DEFAULT NULL,
  `status` enum('Done','Edited','Accepted','Declined') NOT NULL DEFAULT 'Done',
  `reason` varchar(255) DEFAULT NULL,
  `comment` text,
  `gibbonPersonIDProofed` int(10) UNSIGNED ZEROFILL DEFAULT NULL,
  `timestampProofed` timestamp NULL DEFAULT NULL,
  `gibbonPersonIDActioned` int(10) UNSIGNED ZEROFILL DEFAULT NULL,
  `timestampActioned` timestamp NULL DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonReportingProof`
  ADD PRIMARY KEY (`gibbonReportingProofID`),
  ADD UNIQUE KEY `gibbonReportingValueID` (`gibbonReportingValueID`);

-- [TABLE] gibbonReportingScope | [TRIAGE] PLUGIN | [CATEGORY] Assessment, Markbook, Grades & Reports
CREATE TABLE `gibbonReportingScope` (
  `gibbonReportingScopeID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `gibbonReportingCycleID` int(10) UNSIGNED ZEROFILL DEFAULT NULL,
  `scopeType` enum('Year Group','Form Group','Course') NOT NULL DEFAULT 'Year Group',
  `name` varchar(90) DEFAULT NULL,
  `sequenceNumber` int(3) DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonReportingScope`
  ADD PRIMARY KEY (`gibbonReportingScopeID`);

-- [TABLE] gibbonReportingValue | [TRIAGE] PLUGIN | [CATEGORY] Assessment, Markbook, Grades & Reports
CREATE TABLE `gibbonReportingValue` (
  `gibbonReportingValueID` int(12) UNSIGNED ZEROFILL NOT NULL,
  `gibbonReportingCycleID` int(10) UNSIGNED ZEROFILL DEFAULT NULL,
  `gibbonReportingCriteriaID` int(12) UNSIGNED ZEROFILL DEFAULT NULL,
  `gibbonSchoolYearID` int(3) UNSIGNED ZEROFILL DEFAULT NULL,
  `gibbonCourseClassID` int(8) UNSIGNED ZEROFILL NOT NULL,
  `gibbonPersonIDStudent` int(10) UNSIGNED ZEROFILL NOT NULL,
  `gibbonScaleGradeID` int(7) UNSIGNED ZEROFILL DEFAULT NULL,
  `value` varchar(255) DEFAULT NULL,
  `comment` text,
  `gibbonPersonIDCreated` int(10) UNSIGNED ZEROFILL DEFAULT NULL,
  `timestampCreated` timestamp NULL DEFAULT NULL,
  `gibbonPersonIDModified` int(10) UNSIGNED ZEROFILL DEFAULT NULL,
  `timestampModified` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonReportingValue`
  ADD PRIMARY KEY (`gibbonReportingValueID`),
  ADD UNIQUE KEY `gibbonReportingCriteriaID` (`gibbonReportingCriteriaID`,`gibbonPersonIDStudent`,`gibbonCourseClassID`) USING BTREE;

-- [TABLE] gibbonRubric | [TRIAGE] PLUGIN | [CATEGORY] Assessment, Markbook, Grades & Reports
CREATE TABLE `gibbonRubric` (
  `gibbonRubricID` int(8) UNSIGNED ZEROFILL NOT NULL,
  `name` varchar(50) NOT NULL,
  `category` varchar(50) NOT NULL,
  `description` text NOT NULL,
  `active` enum('Y','N') NOT NULL,
  `scope` enum('School','Learning Area') NOT NULL,
  `gibbonDepartmentID` int(4) UNSIGNED ZEROFILL DEFAULT NULL,
  `gibbonYearGroupIDList` varchar(255) NOT NULL,
  `gibbonScaleID` int(5) UNSIGNED ZEROFILL DEFAULT NULL,
  `gibbonPersonIDCreator` int(10) UNSIGNED ZEROFILL NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonRubric`
  ADD PRIMARY KEY (`gibbonRubricID`);

-- [TABLE] gibbonRubricCell | [TRIAGE] PLUGIN | [CATEGORY] Assessment, Markbook, Grades & Reports
CREATE TABLE `gibbonRubricCell` (
  `gibbonRubricCellID` int(11) UNSIGNED ZEROFILL NOT NULL,
  `gibbonRubricID` int(8) UNSIGNED ZEROFILL NOT NULL,
  `gibbonRubricColumnID` int(9) UNSIGNED ZEROFILL NOT NULL,
  `gibbonRubricRowID` int(9) UNSIGNED ZEROFILL NOT NULL,
  `contents` text NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonRubricCell`
  ADD PRIMARY KEY (`gibbonRubricCellID`),
  ADD KEY `gibbonRubricID` (`gibbonRubricID`),
  ADD KEY `gibbonRubricColumnID` (`gibbonRubricColumnID`),
  ADD KEY `gibbonRubricRowID` (`gibbonRubricRowID`);

-- [TABLE] gibbonRubricColumn | [TRIAGE] PLUGIN | [CATEGORY] Assessment, Markbook, Grades & Reports
CREATE TABLE `gibbonRubricColumn` (
  `gibbonRubricColumnID` int(9) UNSIGNED ZEROFILL NOT NULL,
  `gibbonRubricID` int(8) UNSIGNED ZEROFILL NOT NULL,
  `title` varchar(20) NOT NULL,
  `backgroundColor` varchar(7) DEFAULT NULL,
  `sequenceNumber` int(2) NOT NULL,
  `gibbonScaleGradeID` int(7) UNSIGNED ZEROFILL DEFAULT NULL,
  `visualise` enum('Y','N') NOT NULL DEFAULT 'Y'
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonRubricColumn`
  ADD PRIMARY KEY (`gibbonRubricColumnID`),
  ADD KEY `gibbonRubricID` (`gibbonRubricID`);

-- [TABLE] gibbonRubricEntry | [TRIAGE] PLUGIN | [CATEGORY] Assessment, Markbook, Grades & Reports
CREATE TABLE `gibbonRubricEntry` (
  `gibbonRubricEntry` int(14) UNSIGNED ZEROFILL NOT NULL,
  `gibbonRubricID` int(8) UNSIGNED ZEROFILL NOT NULL,
  `gibbonPersonID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `gibbonRubricCellID` int(11) UNSIGNED ZEROFILL NOT NULL,
  `contextDBTable` varchar(255) NOT NULL COMMENT 'Which database table is this entry related to?',
  `contextDBTableID` int(20) UNSIGNED ZEROFILL NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonRubricEntry`
  ADD PRIMARY KEY (`gibbonRubricEntry`),
  ADD KEY `gibbonRubricID` (`gibbonRubricID`),
  ADD KEY `gibbonPersonID` (`gibbonPersonID`),
  ADD KEY `gibbonRubricCellID` (`gibbonRubricCellID`),
  ADD KEY `contextDBTable` (`contextDBTable`),
  ADD KEY `contextDBTableID` (`contextDBTableID`);

-- [TABLE] gibbonRubricRow | [TRIAGE] PLUGIN | [CATEGORY] Assessment, Markbook, Grades & Reports
CREATE TABLE `gibbonRubricRow` (
  `gibbonRubricRowID` int(9) UNSIGNED ZEROFILL NOT NULL,
  `gibbonRubricID` int(8) UNSIGNED ZEROFILL NOT NULL,
  `title` varchar(40) NOT NULL,
  `backgroundColor` varchar(7) DEFAULT NULL,
  `sequenceNumber` int(2) NOT NULL,
  `gibbonOutcomeID` int(8) UNSIGNED ZEROFILL DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonRubricRow`
  ADD PRIMARY KEY (`gibbonRubricRowID`),
  ADD KEY `gibbonRubricID` (`gibbonRubricID`);

-- [TABLE] gibbonScale | [TRIAGE] PLUGIN | [CATEGORY] Assessment, Markbook, Grades & Reports
CREATE TABLE `gibbonScale` (
  `gibbonScaleID` int(5) UNSIGNED ZEROFILL NOT NULL,
  `name` varchar(40) NOT NULL,
  `nameShort` varchar(5) NOT NULL,
  `usage` varchar(50) NOT NULL,
  `lowestAcceptable` varchar(5) DEFAULT NULL COMMENT 'This is the sequence number of the lowest grade a student can get without being unsatisfactory',
  `active` enum('Y','N') NOT NULL DEFAULT 'Y',
  `numeric` enum('N','Y') NOT NULL DEFAULT 'N'
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonScale`
  ADD PRIMARY KEY (`gibbonScaleID`);

-- [TABLE] gibbonScaleGrade | [TRIAGE] PLUGIN | [CATEGORY] Assessment, Markbook, Grades & Reports
CREATE TABLE `gibbonScaleGrade` (
  `gibbonScaleGradeID` int(7) UNSIGNED ZEROFILL NOT NULL,
  `gibbonScaleID` int(5) UNSIGNED ZEROFILL NOT NULL,
  `value` varchar(10) NOT NULL,
  `descriptor` varchar(50) NOT NULL,
  `sequenceNumber` int(5) NOT NULL,
  `isDefault` enum('N','Y') NOT NULL DEFAULT 'N'
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonScaleGrade`
  ADD PRIMARY KEY (`gibbonScaleGradeID`);

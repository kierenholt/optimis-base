-- ============================================================================
-- Gibbon Blueprint Domain File: 05_sen_medical_and_pastoral.sql
-- Tables in this file: 17
-- Note: Stripped of INSERT seed data for fast AI agent reading.
-- ============================================================================

-- [TABLE] gibbonAlertLevel | [TRIAGE] CORE | [CATEGORY] SEN/Individual Needs, Medical & Pastoral
CREATE TABLE `gibbonAlertLevel` (
  `gibbonAlertLevelID` int(3) UNSIGNED ZEROFILL NOT NULL,
  `name` varchar(50) NOT NULL,
  `nameShort` varchar(4) NOT NULL,
  `color` varchar(7) NOT NULL COMMENT 'RGB Hex, leading #',
  `colorBG` varchar(7) NOT NULL COMMENT 'RGB Hex, leading #',
  `description` text NOT NULL,
  `sequenceNumber` int(3) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonAlertLevel`
  ADD PRIMARY KEY (`gibbonAlertLevelID`);

-- [TABLE] gibbonFirstAid | [TRIAGE] PLUGIN | [CATEGORY] SEN/Individual Needs, Medical & Pastoral
CREATE TABLE `gibbonFirstAid` (
  `gibbonFirstAidID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `gibbonPersonIDPatient` int(10) UNSIGNED ZEROFILL NOT NULL,
  `gibbonCourseClassID` int(8) UNSIGNED ZEROFILL DEFAULT NULL,
  `gibbonPersonIDFirstAider` int(10) UNSIGNED ZEROFILL NOT NULL,
  `gibbonPersonIDFollowUp` int(10) UNSIGNED ZEROFILL DEFAULT NULL,
  `gibbonSchoolYearID` int(3) UNSIGNED ZEROFILL NOT NULL,
  `description` text NOT NULL,
  `actionTaken` text NOT NULL,
  `followUp` text NOT NULL,
  `date` date NOT NULL,
  `timeIn` time NOT NULL,
  `timeOut` time DEFAULT NULL,
  `timestamp` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `fields` text
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonFirstAid`
  ADD PRIMARY KEY (`gibbonFirstAidID`);

-- [TABLE] gibbonFirstAidFollowUp | [TRIAGE] PLUGIN | [CATEGORY] SEN/Individual Needs, Medical & Pastoral
CREATE TABLE `gibbonFirstAidFollowUp` (
  `gibbonFirstAidFollowUpID` int(11) UNSIGNED ZEROFILL NOT NULL,
  `gibbonFirstAidID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `gibbonPersonID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `followUp` text NOT NULL,
  `timestamp` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonFirstAidFollowUp`
  ADD PRIMARY KEY (`gibbonFirstAidFollowUpID`),
  ADD KEY `gibbonFirstAidID` (`gibbonFirstAidID`);

-- [TABLE] gibbonIN | [TRIAGE] CORE | [CATEGORY] SEN/Individual Needs, Medical & Pastoral
CREATE TABLE `gibbonIN` (
  `gibbonINID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `gibbonPersonID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `strategies` text NOT NULL,
  `targets` text NOT NULL,
  `notes` text NOT NULL,
  `fields` text
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonIN`
  ADD PRIMARY KEY (`gibbonINID`),
  ADD UNIQUE KEY `gibbonPersonID` (`gibbonPersonID`);

-- [TABLE] gibbonINArchive | [TRIAGE] CORE | [CATEGORY] SEN/Individual Needs, Medical & Pastoral
CREATE TABLE `gibbonINArchive` (
  `gibbonINArchiveID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `gibbonPersonID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `strategies` text NOT NULL,
  `targets` text NOT NULL,
  `notes` text NOT NULL,
  `descriptors` text NOT NULL COMMENT 'Serialised array of descriptors.',
  `archiveTitle` varchar(50) NOT NULL,
  `archiveTimestamp` timestamp NULL DEFAULT NULL,
  `fields` text
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonINArchive`
  ADD PRIMARY KEY (`gibbonINArchiveID`);

-- [TABLE] gibbonINAssistant | [TRIAGE] CORE | [CATEGORY] SEN/Individual Needs, Medical & Pastoral
CREATE TABLE `gibbonINAssistant` (
  `gibbonINAssistantID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `gibbonPersonIDStudent` int(10) UNSIGNED ZEROFILL NOT NULL,
  `gibbonPersonIDAssistant` int(10) UNSIGNED ZEROFILL NOT NULL,
  `comment` text COLLATE utf8_unicode_ci NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_unicode_ci;
ALTER TABLE `gibbonINAssistant`
  ADD PRIMARY KEY (`gibbonINAssistantID`);

-- [TABLE] gibbonINDescriptor | [TRIAGE] CORE | [CATEGORY] SEN/Individual Needs, Medical & Pastoral
CREATE TABLE `gibbonINDescriptor` (
  `gibbonINDescriptorID` int(3) UNSIGNED ZEROFILL NOT NULL,
  `name` varchar(50) NOT NULL,
  `nameShort` varchar(5) NOT NULL,
  `description` text NOT NULL,
  `sequenceNumber` int(3) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonINDescriptor`
  ADD PRIMARY KEY (`gibbonINDescriptorID`);

-- [TABLE] gibbonINInvestigation | [TRIAGE] CORE | [CATEGORY] SEN/Individual Needs, Medical & Pastoral
CREATE TABLE `gibbonINInvestigation` (
  `gibbonINInvestigationID` int(11) UNSIGNED ZEROFILL NOT NULL,
  `gibbonSchoolYearID` int(3) UNSIGNED ZEROFILL NOT NULL,
  `gibbonPersonIDCreator` int(10) UNSIGNED ZEROFILL NOT NULL,
  `gibbonPersonIDStudent` int(10) UNSIGNED ZEROFILL NOT NULL,
  `status` enum('Referral','Resolved','Investigation','Investigation Complete') DEFAULT NULL,
  `date` date NOT NULL,
  `reason` text NOT NULL,
  `strategiesTried` text NOT NULL,
  `parentsInformed` enum('N','Y') NOT NULL DEFAULT 'N',
  `parentsResponse` text,
  `resolutionDetails` text
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonINInvestigation`
  ADD PRIMARY KEY (`gibbonINInvestigationID`);

-- [TABLE] gibbonINInvestigationContribution | [TRIAGE] CORE | [CATEGORY] SEN/Individual Needs, Medical & Pastoral
CREATE TABLE `gibbonINInvestigationContribution` (
  `gibbonINInvestigationContributionID` int(12) UNSIGNED ZEROFILL NOT NULL,
  `gibbonINInvestigationID` int(11) UNSIGNED ZEROFILL NOT NULL,
  `type` enum('Teacher','Head of Year') NOT NULL DEFAULT 'Teacher',
  `gibbonPersonID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `gibbonCourseClassPersonID` int(10) UNSIGNED ZEROFILL DEFAULT NULL,
  `status` enum('Pending','Complete') NOT NULL DEFAULT 'Pending',
  `cognition` text,
  `memory` text,
  `selfManagement` text,
  `attention` text,
  `socialInteraction` text,
  `communication` text,
  `comment` text
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonINInvestigationContribution`
  ADD PRIMARY KEY (`gibbonINInvestigationContributionID`);

-- [TABLE] gibbonINPersonDescriptor | [TRIAGE] CORE | [CATEGORY] SEN/Individual Needs, Medical & Pastoral
CREATE TABLE `gibbonINPersonDescriptor` (
  `gibbonINPersonDescriptorID` int(12) UNSIGNED ZEROFILL NOT NULL,
  `gibbonPersonID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `gibbonINDescriptorID` int(3) UNSIGNED ZEROFILL DEFAULT NULL,
  `gibbonAlertLevelID` int(3) UNSIGNED ZEROFILL DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonINPersonDescriptor`
  ADD PRIMARY KEY (`gibbonINPersonDescriptorID`);

-- [TABLE] gibbonMedicalCondition | [TRIAGE] CORE | [CATEGORY] SEN/Individual Needs, Medical & Pastoral
CREATE TABLE `gibbonMedicalCondition` (
  `gibbonMedicalConditionID` int(4) UNSIGNED ZEROFILL NOT NULL,
  `name` varchar(80) NOT NULL,
  `description` text
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonMedicalCondition`
  ADD PRIMARY KEY (`gibbonMedicalConditionID`),
  ADD UNIQUE KEY `name` (`name`);

-- [TABLE] gibbonPersonMedical | [TRIAGE] CORE | [CATEGORY] SEN/Individual Needs, Medical & Pastoral
CREATE TABLE `gibbonPersonMedical` (
  `gibbonPersonMedicalID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `gibbonPersonID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `longTermMedication` enum('','Y','N') NOT NULL,
  `longTermMedicationDetails` text NOT NULL,
  `comment` text NOT NULL,
  `fields` text
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonPersonMedical`
  ADD PRIMARY KEY (`gibbonPersonMedicalID`),
  ADD KEY `gibbonPersonID` (`gibbonPersonID`);

-- [TABLE] gibbonPersonMedicalCondition | [TRIAGE] CORE | [CATEGORY] SEN/Individual Needs, Medical & Pastoral
CREATE TABLE `gibbonPersonMedicalCondition` (
  `gibbonPersonMedicalConditionID` int(12) UNSIGNED ZEROFILL NOT NULL,
  `gibbonPersonMedicalID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `name` varchar(100) NOT NULL,
  `gibbonAlertLevelID` int(3) UNSIGNED ZEROFILL DEFAULT NULL,
  `triggers` varchar(255) NOT NULL,
  `reaction` varchar(255) NOT NULL,
  `response` varchar(255) NOT NULL,
  `medication` varchar(255) NOT NULL,
  `lastEpisode` date DEFAULT NULL,
  `lastEpisodeTreatment` varchar(255) NOT NULL,
  `comment` text NOT NULL,
  `attachment` varchar(255) DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonPersonMedicalCondition`
  ADD PRIMARY KEY (`gibbonPersonMedicalConditionID`),
  ADD KEY `gibbonPersonMedicalID` (`gibbonPersonMedicalID`);

-- [TABLE] gibbonPersonMedicalConditionUpdate | [TRIAGE] CORE | [CATEGORY] SEN/Individual Needs, Medical & Pastoral
CREATE TABLE `gibbonPersonMedicalConditionUpdate` (
  `gibbonPersonMedicalConditionUpdateID` int(14) UNSIGNED ZEROFILL NOT NULL,
  `gibbonPersonMedicalUpdateID` int(12) UNSIGNED ZEROFILL DEFAULT NULL,
  `gibbonPersonMedicalConditionID` int(12) UNSIGNED ZEROFILL DEFAULT NULL,
  `gibbonPersonMedicalID` int(10) UNSIGNED ZEROFILL DEFAULT NULL,
  `name` varchar(80) NOT NULL,
  `gibbonAlertLevelID` int(3) UNSIGNED ZEROFILL DEFAULT NULL,
  `triggers` varchar(255) NOT NULL,
  `reaction` varchar(255) NOT NULL,
  `response` varchar(255) NOT NULL,
  `medication` varchar(255) NOT NULL,
  `lastEpisode` date DEFAULT NULL,
  `lastEpisodeTreatment` varchar(255) NOT NULL,
  `comment` text NOT NULL,
  `attachment` varchar(255) DEFAULT NULL,
  `gibbonPersonIDUpdater` int(10) UNSIGNED ZEROFILL NOT NULL,
  `timestamp` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonPersonMedicalConditionUpdate`
  ADD PRIMARY KEY (`gibbonPersonMedicalConditionUpdateID`);

-- [TABLE] gibbonPersonMedicalUpdate | [TRIAGE] CORE | [CATEGORY] SEN/Individual Needs, Medical & Pastoral
CREATE TABLE `gibbonPersonMedicalUpdate` (
  `gibbonPersonMedicalUpdateID` int(12) UNSIGNED ZEROFILL NOT NULL,
  `gibbonSchoolYearID` int(3) UNSIGNED ZEROFILL DEFAULT NULL,
  `status` enum('Pending','Complete') NOT NULL DEFAULT 'Pending',
  `gibbonPersonMedicalID` int(10) UNSIGNED ZEROFILL DEFAULT NULL,
  `gibbonPersonID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `longTermMedication` enum('','Y','N') NOT NULL,
  `longTermMedicationDetails` text NOT NULL,
  `comment` text NOT NULL,
  `gibbonPersonIDUpdater` int(10) UNSIGNED ZEROFILL NOT NULL,
  `timestamp` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `fields` text
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonPersonMedicalUpdate`
  ADD PRIMARY KEY (`gibbonPersonMedicalUpdateID`),
  ADD KEY `gibbonMedicalIndex` (`gibbonPersonID`,`gibbonPersonMedicalID`,`gibbonSchoolYearID`);

-- [TABLE] gibbonStudentNote | [TRIAGE] CORE | [CATEGORY] SEN/Individual Needs, Medical & Pastoral
CREATE TABLE `gibbonStudentNote` (
  `gibbonStudentNoteID` int(12) UNSIGNED ZEROFILL NOT NULL,
  `gibbonPersonID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `gibbonStudentNoteCategoryID` int(5) UNSIGNED ZEROFILL DEFAULT NULL,
  `title` varchar(50) NOT NULL,
  `note` text NOT NULL,
  `gibbonPersonIDCreator` int(10) UNSIGNED ZEROFILL NOT NULL,
  `timestamp` timestamp NULL DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonStudentNote`
  ADD PRIMARY KEY (`gibbonStudentNoteID`);

-- [TABLE] gibbonStudentNoteCategory | [TRIAGE] CORE | [CATEGORY] SEN/Individual Needs, Medical & Pastoral
CREATE TABLE `gibbonStudentNoteCategory` (
  `gibbonStudentNoteCategoryID` int(5) UNSIGNED ZEROFILL NOT NULL,
  `name` varchar(30) NOT NULL,
  `template` text NOT NULL,
  `active` enum('Y','N') NOT NULL DEFAULT 'Y'
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonStudentNoteCategory`
  ADD PRIMARY KEY (`gibbonStudentNoteCategoryID`);

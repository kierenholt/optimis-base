-- ============================================================================
-- Gibbon Blueprint Domain File: 01_identity_and_people.sql
-- Tables in this file: 22
-- Note: Stripped of INSERT seed data for fast AI agent reading.
-- ============================================================================

-- [TABLE] gibbonPerson | [TRIAGE] CORE | [CATEGORY] Identity, Pupils, Staff & HR
CREATE TABLE `gibbonPerson` (
  `gibbonPersonID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `title` varchar(5) NOT NULL,
  `surname` varchar(60) NOT NULL DEFAULT '',
  `firstName` varchar(60) NOT NULL DEFAULT '',
  `preferredName` varchar(60) NOT NULL DEFAULT '',
  `officialName` varchar(150) NOT NULL,
  `nameInCharacters` varchar(60) NOT NULL,
  `gender` enum('M','F','Other','Unspecified') NOT NULL DEFAULT 'Unspecified',
  `username` varchar(20) NOT NULL,
  `passwordStrong` varchar(255) NOT NULL,
  `passwordStrongSalt` varchar(255) NOT NULL,
  `passwordForceReset` enum('N','Y') NOT NULL DEFAULT 'N' COMMENT 'Force user to reset password on next login.',
  `status` enum('Full','Expected','Left','Pending Approval') NOT NULL DEFAULT 'Full',
  `canLogin` enum('Y','N') NOT NULL DEFAULT 'Y',
  `gibbonRoleIDPrimary` int(3) UNSIGNED ZEROFILL NOT NULL,
  `gibbonRoleIDAll` varchar(255) NOT NULL,
  `dob` date DEFAULT NULL,
  `email` varchar(75) DEFAULT NULL,
  `emailAlternate` varchar(75) DEFAULT NULL,
  `image_240` varchar(255) DEFAULT NULL,
  `lastIPAddress` varchar(15) NOT NULL DEFAULT '',
  `lastTimestamp` timestamp NULL DEFAULT NULL,
  `lastFailIPAddress` varchar(15) DEFAULT NULL,
  `lastFailTimestamp` timestamp NULL DEFAULT NULL,
  `failCount` int(1) DEFAULT '0',
  `address1` mediumtext NOT NULL,
  `address1District` varchar(255) NOT NULL,
  `address1Country` varchar(255) NOT NULL,
  `address2` mediumtext NOT NULL,
  `address2District` varchar(255) NOT NULL,
  `address2Country` varchar(255) NOT NULL,
  `phone1Type` enum('','Mobile','Home','Work','Fax','Pager','Other') NOT NULL DEFAULT '',
  `phone1CountryCode` varchar(7) NOT NULL,
  `phone1` varchar(20) NOT NULL,
  `phone3Type` enum('','Mobile','Home','Work','Fax','Pager','Other') NOT NULL DEFAULT '',
  `phone3CountryCode` varchar(7) NOT NULL,
  `phone3` varchar(20) NOT NULL,
  `phone2Type` enum('','Mobile','Home','Work','Fax','Pager','Other') NOT NULL DEFAULT '',
  `phone2CountryCode` varchar(7) NOT NULL,
  `phone2` varchar(20) NOT NULL,
  `phone4Type` enum('','Mobile','Home','Work','Fax','Pager','Other') NOT NULL DEFAULT '',
  `phone4CountryCode` varchar(7) NOT NULL,
  `phone4` varchar(20) NOT NULL,
  `website` varchar(255) NOT NULL,
  `languageFirst` varchar(30) NOT NULL,
  `languageSecond` varchar(30) NOT NULL,
  `languageThird` varchar(30) NOT NULL,
  `countryOfBirth` varchar(30) NOT NULL,
  `birthCertificateScan` varchar(255) NOT NULL,
  `ethnicity` varchar(255) NOT NULL,
  `religion` varchar(30) NOT NULL,
  `profession` varchar(90) NOT NULL,
  `employer` varchar(90) NOT NULL,
  `jobTitle` varchar(90) NOT NULL,
  `emergency1Name` varchar(90) NOT NULL,
  `emergency1Number1` varchar(30) NOT NULL,
  `emergency1Number2` varchar(30) NOT NULL,
  `emergency1Relationship` varchar(30) NOT NULL,
  `emergency2Name` varchar(90) NOT NULL,
  `emergency2Number1` varchar(30) NOT NULL,
  `emergency2Number2` varchar(30) NOT NULL,
  `emergency2Relationship` varchar(30) NOT NULL,
  `gibbonHouseID` int(3) UNSIGNED ZEROFILL DEFAULT NULL,
  `studentID` varchar(15) NOT NULL,
  `dateStart` date DEFAULT NULL,
  `dateEnd` date DEFAULT NULL,
  `gibbonSchoolYearIDClassOf` int(3) UNSIGNED ZEROFILL DEFAULT NULL,
  `lastSchool` varchar(100) NOT NULL,
  `nextSchool` varchar(100) NOT NULL,
  `departureReason` varchar(100) NOT NULL,
  `transport` varchar(255) NOT NULL,
  `transportNotes` text NOT NULL,
  `calendarFeedPersonal` text NOT NULL,
  `viewCalendarSchool` enum('Y','N') NOT NULL DEFAULT 'Y',
  `viewCalendarPersonal` enum('Y','N') NOT NULL DEFAULT 'Y',
  `viewCalendarSpaceBooking` enum('Y','N') NOT NULL DEFAULT 'N',
  `gibbonApplicationFormID` int(12) UNSIGNED ZEROFILL DEFAULT NULL,
  `lockerNumber` varchar(20) NOT NULL,
  `vehicleRegistration` varchar(20) NOT NULL,
  `personalBackground` varchar(255) NOT NULL,
  `messengerLastRead` datetime DEFAULT NULL,
  `privacy` text,
  `dayType` varchar(255) DEFAULT NULL COMMENT 'Student day type, as specified in the application form.',
  `gibbonThemeIDPersonal` int(4) UNSIGNED ZEROFILL DEFAULT NULL,
  `gibboni18nIDPersonal` int(4) UNSIGNED ZEROFILL DEFAULT NULL,
  `studentAgreements` text,
  `googleAPIRefreshToken` text NOT NULL,
  `microsoftAPIRefreshToken` text NOT NULL,
  `genericAPIRefreshToken` text NOT NULL,
  `receiveNotificationEmails` enum('Y','N') NOT NULL DEFAULT 'Y',
  `mfaSecret` varchar(16) DEFAULT NULL,
  `mfaToken` text,
  `cookieConsent` enum('Y','N') DEFAULT NULL,
  `fields` text NOT NULL COMMENT 'JSON object of custom field values'
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonPerson`
  ADD PRIMARY KEY (`gibbonPersonID`),
  ADD UNIQUE KEY `username` (`username`),
  ADD KEY `username_2` (`username`,`email`);

-- [TABLE] gibbonPersonReset | [TRIAGE] DROP | [CATEGORY] Identity, Pupils, Staff & HR
CREATE TABLE `gibbonPersonReset` (
  `gibbonPersonResetID` int(12) UNSIGNED ZEROFILL NOT NULL,
  `gibbonPersonID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `key` varchar(40) COLLATE utf8_unicode_ci NOT NULL,
  `timestamp` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_unicode_ci;
ALTER TABLE `gibbonPersonReset`
  ADD PRIMARY KEY (`gibbonPersonResetID`);

-- [TABLE] gibbonPersonStatusLog | [TRIAGE] CORE | [CATEGORY] Identity, Pupils, Staff & HR
CREATE TABLE `gibbonPersonStatusLog` (
  `gibbonPersonStatusLogID` int(12) UNSIGNED ZEROFILL NOT NULL,
  `gibbonPersonID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `statusOld` enum('Full','Expected','Left','Pending Approval') NOT NULL DEFAULT 'Full',
  `statusNew` enum('Full','Expected','Left','Pending Approval') NOT NULL DEFAULT 'Full',
  `reason` text NOT NULL,
  `timestamp` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `gibbonPersonIDModified` int(10) UNSIGNED ZEROFILL DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonPersonStatusLog`
  ADD PRIMARY KEY (`gibbonPersonStatusLogID`),
  ADD KEY `gibbonPersonID` (`gibbonPersonID`);

-- [TABLE] gibbonPersonUpdate | [TRIAGE] CORE | [CATEGORY] Identity, Pupils, Staff & HR
CREATE TABLE `gibbonPersonUpdate` (
  `gibbonPersonUpdateID` int(12) UNSIGNED ZEROFILL NOT NULL,
  `gibbonSchoolYearID` int(3) UNSIGNED ZEROFILL DEFAULT NULL,
  `status` enum('Pending','Complete') NOT NULL DEFAULT 'Pending',
  `gibbonPersonID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `title` varchar(5) NOT NULL,
  `surname` varchar(60) NOT NULL DEFAULT '',
  `firstName` varchar(60) NOT NULL DEFAULT '',
  `preferredName` varchar(60) NOT NULL DEFAULT '',
  `officialName` varchar(150) NOT NULL,
  `nameInCharacters` varchar(60) NOT NULL,
  `dob` date DEFAULT NULL,
  `email` varchar(75) DEFAULT NULL,
  `emailAlternate` varchar(75) DEFAULT NULL,
  `address1` mediumtext NOT NULL,
  `address1District` varchar(255) NOT NULL,
  `address1Country` varchar(255) NOT NULL,
  `address2` mediumtext NOT NULL,
  `address2District` varchar(255) NOT NULL,
  `address2Country` varchar(255) NOT NULL,
  `phone1Type` enum('','Mobile','Home','Work','Fax','Pager','Other') NOT NULL DEFAULT '',
  `phone1CountryCode` varchar(7) NOT NULL,
  `phone1` varchar(20) NOT NULL,
  `phone3Type` enum('','Mobile','Home','Work','Fax','Pager','Other') NOT NULL DEFAULT '',
  `phone3CountryCode` varchar(7) NOT NULL,
  `phone3` varchar(20) NOT NULL,
  `phone2Type` enum('','Mobile','Home','Work','Fax','Pager','Other') NOT NULL DEFAULT '',
  `phone2CountryCode` varchar(7) NOT NULL,
  `phone2` varchar(20) NOT NULL,
  `phone4Type` enum('','Mobile','Home','Work','Fax','Pager','Other') NOT NULL DEFAULT '',
  `phone4CountryCode` varchar(7) NOT NULL,
  `phone4` varchar(20) NOT NULL,
  `languageFirst` varchar(30) NOT NULL,
  `languageSecond` varchar(30) NOT NULL,
  `languageThird` varchar(30) NOT NULL,
  `countryOfBirth` varchar(30) NOT NULL,
  `ethnicity` varchar(255) NOT NULL,
  `religion` varchar(30) NOT NULL,
  `profession` varchar(90) DEFAULT NULL,
  `employer` varchar(90) DEFAULT NULL,
  `jobTitle` varchar(90) DEFAULT NULL,
  `emergency1Name` varchar(90) DEFAULT NULL,
  `emergency1Number1` varchar(30) DEFAULT NULL,
  `emergency1Number2` varchar(30) DEFAULT NULL,
  `emergency1Relationship` varchar(30) DEFAULT NULL,
  `emergency2Name` varchar(90) DEFAULT NULL,
  `emergency2Number1` varchar(30) DEFAULT NULL,
  `emergency2Number2` varchar(30) DEFAULT NULL,
  `emergency2Relationship` varchar(30) DEFAULT NULL,
  `vehicleRegistration` varchar(20) NOT NULL,
  `gibbonPersonIDUpdater` int(10) UNSIGNED ZEROFILL NOT NULL,
  `timestamp` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `privacy` text,
  `fields` text NOT NULL COMMENT 'Serialised array of custom field values'
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonPersonUpdate`
  ADD PRIMARY KEY (`gibbonPersonUpdateID`),
  ADD KEY `gibbonPersonIndex` (`gibbonPersonID`,`gibbonSchoolYearID`);

-- [TABLE] gibbonPersonalDocument | [TRIAGE] CORE | [CATEGORY] Identity, Pupils, Staff & HR
CREATE TABLE `gibbonPersonalDocument` (
  `gibbonPersonalDocumentID` int(12) UNSIGNED ZEROFILL NOT NULL,
  `gibbonPersonalDocumentTypeID` int(3) UNSIGNED ZEROFILL NOT NULL,
  `foreignTable` varchar(60) NOT NULL,
  `foreignTableID` int(12) UNSIGNED ZEROFILL NOT NULL,
  `document` enum('Passport','ID Card','Visa','Document') NOT NULL DEFAULT 'Document',
  `documentNumber` varchar(120) DEFAULT NULL,
  `documentName` varchar(120) DEFAULT NULL,
  `documentType` varchar(60) DEFAULT NULL,
  `dateIssue` date DEFAULT NULL,
  `dateExpiry` date DEFAULT NULL,
  `filePath` varchar(255) DEFAULT NULL,
  `country` varchar(60) DEFAULT NULL,
  `gibbonPersonIDUpdater` int(10) UNSIGNED ZEROFILL DEFAULT NULL,
  `timestamp` timestamp NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonPersonalDocument`
  ADD PRIMARY KEY (`gibbonPersonalDocumentID`),
  ADD UNIQUE KEY `foreignTableID` (`gibbonPersonalDocumentTypeID`,`foreignTable`,`foreignTableID`);

-- [TABLE] gibbonPersonalDocumentType | [TRIAGE] CORE | [CATEGORY] Identity, Pupils, Staff & HR
CREATE TABLE `gibbonPersonalDocumentType` (
  `gibbonPersonalDocumentTypeID` int(3) UNSIGNED ZEROFILL NOT NULL,
  `name` varchar(60) NOT NULL,
  `description` varchar(255) NOT NULL,
  `active` enum('Y','N') NOT NULL DEFAULT 'Y',
  `type` enum('Core','Additional') NOT NULL DEFAULT 'Additional',
  `document` enum('Passport','ID Card','Visa','Document') NOT NULL DEFAULT 'Document',
  `fields` text,
  `required` enum('Y','N') NOT NULL DEFAULT 'Y',
  `sequenceNumber` int(3) NOT NULL DEFAULT '0',
  `activePersonStudent` tinyint(1) NOT NULL DEFAULT '0',
  `activePersonStaff` tinyint(1) NOT NULL DEFAULT '0',
  `activePersonParent` tinyint(1) NOT NULL DEFAULT '0',
  `activePersonOther` tinyint(1) NOT NULL DEFAULT '0',
  `activeApplicationForm` tinyint(1) NOT NULL DEFAULT '0',
  `activeDataUpdater` tinyint(1) NOT NULL DEFAULT '0'
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonPersonalDocumentType`
  ADD PRIMARY KEY (`gibbonPersonalDocumentTypeID`);

-- [TABLE] gibbonRole | [TRIAGE] CORE | [CATEGORY] Identity, Pupils, Staff & HR
CREATE TABLE `gibbonRole` (
  `gibbonRoleID` int(3) UNSIGNED ZEROFILL NOT NULL,
  `category` enum('Staff','Student','Parent','Other') NOT NULL DEFAULT 'Staff',
  `name` varchar(20) NOT NULL,
  `nameShort` varchar(4) NOT NULL,
  `description` varchar(60) NOT NULL,
  `type` enum('Core','Additional') NOT NULL DEFAULT 'Core',
  `canLoginRole` enum('Y','N') NOT NULL DEFAULT 'Y',
  `futureYearsLogin` enum('Y','N') NOT NULL DEFAULT 'Y',
  `pastYearsLogin` enum('Y','N') NOT NULL DEFAULT 'Y',
  `restriction` enum('None','Same Role','Admin Only') NOT NULL DEFAULT 'None'
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonRole`
  ADD PRIMARY KEY (`gibbonRoleID`),
  ADD UNIQUE KEY `name` (`name`),
  ADD UNIQUE KEY `nameShort` (`nameShort`);

-- [TABLE] gibbonStaff | [TRIAGE] CORE | [CATEGORY] Identity, Pupils, Staff & HR
CREATE TABLE `gibbonStaff` (
  `gibbonStaffID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `gibbonPersonID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `type` varchar(20) NOT NULL,
  `initials` varchar(4) DEFAULT NULL,
  `jobTitle` varchar(100) NOT NULL,
  `firstAidQualified` enum('','N','Y') NOT NULL DEFAULT '',
  `firstAidQualification` varchar(100) DEFAULT NULL,
  `firstAidExpiry` date DEFAULT NULL,
  `countryOfOrigin` varchar(80) NOT NULL,
  `qualifications` varchar(255) NOT NULL,
  `biography` text NOT NULL,
  `biographicalGrouping` varchar(100) NOT NULL COMMENT 'Used for group staff when creating a staff directory.',
  `biographicalGroupingPriority` int(3) NOT NULL,
  `coverageExclude` enum('N','Y') NOT NULL DEFAULT 'N',
  `coveragePriority` int(1) DEFAULT '0',
  `fields` text
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonStaff`
  ADD PRIMARY KEY (`gibbonStaffID`),
  ADD UNIQUE KEY `gibbonPersonID` (`gibbonPersonID`),
  ADD UNIQUE KEY `initials` (`initials`);

-- [TABLE] gibbonStaffAbsence | [TRIAGE] CORE | [CATEGORY] Identity, Pupils, Staff & HR
CREATE TABLE `gibbonStaffAbsence` (
  `gibbonStaffAbsenceID` int(14) UNSIGNED ZEROFILL NOT NULL,
  `gibbonStaffAbsenceTypeID` int(6) UNSIGNED ZEROFILL NOT NULL,
  `gibbonSchoolYearID` int(3) UNSIGNED ZEROFILL NOT NULL,
  `gibbonPersonID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `reason` varchar(60) DEFAULT NULL,
  `comment` text,
  `commentConfidential` text,
  `status` enum('Pending Approval','Approved','Declined','Cancelled') DEFAULT 'Approved',
  `coverageRequired` enum('N','Y') DEFAULT 'N',
  `gibbonPersonIDApproval` int(10) UNSIGNED ZEROFILL DEFAULT NULL,
  `timestampApproval` timestamp NULL DEFAULT NULL,
  `notesApproval` text,
  `gibbonPersonIDCreator` int(10) UNSIGNED ZEROFILL NOT NULL,
  `timestampCreator` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `notificationSent` enum('N','Y') DEFAULT 'N',
  `notificationList` text,
  `gibbonGroupID` int(8) UNSIGNED ZEROFILL DEFAULT NULL,
  `googleCalendarEventID` text
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonStaffAbsence`
  ADD PRIMARY KEY (`gibbonStaffAbsenceID`);

-- [TABLE] gibbonStaffAbsenceDate | [TRIAGE] CORE | [CATEGORY] Identity, Pupils, Staff & HR
CREATE TABLE `gibbonStaffAbsenceDate` (
  `gibbonStaffAbsenceDateID` int(14) UNSIGNED ZEROFILL NOT NULL,
  `gibbonStaffAbsenceID` int(14) UNSIGNED ZEROFILL DEFAULT NULL,
  `date` date DEFAULT NULL,
  `allDay` enum('N','Y') DEFAULT 'Y',
  `timeStart` time DEFAULT NULL,
  `timeEnd` time DEFAULT NULL,
  `value` decimal(3,2) NOT NULL DEFAULT '1.00'
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonStaffAbsenceDate`
  ADD PRIMARY KEY (`gibbonStaffAbsenceDateID`);

-- [TABLE] gibbonStaffAbsenceType | [TRIAGE] CORE | [CATEGORY] Identity, Pupils, Staff & HR
CREATE TABLE `gibbonStaffAbsenceType` (
  `gibbonStaffAbsenceTypeID` int(6) UNSIGNED ZEROFILL NOT NULL,
  `name` varchar(60) DEFAULT NULL,
  `nameShort` varchar(10) DEFAULT NULL,
  `active` enum('N','Y') DEFAULT 'Y',
  `requiresApproval` enum('N','Y') DEFAULT 'N',
  `reasons` text,
  `sequenceNumber` int(3) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonStaffAbsenceType`
  ADD PRIMARY KEY (`gibbonStaffAbsenceTypeID`);

-- [TABLE] gibbonStaffApplicationForm | [TRIAGE] PLUGIN | [CATEGORY] Identity, Pupils, Staff & HR
CREATE TABLE `gibbonStaffApplicationForm` (
  `gibbonStaffApplicationFormID` int(12) UNSIGNED ZEROFILL NOT NULL,
  `gibbonStaffJobOpeningID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `gibbonPersonID` int(10) UNSIGNED ZEROFILL DEFAULT NULL,
  `surname` varchar(60) DEFAULT NULL,
  `firstName` varchar(60) DEFAULT NULL,
  `preferredName` varchar(60) DEFAULT NULL,
  `officialName` varchar(150) DEFAULT NULL,
  `nameInCharacters` varchar(60) DEFAULT NULL,
  `gender` enum('M','F','Other','Unspecified') NOT NULL DEFAULT 'Unspecified',
  `status` enum('Pending','Accepted','Rejected','Withdrawn') NOT NULL DEFAULT 'Pending',
  `dob` date DEFAULT NULL,
  `email` varchar(75) DEFAULT NULL,
  `homeAddress` mediumtext,
  `homeAddressDistrict` varchar(255) DEFAULT NULL,
  `homeAddressCountry` varchar(255) DEFAULT NULL,
  `phone1Type` enum('','Mobile','Home','Work','Fax','Pager','Other') DEFAULT NULL,
  `phone1CountryCode` varchar(7) DEFAULT NULL,
  `phone1` varchar(20) DEFAULT NULL,
  `countryOfBirth` varchar(30) DEFAULT NULL,
  `languageFirst` varchar(30) DEFAULT NULL,
  `languageSecond` varchar(30) DEFAULT NULL,
  `languageThird` varchar(30) DEFAULT NULL,
  `agreement` enum('N','Y') DEFAULT NULL,
  `timestamp` timestamp NULL DEFAULT NULL,
  `priority` int(1) NOT NULL DEFAULT '0',
  `milestones` text NOT NULL,
  `notes` text NOT NULL,
  `dateStart` date DEFAULT NULL,
  `questions` text NOT NULL,
  `fields` text NOT NULL COMMENT 'Serialised array of custom field values',
  `staffFields` text,
  `referenceEmail1` varchar(100) NOT NULL,
  `referenceEmail2` varchar(100) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonStaffApplicationForm`
  ADD PRIMARY KEY (`gibbonStaffApplicationFormID`);

-- [TABLE] gibbonStaffApplicationFormFile | [TRIAGE] PLUGIN | [CATEGORY] Identity, Pupils, Staff & HR
CREATE TABLE `gibbonStaffApplicationFormFile` (
  `gibbonStaffApplicationFormFileID` int(14) UNSIGNED ZEROFILL NOT NULL,
  `gibbonStaffApplicationFormID` int(12) UNSIGNED ZEROFILL NOT NULL,
  `name` varchar(255) NOT NULL,
  `path` varchar(255) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonStaffApplicationFormFile`
  ADD PRIMARY KEY (`gibbonStaffApplicationFormFileID`);

-- [TABLE] gibbonStaffContract | [TRIAGE] CORE | [CATEGORY] Identity, Pupils, Staff & HR
CREATE TABLE `gibbonStaffContract` (
  `gibbonStaffContractID` int(12) UNSIGNED ZEROFILL NOT NULL,
  `gibbonStaffID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `title` varchar(100) NOT NULL,
  `status` enum('','Pending','Active','Expired') NOT NULL DEFAULT '',
  `dateStart` date NOT NULL,
  `dateEnd` date DEFAULT NULL,
  `salaryScale` varchar(255) DEFAULT NULL,
  `salaryAmount` decimal(12,2) DEFAULT NULL,
  `salaryPeriod` enum('','Week','Month','Year','Contract') DEFAULT NULL,
  `responsibility` varchar(255) DEFAULT NULL,
  `responsibilityAmount` decimal(12,2) DEFAULT NULL,
  `responsibilityPeriod` enum('','Week','Month','Year','Contract') DEFAULT NULL,
  `housingAmount` decimal(12,2) DEFAULT NULL,
  `housingPeriod` enum('','Week','Month','Year','Contract') DEFAULT NULL,
  `travelAmount` decimal(12,2) DEFAULT NULL,
  `travelPeriod` enum('','Week','Month','Year','Contract') DEFAULT NULL,
  `retirementAmount` decimal(12,2) DEFAULT NULL,
  `retirementPeriod` enum('','Week','Month','Year','Contract') DEFAULT NULL,
  `bonusAmount` decimal(12,2) DEFAULT NULL,
  `bonusPeriod` enum('','Week','Month','Year','Contract') DEFAULT NULL,
  `education` text NOT NULL,
  `notes` text NOT NULL,
  `contractUpload` varchar(255) DEFAULT NULL,
  `gibbonPersonIDCreator` int(10) UNSIGNED ZEROFILL NOT NULL,
  `timestampCreator` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonStaffContract`
  ADD PRIMARY KEY (`gibbonStaffContractID`);

-- [TABLE] gibbonStaffCoverage | [TRIAGE] CORE | [CATEGORY] Identity, Pupils, Staff & HR
CREATE TABLE `gibbonStaffCoverage` (
  `gibbonStaffCoverageID` int(14) UNSIGNED ZEROFILL NOT NULL,
  `gibbonStaffAbsenceID` int(14) UNSIGNED ZEROFILL DEFAULT NULL,
  `gibbonSchoolYearID` int(3) UNSIGNED ZEROFILL NOT NULL,
  `gibbonPersonID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `status` enum('Requested','Accepted','Declined','Cancelled','Pending','Not Required') DEFAULT 'Requested',
  `requestType` enum('Individual','Broadcast','Assigned') DEFAULT 'Broadcast',
  `substituteTypes` varchar(255) DEFAULT NULL,
  `gibbonPersonIDStatus` int(10) UNSIGNED ZEROFILL NOT NULL,
  `timestampStatus` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `notesStatus` text,
  `gibbonPersonIDCoverage` int(10) UNSIGNED ZEROFILL DEFAULT NULL,
  `timestampCoverage` timestamp NULL DEFAULT NULL,
  `notesCoverage` text,
  `attachmentType` enum('File','HTML','Link') DEFAULT NULL,
  `attachmentContent` text,
  `notificationSent` enum('N','Y') DEFAULT 'N',
  `notificationList` text
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonStaffCoverage`
  ADD PRIMARY KEY (`gibbonStaffCoverageID`);

-- [TABLE] gibbonStaffCoverageDate | [TRIAGE] CORE | [CATEGORY] Identity, Pupils, Staff & HR
CREATE TABLE `gibbonStaffCoverageDate` (
  `gibbonStaffCoverageDateID` int(14) UNSIGNED ZEROFILL NOT NULL,
  `gibbonStaffCoverageID` int(14) UNSIGNED ZEROFILL DEFAULT NULL,
  `gibbonStaffAbsenceDateID` int(14) UNSIGNED ZEROFILL DEFAULT NULL,
  `foreignTable` varchar(60) DEFAULT NULL,
  `foreignTableID` int(12) UNSIGNED ZEROFILL DEFAULT NULL,
  `date` date DEFAULT NULL,
  `allDay` enum('N','Y') DEFAULT 'Y',
  `timeStart` time DEFAULT NULL,
  `timeEnd` time DEFAULT NULL,
  `value` decimal(3,2) NOT NULL DEFAULT '1.00',
  `gibbonPersonIDUnavailable` int(10) UNSIGNED ZEROFILL DEFAULT NULL,
  `reason` varchar(255) DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonStaffCoverageDate`
  ADD PRIMARY KEY (`gibbonStaffCoverageDateID`),
  ADD KEY `foreignTable` (`foreignTable`,`foreignTableID`);

-- [TABLE] gibbonStaffDuty | [TRIAGE] CORE | [CATEGORY] Identity, Pupils, Staff & HR
CREATE TABLE `gibbonStaffDuty` (
  `gibbonStaffDutyID` int(6) UNSIGNED ZEROFILL NOT NULL,
  `gibbonDaysOfWeekIDList` varchar(30) DEFAULT NULL,
  `name` varchar(90) NOT NULL,
  `nameShort` varchar(20) NOT NULL,
  `timeStart` time NOT NULL,
  `timeEnd` time NOT NULL,
  `sequenceNumber` int(6) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonStaffDuty`
  ADD PRIMARY KEY (`gibbonStaffDutyID`);

-- [TABLE] gibbonStaffDutyPerson | [TRIAGE] CORE | [CATEGORY] Identity, Pupils, Staff & HR
CREATE TABLE `gibbonStaffDutyPerson` (
  `gibbonStaffDutyPersonID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `gibbonStaffDutyID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `gibbonDaysOfWeekID` int(2) UNSIGNED ZEROFILL NOT NULL,
  `gibbonPersonID` int(10) UNSIGNED ZEROFILL NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonStaffDutyPerson`
  ADD PRIMARY KEY (`gibbonStaffDutyPersonID`),
  ADD UNIQUE KEY `gibbonStaffDutyID` (`gibbonStaffDutyID`,`gibbonDaysOfWeekID`,`gibbonPersonID`);

-- [TABLE] gibbonStaffJobOpening | [TRIAGE] PLUGIN | [CATEGORY] Identity, Pupils, Staff & HR
CREATE TABLE `gibbonStaffJobOpening` (
  `gibbonStaffJobOpeningID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `type` varchar(20) NOT NULL,
  `jobTitle` varchar(100) NOT NULL,
  `dateOpen` date NOT NULL,
  `active` enum('Y','N') NOT NULL DEFAULT 'Y',
  `description` text NOT NULL,
  `gibbonPersonIDCreator` int(10) UNSIGNED ZEROFILL NOT NULL,
  `timestampCreator` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonStaffJobOpening`
  ADD PRIMARY KEY (`gibbonStaffJobOpeningID`);

-- [TABLE] gibbonStaffUpdate | [TRIAGE] CORE | [CATEGORY] Identity, Pupils, Staff & HR
CREATE TABLE `gibbonStaffUpdate` (
  `gibbonStaffUpdateID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `gibbonSchoolYearID` int(3) UNSIGNED ZEROFILL DEFAULT NULL,
  `gibbonStaffID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `status` enum('Pending','Complete') NOT NULL DEFAULT 'Pending',
  `type` varchar(20) NOT NULL,
  `initials` varchar(4) DEFAULT NULL,
  `jobTitle` varchar(100) NOT NULL,
  `firstAidQualified` enum('','N','Y') NOT NULL DEFAULT '',
  `firstAidQualification` varchar(100) DEFAULT NULL,
  `firstAidExpiry` date DEFAULT NULL,
  `countryOfOrigin` varchar(80) NOT NULL,
  `qualifications` varchar(255) NOT NULL,
  `biography` text NOT NULL,
  `fields` text NOT NULL,
  `gibbonPersonIDUpdater` int(10) UNSIGNED ZEROFILL NOT NULL,
  `timestamp` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonStaffUpdate`
  ADD PRIMARY KEY (`gibbonStaffUpdateID`);

-- [TABLE] gibbonStudentEnrolment | [TRIAGE] CORE | [CATEGORY] Identity, Pupils, Staff & HR
CREATE TABLE `gibbonStudentEnrolment` (
  `gibbonStudentEnrolmentID` int(8) UNSIGNED ZEROFILL NOT NULL,
  `gibbonPersonID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `gibbonSchoolYearID` int(3) UNSIGNED ZEROFILL NOT NULL,
  `gibbonYearGroupID` int(3) UNSIGNED ZEROFILL NOT NULL,
  `gibbonFormGroupID` int(5) UNSIGNED ZEROFILL NOT NULL,
  `rollOrder` int(2) DEFAULT NULL,
  `fields` text
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonStudentEnrolment`
  ADD PRIMARY KEY (`gibbonStudentEnrolmentID`),
  ADD KEY `gibbonSchoolYearID` (`gibbonSchoolYearID`),
  ADD KEY `gibbonYearGroupID` (`gibbonYearGroupID`),
  ADD KEY `gibbonPersonIndex` (`gibbonPersonID`,`gibbonSchoolYearID`),
  ADD KEY `gibbonFormGroupID` (`gibbonFormGroupID`);

-- [TABLE] gibbonSubstitute | [TRIAGE] CORE | [CATEGORY] Identity, Pupils, Staff & HR
CREATE TABLE `gibbonSubstitute` (
  `gibbonSubstituteID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `gibbonPersonID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `active` enum('Y','N') DEFAULT 'Y',
  `type` varchar(60) DEFAULT NULL,
  `details` varchar(255) DEFAULT NULL,
  `priority` int(2) NOT NULL DEFAULT '0'
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonSubstitute`
  ADD PRIMARY KEY (`gibbonSubstituteID`),
  ADD UNIQUE KEY `gibbonPersonID` (`gibbonPersonID`);

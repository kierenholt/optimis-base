-- ============================================================================
-- Gibbon Blueprint Domain File: 02_family_and_guardians.sql
-- Tables in this file: 5
-- Note: Stripped of INSERT seed data for fast AI agent reading.
-- ============================================================================

-- [TABLE] gibbonFamily | [TRIAGE] CORE | [CATEGORY] Households, Guardians & PR relationships
CREATE TABLE `gibbonFamily` (
  `gibbonFamilyID` int(7) UNSIGNED ZEROFILL NOT NULL,
  `name` varchar(100) NOT NULL,
  `nameAddress` varchar(100) NOT NULL COMMENT 'The formal name to be used for addressing the family (e.g. Mr. & Mrs. Smith)',
  `homeAddress` mediumtext NOT NULL,
  `homeAddressDistrict` varchar(255) NOT NULL,
  `homeAddressCountry` varchar(255) NOT NULL,
  `status` enum('Married','Separated','Divorced','De Facto','Other','Single') NOT NULL,
  `languageHomePrimary` varchar(30) NOT NULL,
  `languageHomeSecondary` varchar(30) DEFAULT NULL,
  `familySync` varchar(50) DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonFamily`
  ADD PRIMARY KEY (`gibbonFamilyID`);

-- [TABLE] gibbonFamilyAdult | [TRIAGE] CORE | [CATEGORY] Households, Guardians & PR relationships
CREATE TABLE `gibbonFamilyAdult` (
  `gibbonFamilyAdultID` int(8) UNSIGNED ZEROFILL NOT NULL,
  `gibbonFamilyID` int(7) UNSIGNED ZEROFILL NOT NULL,
  `gibbonPersonID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `comment` text NOT NULL,
  `childDataAccess` enum('Y','N') NOT NULL,
  `contactPriority` int(2) NOT NULL DEFAULT '1',
  `contactCall` enum('Y','N') NOT NULL,
  `contactSMS` enum('Y','N') NOT NULL,
  `contactEmail` enum('Y','N') NOT NULL,
  `contactMail` enum('Y','N') NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonFamilyAdult`
  ADD PRIMARY KEY (`gibbonFamilyAdultID`),
  ADD KEY `gibbonFamilyID` (`gibbonFamilyID`,`contactPriority`),
  ADD KEY `gibbonPersonIndex` (`gibbonPersonID`);

-- [TABLE] gibbonFamilyChild | [TRIAGE] CORE | [CATEGORY] Households, Guardians & PR relationships
CREATE TABLE `gibbonFamilyChild` (
  `gibbonFamilyChildID` int(8) UNSIGNED ZEROFILL NOT NULL,
  `gibbonFamilyID` int(7) UNSIGNED ZEROFILL NOT NULL,
  `gibbonPersonID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `comment` text NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonFamilyChild`
  ADD PRIMARY KEY (`gibbonFamilyChildID`),
  ADD KEY `gibbonPersonIndex` (`gibbonPersonID`),
  ADD KEY `gibbonFamilyIndex` (`gibbonFamilyID`);

-- [TABLE] gibbonFamilyRelationship | [TRIAGE] CORE | [CATEGORY] Households, Guardians & PR relationships
CREATE TABLE `gibbonFamilyRelationship` (
  `gibbonFamilyRelationshipID` int(9) UNSIGNED ZEROFILL NOT NULL,
  `gibbonFamilyID` int(7) UNSIGNED ZEROFILL NOT NULL,
  `gibbonPersonID1` int(10) UNSIGNED ZEROFILL NOT NULL,
  `gibbonPersonID2` int(10) UNSIGNED ZEROFILL NOT NULL,
  `relationship` varchar(50) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COMMENT='Person 1 is [relationship] to person 2?';
ALTER TABLE `gibbonFamilyRelationship`
  ADD PRIMARY KEY (`gibbonFamilyRelationshipID`);

-- [TABLE] gibbonFamilyUpdate | [TRIAGE] CORE | [CATEGORY] Households, Guardians & PR relationships
CREATE TABLE `gibbonFamilyUpdate` (
  `gibbonFamilyUpdateID` int(9) UNSIGNED ZEROFILL NOT NULL,
  `gibbonSchoolYearID` int(3) UNSIGNED ZEROFILL DEFAULT NULL,
  `status` enum('Pending','Complete') NOT NULL DEFAULT 'Pending',
  `gibbonFamilyID` int(7) UNSIGNED ZEROFILL NOT NULL,
  `nameAddress` varchar(100) NOT NULL DEFAULT '',
  `homeAddress` mediumtext NOT NULL,
  `homeAddressDistrict` varchar(255) NOT NULL DEFAULT '',
  `homeAddressCountry` varchar(255) NOT NULL DEFAULT '',
  `languageHomePrimary` varchar(30) NOT NULL,
  `languageHomeSecondary` varchar(30) NOT NULL,
  `gibbonPersonIDUpdater` int(10) UNSIGNED ZEROFILL NOT NULL,
  `timestamp` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonFamilyUpdate`
  ADD PRIMARY KEY (`gibbonFamilyUpdateID`),
  ADD KEY `gibbonFamilyIndex` (`gibbonFamilyID`,`gibbonSchoolYearID`);

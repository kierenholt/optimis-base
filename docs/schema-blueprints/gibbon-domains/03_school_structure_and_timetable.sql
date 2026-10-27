-- ============================================================================
-- Gibbon Blueprint Domain File: 03_school_structure_and_timetable.sql
-- Tables in this file: 28
-- Note: Stripped of INSERT seed data for fast AI agent reading.
-- ============================================================================

-- [TABLE] gibbonCourse | [TRIAGE] CORE | [CATEGORY] Academic Cycle, Groups, Rooms & Timetable Spine
CREATE TABLE `gibbonCourse` (
  `gibbonCourseID` int(8) UNSIGNED ZEROFILL NOT NULL,
  `gibbonSchoolYearID` int(3) UNSIGNED ZEROFILL NOT NULL,
  `gibbonDepartmentID` int(4) UNSIGNED ZEROFILL DEFAULT NULL,
  `name` varchar(60) NOT NULL,
  `nameShort` varchar(12) NOT NULL,
  `description` text NOT NULL,
  `map` enum('Y','N') NOT NULL DEFAULT 'Y' COMMENT 'Should this course be included in curriculum maps and other summaries?',
  `gibbonYearGroupIDList` varchar(255) NOT NULL,
  `orderBy` int(3) NOT NULL,
  `fields` text
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonCourse`
  ADD PRIMARY KEY (`gibbonCourseID`),
  ADD UNIQUE KEY `nameYear` (`gibbonSchoolYearID`,`name`),
  ADD KEY `gibbonSchoolYearID` (`gibbonSchoolYearID`);

-- [TABLE] gibbonCourseClass | [TRIAGE] CORE | [CATEGORY] Academic Cycle, Groups, Rooms & Timetable Spine
CREATE TABLE `gibbonCourseClass` (
  `gibbonCourseClassID` int(8) UNSIGNED ZEROFILL NOT NULL,
  `gibbonCourseID` int(8) UNSIGNED ZEROFILL NOT NULL,
  `name` varchar(30) NOT NULL DEFAULT '',
  `nameShort` varchar(8) NOT NULL,
  `reportable` enum('Y','N') NOT NULL DEFAULT 'Y',
  `attendance` enum('Y','N') NOT NULL DEFAULT 'Y',
  `enrolmentMin` int(4) DEFAULT NULL,
  `enrolmentMax` int(4) DEFAULT NULL,
  `gibbonScaleIDTarget` int(5) UNSIGNED ZEROFILL DEFAULT NULL,
  `fields` text
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonCourseClass`
  ADD PRIMARY KEY (`gibbonCourseClassID`),
  ADD KEY `gibbonCourseID` (`gibbonCourseID`);

-- [TABLE] gibbonCourseClassMap | [TRIAGE] CORE | [CATEGORY] Academic Cycle, Groups, Rooms & Timetable Spine
CREATE TABLE `gibbonCourseClassMap` (
  `gibbonCourseClassMapID` int(8) UNSIGNED ZEROFILL NOT NULL,
  `gibbonCourseClassID` int(8) UNSIGNED ZEROFILL DEFAULT NULL,
  `gibbonFormGroupID` int(5) UNSIGNED ZEROFILL DEFAULT NULL,
  `gibbonYearGroupID` int(3) UNSIGNED ZEROFILL DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_unicode_ci;
ALTER TABLE `gibbonCourseClassMap`
  ADD PRIMARY KEY (`gibbonCourseClassMapID`),
  ADD UNIQUE KEY `gibbonCourseClassID` (`gibbonCourseClassID`);

-- [TABLE] gibbonCourseClassPerson | [TRIAGE] CORE | [CATEGORY] Academic Cycle, Groups, Rooms & Timetable Spine
CREATE TABLE `gibbonCourseClassPerson` (
  `gibbonCourseClassPersonID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `gibbonCourseClassID` int(8) UNSIGNED ZEROFILL NOT NULL,
  `gibbonPersonID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `role` enum('Student','Teacher','Assistant','Technician','Parent','Student - Left','Teacher - Left') NOT NULL,
  `dateEnrolled` date DEFAULT NULL,
  `dateUnenrolled` date DEFAULT NULL,
  `reportable` enum('Y','N') NOT NULL DEFAULT 'Y'
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonCourseClassPerson`
  ADD PRIMARY KEY (`gibbonCourseClassPersonID`),
  ADD KEY `gibbonCourseClassID` (`gibbonCourseClassID`),
  ADD KEY `gibbonPersonID` (`gibbonPersonID`,`role`);

-- [TABLE] gibbonDaysOfWeek | [TRIAGE] CORE | [CATEGORY] Academic Cycle, Groups, Rooms & Timetable Spine
CREATE TABLE `gibbonDaysOfWeek` (
  `gibbonDaysOfWeekID` int(2) UNSIGNED ZEROFILL NOT NULL,
  `name` varchar(10) NOT NULL,
  `nameShort` varchar(4) NOT NULL,
  `sequenceNumber` int(2) NOT NULL,
  `schoolDay` enum('Y','N') NOT NULL DEFAULT 'Y',
  `schoolOpen` time DEFAULT NULL,
  `schoolStart` time DEFAULT NULL,
  `schoolEnd` time DEFAULT NULL,
  `schoolClose` time DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonDaysOfWeek`
  ADD PRIMARY KEY (`gibbonDaysOfWeekID`),
  ADD UNIQUE KEY `name` (`name`,`nameShort`),
  ADD UNIQUE KEY `sequenceNumber` (`sequenceNumber`),
  ADD UNIQUE KEY `nameShort` (`nameShort`);

-- [TABLE] gibbonDepartment | [TRIAGE] CORE | [CATEGORY] Academic Cycle, Groups, Rooms & Timetable Spine
CREATE TABLE `gibbonDepartment` (
  `gibbonDepartmentID` int(4) UNSIGNED ZEROFILL NOT NULL,
  `type` enum('Learning Area','Administration') NOT NULL DEFAULT 'Learning Area',
  `name` varchar(40) NOT NULL,
  `nameShort` varchar(4) NOT NULL,
  `subjectListing` varchar(255) NOT NULL,
  `blurb` text NOT NULL,
  `logo` varchar(255) NOT NULL,
  `sequenceNumber` int(4) UNSIGNED DEFAULT NULL,
  `fields` text
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonDepartment`
  ADD PRIMARY KEY (`gibbonDepartmentID`);

-- [TABLE] gibbonDepartmentResource | [TRIAGE] PLUGIN | [CATEGORY] Academic Cycle, Groups, Rooms & Timetable Spine
CREATE TABLE `gibbonDepartmentResource` (
  `gibbonDepartmentResourceID` int(8) UNSIGNED ZEROFILL NOT NULL,
  `gibbonDepartmentID` int(4) UNSIGNED ZEROFILL NOT NULL,
  `type` enum('Link','File') NOT NULL,
  `name` varchar(100) NOT NULL,
  `url` varchar(255) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonDepartmentResource`
  ADD PRIMARY KEY (`gibbonDepartmentResourceID`);

-- [TABLE] gibbonDepartmentStaff | [TRIAGE] CORE | [CATEGORY] Academic Cycle, Groups, Rooms & Timetable Spine
CREATE TABLE `gibbonDepartmentStaff` (
  `gibbonDepartmentStaffID` int(6) UNSIGNED ZEROFILL NOT NULL,
  `gibbonDepartmentID` int(4) UNSIGNED ZEROFILL NOT NULL,
  `gibbonPersonID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `role` enum('Coordinator','Assistant Coordinator','Teacher (Curriculum)','Teacher','Director','Manager','Administrator','Other') NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonDepartmentStaff`
  ADD PRIMARY KEY (`gibbonDepartmentStaffID`);

-- [TABLE] gibbonFormGroup | [TRIAGE] CORE | [CATEGORY] Academic Cycle, Groups, Rooms & Timetable Spine
CREATE TABLE `gibbonFormGroup` (
  `gibbonFormGroupID` int(5) UNSIGNED ZEROFILL NOT NULL,
  `gibbonSchoolYearID` int(3) UNSIGNED ZEROFILL NOT NULL,
  `name` varchar(20) NOT NULL,
  `nameShort` varchar(8) NOT NULL,
  `gibbonPersonIDTutor` int(10) UNSIGNED ZEROFILL DEFAULT NULL,
  `gibbonPersonIDTutor2` int(10) UNSIGNED ZEROFILL DEFAULT NULL,
  `gibbonPersonIDTutor3` int(10) UNSIGNED ZEROFILL DEFAULT NULL,
  `gibbonPersonIDEA` int(10) UNSIGNED ZEROFILL DEFAULT NULL,
  `gibbonPersonIDEA2` int(10) UNSIGNED ZEROFILL DEFAULT NULL,
  `gibbonPersonIDEA3` int(10) UNSIGNED ZEROFILL DEFAULT NULL,
  `gibbonSpaceID` int(5) UNSIGNED ZEROFILL DEFAULT NULL,
  `gibbonFormGroupIDNext` int(5) UNSIGNED ZEROFILL DEFAULT NULL,
  `attendance` enum('Y','N') NOT NULL DEFAULT 'Y',
  `website` varchar(255) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonFormGroup`
  ADD PRIMARY KEY (`gibbonFormGroupID`);

-- [TABLE] gibbonGroup | [TRIAGE] CORE | [CATEGORY] Academic Cycle, Groups, Rooms & Timetable Spine
CREATE TABLE `gibbonGroup` (
  `gibbonGroupID` int(8) UNSIGNED ZEROFILL NOT NULL,
  `gibbonPersonIDOwner` int(10) UNSIGNED ZEROFILL NOT NULL,
  `gibbonSchoolYearID` int(3) UNSIGNED ZEROFILL NOT NULL,
  `name` varchar(60) NOT NULL,
  `timestampCreated` timestamp NULL DEFAULT NULL,
  `timestampUpdated` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonGroup`
  ADD PRIMARY KEY (`gibbonGroupID`);

-- [TABLE] gibbonGroupPerson | [TRIAGE] CORE | [CATEGORY] Academic Cycle, Groups, Rooms & Timetable Spine
CREATE TABLE `gibbonGroupPerson` (
  `gibbonGroupPersonID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `gibbonGroupID` int(8) UNSIGNED ZEROFILL NOT NULL,
  `gibbonPersonID` int(10) UNSIGNED ZEROFILL NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonGroupPerson`
  ADD PRIMARY KEY (`gibbonGroupPersonID`),
  ADD UNIQUE KEY `gibbonGroupID` (`gibbonGroupID`,`gibbonPersonID`);

-- [TABLE] gibbonHouse | [TRIAGE] CORE | [CATEGORY] Academic Cycle, Groups, Rooms & Timetable Spine
CREATE TABLE `gibbonHouse` (
  `gibbonHouseID` int(3) UNSIGNED ZEROFILL NOT NULL,
  `name` varchar(30) NOT NULL,
  `nameShort` varchar(10) NOT NULL,
  `logo` varchar(255) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonHouse`
  ADD PRIMARY KEY (`gibbonHouseID`),
  ADD UNIQUE KEY `name` (`name`,`nameShort`);

-- [TABLE] gibbonSchoolYear | [TRIAGE] CORE | [CATEGORY] Academic Cycle, Groups, Rooms & Timetable Spine
CREATE TABLE `gibbonSchoolYear` (
  `gibbonSchoolYearID` int(3) UNSIGNED ZEROFILL NOT NULL,
  `name` varchar(9) NOT NULL DEFAULT '',
  `status` enum('Past','Current','Upcoming') NOT NULL DEFAULT 'Upcoming',
  `sequenceNumber` int(3) NOT NULL,
  `firstDay` date DEFAULT NULL,
  `lastDay` date DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonSchoolYear`
  ADD PRIMARY KEY (`gibbonSchoolYearID`),
  ADD UNIQUE KEY `academicYearName` (`name`),
  ADD UNIQUE KEY `sequenceNumber` (`sequenceNumber`);

-- [TABLE] gibbonSchoolYearSpecialDay | [TRIAGE] CORE | [CATEGORY] Academic Cycle, Groups, Rooms & Timetable Spine
CREATE TABLE `gibbonSchoolYearSpecialDay` (
  `gibbonSchoolYearSpecialDayID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `gibbonSchoolYearTermID` int(5) UNSIGNED ZEROFILL NOT NULL,
  `type` enum('School Closure','Timing Change','Off Timetable') DEFAULT NULL,
  `name` varchar(20) NOT NULL,
  `description` varchar(255) NOT NULL,
  `date` date NOT NULL,
  `schoolOpen` time DEFAULT NULL,
  `schoolStart` time DEFAULT NULL,
  `schoolEnd` time DEFAULT NULL,
  `schoolClose` time DEFAULT NULL,
  `context` enum('Year Group','Form Group') DEFAULT NULL,
  `gibbonYearGroupIDList` varchar(255) DEFAULT NULL,
  `gibbonFormGroupIDList` varchar(255) DEFAULT NULL,
  `cancelActivities` enum('Y','N') DEFAULT 'N'
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonSchoolYearSpecialDay`
  ADD PRIMARY KEY (`gibbonSchoolYearSpecialDayID`),
  ADD UNIQUE KEY `date` (`date`);

-- [TABLE] gibbonSchoolYearTerm | [TRIAGE] CORE | [CATEGORY] Academic Cycle, Groups, Rooms & Timetable Spine
CREATE TABLE `gibbonSchoolYearTerm` (
  `gibbonSchoolYearTermID` int(5) UNSIGNED ZEROFILL NOT NULL,
  `gibbonSchoolYearID` int(3) UNSIGNED ZEROFILL NOT NULL,
  `sequenceNumber` int(5) NOT NULL,
  `name` varchar(20) NOT NULL,
  `nameShort` varchar(4) NOT NULL,
  `firstDay` date NOT NULL,
  `lastDay` date NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonSchoolYearTerm`
  ADD PRIMARY KEY (`gibbonSchoolYearTermID`),
  ADD UNIQUE KEY `sequenceNumber` (`sequenceNumber`,`gibbonSchoolYearID`);

-- [TABLE] gibbonSpace | [TRIAGE] CORE | [CATEGORY] Academic Cycle, Groups, Rooms & Timetable Spine
CREATE TABLE `gibbonSpace` (
  `gibbonSpaceID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `name` varchar(30) NOT NULL,
  `type` varchar(50) NOT NULL,
  `active` enum('N','Y') NOT NULL DEFAULT 'Y',
  `capacity` int(5) NOT NULL,
  `computer` enum('N','Y') NOT NULL,
  `computerStudent` int(3) NOT NULL DEFAULT '0',
  `projector` enum('N','Y') NOT NULL,
  `tv` enum('N','Y') NOT NULL,
  `dvd` enum('N','Y') NOT NULL,
  `hifi` enum('N','Y') NOT NULL,
  `speakers` enum('N','Y') NOT NULL,
  `iwb` enum('N','Y') NOT NULL,
  `phoneInternal` varchar(5) NOT NULL,
  `phoneExternal` varchar(20) NOT NULL,
  `comment` text NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonSpace`
  ADD PRIMARY KEY (`gibbonSpaceID`),
  ADD UNIQUE KEY `name` (`name`);

-- [TABLE] gibbonSpacePerson | [TRIAGE] CORE | [CATEGORY] Academic Cycle, Groups, Rooms & Timetable Spine
CREATE TABLE `gibbonSpacePerson` (
  `gibbonSpacePersonID` int(12) UNSIGNED ZEROFILL NOT NULL,
  `gibbonSpaceID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `gibbonPersonID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `usageType` enum('','Teaching','Office','Other') COLLATE utf8_unicode_ci DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_unicode_ci;
ALTER TABLE `gibbonSpacePerson`
  ADD PRIMARY KEY (`gibbonSpacePersonID`);

-- [TABLE] gibbonTT | [TRIAGE] CORE | [CATEGORY] Academic Cycle, Groups, Rooms & Timetable Spine
CREATE TABLE `gibbonTT` (
  `gibbonTTID` int(8) UNSIGNED ZEROFILL NOT NULL,
  `gibbonSchoolYearID` int(3) UNSIGNED ZEROFILL NOT NULL,
  `name` varchar(30) NOT NULL,
  `nameShort` varchar(12) NOT NULL,
  `nameShortDisplay` enum('Day Of The Week','Timetable Day Short Name','') NOT NULL DEFAULT 'Day Of The Week',
  `gibbonYearGroupIDList` varchar(255) NOT NULL,
  `active` enum('Y','N') NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonTT`
  ADD PRIMARY KEY (`gibbonTTID`);

-- [TABLE] gibbonTTColumn | [TRIAGE] CORE | [CATEGORY] Academic Cycle, Groups, Rooms & Timetable Spine
CREATE TABLE `gibbonTTColumn` (
  `gibbonTTColumnID` int(6) UNSIGNED ZEROFILL NOT NULL,
  `name` varchar(30) NOT NULL,
  `nameShort` varchar(12) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonTTColumn`
  ADD PRIMARY KEY (`gibbonTTColumnID`);

-- [TABLE] gibbonTTColumnRow | [TRIAGE] CORE | [CATEGORY] Academic Cycle, Groups, Rooms & Timetable Spine
CREATE TABLE `gibbonTTColumnRow` (
  `gibbonTTColumnRowID` int(8) UNSIGNED ZEROFILL NOT NULL,
  `gibbonTTColumnID` int(6) UNSIGNED ZEROFILL NOT NULL,
  `name` varchar(12) NOT NULL,
  `nameShort` varchar(4) NOT NULL,
  `timeStart` time NOT NULL,
  `timeEnd` time NOT NULL,
  `type` enum('Lesson','Pastoral','Sport','Break','Service','Other') NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonTTColumnRow`
  ADD PRIMARY KEY (`gibbonTTColumnRowID`),
  ADD KEY `gibbonTTColumnID` (`gibbonTTColumnID`);

-- [TABLE] gibbonTTDay | [TRIAGE] CORE | [CATEGORY] Academic Cycle, Groups, Rooms & Timetable Spine
CREATE TABLE `gibbonTTDay` (
  `gibbonTTDayID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `gibbonTTID` int(8) UNSIGNED ZEROFILL NOT NULL,
  `gibbonTTColumnID` int(6) UNSIGNED ZEROFILL NOT NULL,
  `name` varchar(12) NOT NULL,
  `nameShort` varchar(4) NOT NULL,
  `color` varchar(7) NOT NULL COMMENT 'RGB Hex, leading #',
  `fontColor` varchar(7) NOT NULL COMMENT 'RGB Hex, leading #'
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonTTDay`
  ADD PRIMARY KEY (`gibbonTTDayID`);

-- [TABLE] gibbonTTDayDate | [TRIAGE] CORE | [CATEGORY] Academic Cycle, Groups, Rooms & Timetable Spine
CREATE TABLE `gibbonTTDayDate` (
  `gibbonTTDayDateID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `gibbonTTDayID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `date` date NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonTTDayDate`
  ADD PRIMARY KEY (`gibbonTTDayDateID`),
  ADD KEY `gibbonTTDayID` (`gibbonTTDayID`);

-- [TABLE] gibbonTTDayRowClass | [TRIAGE] CORE | [CATEGORY] Academic Cycle, Groups, Rooms & Timetable Spine
CREATE TABLE `gibbonTTDayRowClass` (
  `gibbonTTDayRowClassID` int(12) UNSIGNED ZEROFILL NOT NULL,
  `gibbonTTColumnRowID` int(8) UNSIGNED ZEROFILL NOT NULL,
  `gibbonTTDayID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `gibbonCourseClassID` int(8) UNSIGNED ZEROFILL NOT NULL,
  `gibbonSpaceID` int(10) UNSIGNED ZEROFILL DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonTTDayRowClass`
  ADD PRIMARY KEY (`gibbonTTDayRowClassID`),
  ADD KEY `gibbonCourseClassID` (`gibbonCourseClassID`),
  ADD KEY `gibbonSpaceID` (`gibbonSpaceID`),
  ADD KEY `gibbonTTColumnRowID` (`gibbonTTColumnRowID`);

-- [TABLE] gibbonTTDayRowClassException | [TRIAGE] CORE | [CATEGORY] Academic Cycle, Groups, Rooms & Timetable Spine
CREATE TABLE `gibbonTTDayRowClassException` (
  `gibbonTTDayRowClassExceptionID` int(14) UNSIGNED ZEROFILL NOT NULL,
  `gibbonTTDayRowClassID` int(12) UNSIGNED ZEROFILL NOT NULL,
  `gibbonPersonID` int(10) UNSIGNED ZEROFILL NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonTTDayRowClassException`
  ADD PRIMARY KEY (`gibbonTTDayRowClassExceptionID`);

-- [TABLE] gibbonTTImport | [TRIAGE] PLUGIN | [CATEGORY] Academic Cycle, Groups, Rooms & Timetable Spine
CREATE TABLE `gibbonTTImport` (
  `gibbonTTImportID` int(14) UNSIGNED ZEROFILL NOT NULL,
  `courseNameShort` varchar(12) NOT NULL DEFAULT '',
  `classNameShort` varchar(8) NOT NULL DEFAULT '',
  `dayName` varchar(12) NOT NULL,
  `rowName` varchar(12) NOT NULL,
  `teacherUsernameList` text NOT NULL,
  `spaceName` varchar(30) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonTTImport`
  ADD PRIMARY KEY (`gibbonTTImportID`);

-- [TABLE] gibbonTTSpaceBooking | [TRIAGE] PLUGIN | [CATEGORY] Academic Cycle, Groups, Rooms & Timetable Spine
CREATE TABLE `gibbonTTSpaceBooking` (
  `gibbonTTSpaceBookingID` int(12) UNSIGNED ZEROFILL NOT NULL,
  `foreignKey` enum('gibbonSpaceID','gibbonLibraryItemID') NOT NULL DEFAULT 'gibbonSpaceID',
  `foreignKeyID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `gibbonPersonID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `date` date NOT NULL,
  `timeStart` time NOT NULL,
  `timeEnd` time NOT NULL,
  `reason` varchar(255) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonTTSpaceBooking`
  ADD PRIMARY KEY (`gibbonTTSpaceBookingID`);

-- [TABLE] gibbonTTSpaceChange | [TRIAGE] CORE | [CATEGORY] Academic Cycle, Groups, Rooms & Timetable Spine
CREATE TABLE `gibbonTTSpaceChange` (
  `gibbonTTSpaceChangeID` int(12) UNSIGNED ZEROFILL NOT NULL,
  `gibbonTTDayRowClassID` int(12) UNSIGNED ZEROFILL NOT NULL,
  `gibbonSpaceID` int(10) UNSIGNED ZEROFILL DEFAULT NULL,
  `date` date NOT NULL,
  `gibbonPersonID` int(10) UNSIGNED ZEROFILL NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonTTSpaceChange`
  ADD PRIMARY KEY (`gibbonTTSpaceChangeID`),
  ADD KEY `gibbonTTDayRowClassID` (`gibbonTTDayRowClassID`),
  ADD KEY `date` (`date`);

-- [TABLE] gibbonYearGroup | [TRIAGE] CORE | [CATEGORY] Academic Cycle, Groups, Rooms & Timetable Spine
CREATE TABLE `gibbonYearGroup` (
  `gibbonYearGroupID` int(3) UNSIGNED ZEROFILL NOT NULL,
  `name` varchar(15) NOT NULL,
  `nameShort` varchar(4) NOT NULL,
  `sequenceNumber` int(3) NOT NULL,
  `gibbonPersonIDHOY` int(10) UNSIGNED ZEROFILL DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonYearGroup`
  ADD PRIMARY KEY (`gibbonYearGroupID`),
  ADD UNIQUE KEY `name` (`name`,`nameShort`,`sequenceNumber`);

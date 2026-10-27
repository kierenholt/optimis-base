-- ============================================================================
-- Gibbon Blueprint Domain File: 04_attendance.sql
-- Tables in this file: 4
-- Note: Stripped of INSERT seed data for fast AI agent reading.
-- ============================================================================

-- [TABLE] gibbonAttendanceCode | [TRIAGE] CORE | [CATEGORY] Statutory AM/PM & Class Attendance
CREATE TABLE `gibbonAttendanceCode` (
  `gibbonAttendanceCodeID` int(3) UNSIGNED ZEROFILL NOT NULL,
  `name` varchar(30) NOT NULL,
  `nameShort` varchar(4) NOT NULL,
  `type` enum('Core','Additional') NOT NULL,
  `direction` enum('In','Out') NOT NULL,
  `scope` enum('Onsite','Onsite - Late','Offsite','Offsite - Left','Offsite - Late') NOT NULL,
  `active` enum('Y','N') NOT NULL,
  `reportable` enum('Y','N') NOT NULL,
  `future` enum('Y','N') NOT NULL,
  `prefill` enum('Y','N') NOT NULL DEFAULT 'Y',
  `gibbonRoleIDAll` varchar(90) NOT NULL,
  `sequenceNumber` int(3) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonAttendanceCode`
  ADD PRIMARY KEY (`gibbonAttendanceCodeID`),
  ADD UNIQUE KEY `name` (`name`),
  ADD UNIQUE KEY `nameShort` (`nameShort`);

-- [TABLE] gibbonAttendanceLogCourseClass | [TRIAGE] CORE | [CATEGORY] Statutory AM/PM & Class Attendance
CREATE TABLE `gibbonAttendanceLogCourseClass` (
  `gibbonAttendanceLogCourseClassID` int(14) UNSIGNED ZEROFILL NOT NULL,
  `gibbonCourseClassID` int(8) UNSIGNED ZEROFILL NOT NULL,
  `gibbonTTDayRowClassID` int(12) UNSIGNED ZEROFILL DEFAULT NULL,
  `gibbonPersonIDTaker` int(10) UNSIGNED ZEROFILL NOT NULL,
  `date` date DEFAULT NULL,
  `timestampTaken` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonAttendanceLogCourseClass`
  ADD PRIMARY KEY (`gibbonAttendanceLogCourseClassID`);

-- [TABLE] gibbonAttendanceLogFormGroup | [TRIAGE] CORE | [CATEGORY] Statutory AM/PM & Class Attendance
CREATE TABLE `gibbonAttendanceLogFormGroup` (
  `gibbonAttendanceLogFormGroupID` int(14) UNSIGNED ZEROFILL NOT NULL,
  `gibbonFormGroupID` int(5) UNSIGNED ZEROFILL NOT NULL,
  `gibbonPersonIDTaker` int(10) UNSIGNED ZEROFILL NOT NULL,
  `date` date DEFAULT NULL,
  `timestampTaken` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonAttendanceLogFormGroup`
  ADD PRIMARY KEY (`gibbonAttendanceLogFormGroupID`);

-- [TABLE] gibbonAttendanceLogPerson | [TRIAGE] CORE | [CATEGORY] Statutory AM/PM & Class Attendance
CREATE TABLE `gibbonAttendanceLogPerson` (
  `gibbonAttendanceLogPersonID` int(14) UNSIGNED ZEROFILL NOT NULL,
  `gibbonAttendanceCodeID` int(3) UNSIGNED ZEROFILL DEFAULT NULL,
  `gibbonPersonID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `direction` enum('In','Out') NOT NULL,
  `type` varchar(30) NOT NULL DEFAULT '',
  `reason` varchar(30) NOT NULL DEFAULT '',
  `context` enum('Form Group','Class','Person','Future','Self Registration') DEFAULT NULL,
  `comment` varchar(255) NOT NULL,
  `date` date DEFAULT NULL,
  `gibbonPersonIDTaker` int(10) UNSIGNED ZEROFILL NOT NULL,
  `gibbonFormGroupID` int(5) UNSIGNED ZEROFILL DEFAULT NULL,
  `gibbonCourseClassID` int(8) UNSIGNED ZEROFILL DEFAULT NULL,
  `gibbonTTDayRowClassID` int(12) UNSIGNED ZEROFILL DEFAULT NULL,
  `timestampTaken` timestamp NULL DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonAttendanceLogPerson`
  ADD PRIMARY KEY (`gibbonAttendanceLogPersonID`),
  ADD KEY `date` (`date`),
  ADD KEY `gibbonPersonID` (`gibbonPersonID`),
  ADD KEY `dateAndPerson` (`date`,`gibbonPersonID`) USING BTREE,
  ADD KEY `context` (`context`,`gibbonCourseClassID`);

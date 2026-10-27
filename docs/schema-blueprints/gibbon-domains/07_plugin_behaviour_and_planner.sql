-- ============================================================================
-- Gibbon Blueprint Domain File: 07_plugin_behaviour_and_planner.sql
-- Tables in this file: 28
-- Note: Stripped of INSERT seed data for fast AI agent reading.
-- ============================================================================

-- [TABLE] gibbonActivity | [TRIAGE] PLUGIN | [CATEGORY] Behaviour, Lesson Planner, Units & Co-Curricular
CREATE TABLE `gibbonActivity` (
  `gibbonActivityID` int(8) UNSIGNED ZEROFILL NOT NULL,
  `gibbonSchoolYearID` int(3) UNSIGNED ZEROFILL NOT NULL DEFAULT '000',
  `gibbonActivityCategoryID` int(8) UNSIGNED ZEROFILL DEFAULT NULL,
  `active` enum('Y','N') NOT NULL DEFAULT 'Y',
  `registration` enum('Y','N') NOT NULL DEFAULT 'Y' COMMENT 'Can a parent/student select this for registration?',
  `name` varchar(40) NOT NULL DEFAULT '',
  `provider` enum('School','External') NOT NULL DEFAULT 'School',
  `type` varchar(255) NOT NULL,
  `gibbonSchoolYearTermIDList` text NOT NULL,
  `listingStart` date DEFAULT NULL,
  `listingEnd` date DEFAULT NULL,
  `programStart` date DEFAULT NULL,
  `programEnd` date DEFAULT NULL,
  `gibbonYearGroupIDList` varchar(255) NOT NULL DEFAULT '',
  `maxParticipants` int(3) NOT NULL DEFAULT '0',
  `description` text,
  `payment` decimal(8,2) DEFAULT NULL,
  `paymentType` enum('Entire Programme','Per Session','Per Week','Per Term') DEFAULT 'Entire Programme',
  `paymentFirmness` enum('Finalised','Estimated') DEFAULT 'Finalised'
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonActivity`
  ADD PRIMARY KEY (`gibbonActivityID`);

-- [TABLE] gibbonActivityAttendance | [TRIAGE] PLUGIN | [CATEGORY] Behaviour, Lesson Planner, Units & Co-Curricular
CREATE TABLE `gibbonActivityAttendance` (
  `gibbonActivityAttendanceID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `gibbonActivityID` int(8) UNSIGNED ZEROFILL NOT NULL,
  `gibbonPersonIDTaker` int(10) UNSIGNED ZEROFILL NOT NULL,
  `attendance` text NOT NULL,
  `date` date DEFAULT NULL,
  `timestampTaken` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonActivityAttendance`
  ADD PRIMARY KEY (`gibbonActivityAttendanceID`);

-- [TABLE] gibbonActivityCategory | [TRIAGE] PLUGIN | [CATEGORY] Behaviour, Lesson Planner, Units & Co-Curricular
CREATE TABLE `gibbonActivityCategory` (
  `gibbonActivityCategoryID` int(8) UNSIGNED ZEROFILL NOT NULL,
  `gibbonSchoolYearID` int(3) UNSIGNED ZEROFILL NOT NULL,
  `name` varchar(90) NOT NULL,
  `nameShort` varchar(12) NOT NULL,
  `description` text,
  `active` enum('Y','N') NOT NULL DEFAULT 'N',
  `signUpChoices` int(3) NOT NULL DEFAULT '3',
  `viewableDate` datetime DEFAULT NULL,
  `accessOpenDate` datetime DEFAULT NULL,
  `accessCloseDate` datetime DEFAULT NULL,
  `accessEnrolmentDate` datetime DEFAULT NULL,
  `backgroundImage` text,
  `sequenceNumber` int(6) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonActivityCategory`
  ADD PRIMARY KEY (`gibbonActivityCategoryID`),
  ADD UNIQUE KEY `name` (`name`,`gibbonSchoolYearID`);

-- [TABLE] gibbonActivityChoice | [TRIAGE] PLUGIN | [CATEGORY] Behaviour, Lesson Planner, Units & Co-Curricular
CREATE TABLE `gibbonActivityChoice` (
  `gibbonActivityChoiceID` int(12) UNSIGNED ZEROFILL NOT NULL,
  `gibbonActivityCategoryID` int(8) UNSIGNED ZEROFILL NOT NULL,
  `gibbonActivityID` int(8) UNSIGNED ZEROFILL NOT NULL,
  `gibbonPersonID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `choice` int(2) NOT NULL DEFAULT '1',
  `timestampCreated` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `timestampModified` timestamp NULL DEFAULT NULL,
  `gibbonPersonIDCreated` int(10) UNSIGNED ZEROFILL NOT NULL,
  `gibbonPersonIDModified` int(10) UNSIGNED ZEROFILL NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonActivityChoice`
  ADD PRIMARY KEY (`gibbonActivityChoiceID`),
  ADD UNIQUE KEY `gibbonPersonID` (`gibbonPersonID`,`choice`,`gibbonActivityCategoryID`);

-- [TABLE] gibbonActivityPhoto | [TRIAGE] PLUGIN | [CATEGORY] Behaviour, Lesson Planner, Units & Co-Curricular
CREATE TABLE `gibbonActivityPhoto` (
  `gibbonActivityPhotoID` int(12) UNSIGNED ZEROFILL NOT NULL,
  `gibbonActivityID` int(8) UNSIGNED ZEROFILL NOT NULL,
  `filePath` varchar(255) NOT NULL,
  `caption` varchar(120) NOT NULL,
  `sequenceNumber` int(6) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonActivityPhoto`
  ADD PRIMARY KEY (`gibbonActivityPhotoID`);

-- [TABLE] gibbonActivitySlot | [TRIAGE] PLUGIN | [CATEGORY] Behaviour, Lesson Planner, Units & Co-Curricular
CREATE TABLE `gibbonActivitySlot` (
  `gibbonActivitySlotID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `gibbonActivityID` int(8) UNSIGNED ZEROFILL NOT NULL,
  `gibbonSpaceID` int(10) UNSIGNED ZEROFILL DEFAULT NULL,
  `locationExternal` varchar(50) NOT NULL,
  `gibbonDaysOfWeekID` int(2) UNSIGNED ZEROFILL NOT NULL,
  `timeStart` time NOT NULL,
  `timeEnd` time NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonActivitySlot`
  ADD PRIMARY KEY (`gibbonActivitySlotID`);

-- [TABLE] gibbonActivityStaff | [TRIAGE] PLUGIN | [CATEGORY] Behaviour, Lesson Planner, Units & Co-Curricular
CREATE TABLE `gibbonActivityStaff` (
  `gibbonActivityStaffID` int(8) UNSIGNED ZEROFILL NOT NULL,
  `gibbonActivityID` int(8) UNSIGNED ZEROFILL NOT NULL DEFAULT '00000000',
  `gibbonPersonID` int(10) UNSIGNED ZEROFILL NOT NULL DEFAULT '0000000000',
  `role` enum('Organiser','Coach','Assistant','Other') NOT NULL DEFAULT 'Organiser'
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonActivityStaff`
  ADD PRIMARY KEY (`gibbonActivityStaffID`),
  ADD KEY `gibbonActivityID` (`gibbonActivityID`,`gibbonPersonID`);

-- [TABLE] gibbonActivityStudent | [TRIAGE] PLUGIN | [CATEGORY] Behaviour, Lesson Planner, Units & Co-Curricular
CREATE TABLE `gibbonActivityStudent` (
  `gibbonActivityStudentID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `gibbonActivityID` int(8) UNSIGNED ZEROFILL NOT NULL DEFAULT '00000000',
  `gibbonPersonID` int(10) UNSIGNED ZEROFILL NOT NULL DEFAULT '0000000000',
  `status` enum('Accepted','Pending','Waiting List','Not Accepted','Left') NOT NULL DEFAULT 'Pending',
  `timestamp` datetime NOT NULL,
  `gibbonActivityIDBackup` int(8) UNSIGNED ZEROFILL DEFAULT NULL,
  `invoiceGenerated` enum('N','Y') NOT NULL DEFAULT 'N',
  `gibbonFinanceInvoiceID` int(14) UNSIGNED ZEROFILL DEFAULT NULL,
  `gibbonActivityChoiceID` int(12) UNSIGNED ZEROFILL DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonActivityStudent`
  ADD PRIMARY KEY (`gibbonActivityStudentID`),
  ADD KEY `gibbonActivityID` (`gibbonActivityID`,`status`);

-- [TABLE] gibbonActivityType | [TRIAGE] PLUGIN | [CATEGORY] Behaviour, Lesson Planner, Units & Co-Curricular
CREATE TABLE `gibbonActivityType` (
  `gibbonActivityTypeID` int(6) UNSIGNED ZEROFILL NOT NULL,
  `name` varchar(60) DEFAULT NULL,
  `description` text,
  `access` enum('None','View','Register') DEFAULT 'Register',
  `enrolmentType` enum('Competitive','Selection') DEFAULT 'Competitive',
  `maxPerStudent` int(3) NOT NULL DEFAULT '0',
  `waitingList` enum('Y','N') DEFAULT 'Y',
  `backupChoice` enum('Y','N') DEFAULT 'Y'
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonActivityType`
  ADD PRIMARY KEY (`gibbonActivityTypeID`),
  ADD UNIQUE KEY `name` (`name`);

-- [TABLE] gibbonBehaviour | [TRIAGE] PLUGIN | [CATEGORY] Behaviour, Lesson Planner, Units & Co-Curricular
CREATE TABLE `gibbonBehaviour` (
  `gibbonBehaviourID` int(12) UNSIGNED ZEROFILL NOT NULL,
  `gibbonSchoolYearID` int(3) UNSIGNED ZEROFILL NOT NULL,
  `date` date NOT NULL,
  `gibbonPersonID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `type` enum('Positive','Negative') CHARACTER SET utf8 NOT NULL,
  `descriptor` varchar(100) CHARACTER SET utf8 DEFAULT NULL,
  `level` varchar(100) CHARACTER SET utf8 DEFAULT NULL,
  `comment` text CHARACTER SET utf8 NOT NULL,
  `followup` text COLLATE utf8_unicode_ci NOT NULL,
  `gibbonPlannerEntryID` int(14) UNSIGNED ZEROFILL DEFAULT NULL,
  `gibbonPersonIDCreator` int(10) UNSIGNED ZEROFILL NOT NULL,
  `timestamp` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `fields` text COLLATE utf8_unicode_ci,
  `gibbonMultiIncidentID` varchar(64) COLLATE utf8_unicode_ci DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_unicode_ci;
ALTER TABLE `gibbonBehaviour`
  ADD PRIMARY KEY (`gibbonBehaviourID`),
  ADD KEY `gibbonPersonID` (`gibbonPersonID`);

-- [TABLE] gibbonBehaviourFollowUp | [TRIAGE] PLUGIN | [CATEGORY] Behaviour, Lesson Planner, Units & Co-Curricular
CREATE TABLE `gibbonBehaviourFollowUp` (
  `gibbonBehaviourFollowUpID` int(11) UNSIGNED ZEROFILL NOT NULL,
  `gibbonBehaviourID` int(12) UNSIGNED ZEROFILL NOT NULL,
  `gibbonPersonID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `followUp` text,
  `timestamp` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonBehaviourFollowUp`
  ADD PRIMARY KEY (`gibbonBehaviourFollowUpID`),
  ADD KEY `gibbonBehaviourID` (`gibbonBehaviourID`);

-- [TABLE] gibbonBehaviourLetter | [TRIAGE] PLUGIN | [CATEGORY] Behaviour, Lesson Planner, Units & Co-Curricular
CREATE TABLE `gibbonBehaviourLetter` (
  `gibbonBehaviourLetterID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `gibbonSchoolYearID` int(3) UNSIGNED ZEROFILL NOT NULL,
  `gibbonPersonID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `letterLevel` enum('1','2','3') NOT NULL,
  `status` enum('Warning','Issued') NOT NULL,
  `type` enum('Negative','Positive') NOT NULL DEFAULT 'Negative',
  `recordCountAtCreation` int(3) NOT NULL,
  `body` text NOT NULL,
  `recipientList` text NOT NULL,
  `timestamp` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonBehaviourLetter`
  ADD PRIMARY KEY (`gibbonBehaviourLetterID`);

-- [TABLE] gibbonDiscussion | [TRIAGE] PLUGIN | [CATEGORY] Behaviour, Lesson Planner, Units & Co-Curricular
CREATE TABLE `gibbonDiscussion` (
  `gibbonDiscussionID` int(12) UNSIGNED ZEROFILL NOT NULL,
  `foreignTable` varchar(60) NOT NULL,
  `foreignTableID` int(14) UNSIGNED ZEROFILL NOT NULL,
  `gibbonModuleID` int(4) UNSIGNED ZEROFILL NOT NULL,
  `gibbonPersonID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `gibbonPersonIDTarget` int(10) UNSIGNED ZEROFILL DEFAULT NULL,
  `type` varchar(60) DEFAULT NULL,
  `tag` varchar(60) DEFAULT NULL,
  `comment` text,
  `attachmentType` enum('File','Link') DEFAULT NULL,
  `attachmentLocation` text,
  `gibbonDiscussionIDReplyTo` int(12) UNSIGNED ZEROFILL DEFAULT NULL,
  `timestamp` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonDiscussion`
  ADD PRIMARY KEY (`gibbonDiscussionID`),
  ADD KEY `foreignTable` (`foreignTable`,`foreignTableID`),
  ADD KEY `gibbonPersonID` (`gibbonPersonID`);

-- [TABLE] gibbonPlannerEntry | [TRIAGE] PLUGIN | [CATEGORY] Behaviour, Lesson Planner, Units & Co-Curricular
CREATE TABLE `gibbonPlannerEntry` (
  `gibbonPlannerEntryID` int(14) UNSIGNED ZEROFILL NOT NULL,
  `gibbonCourseClassID` int(8) UNSIGNED ZEROFILL NOT NULL,
  `gibbonUnitID` int(10) UNSIGNED ZEROFILL DEFAULT NULL,
  `date` date DEFAULT NULL,
  `timeStart` time DEFAULT NULL,
  `timeEnd` time DEFAULT NULL,
  `name` varchar(50) NOT NULL,
  `summary` varchar(255) NOT NULL,
  `description` mediumtext NOT NULL,
  `teachersNotes` mediumtext NOT NULL,
  `homework` enum('N','Y') NOT NULL DEFAULT 'N',
  `homeworkDueDateTime` datetime DEFAULT NULL,
  `homeworkDetails` mediumtext NOT NULL,
  `homeworkTimeCap` int(3) DEFAULT NULL,
  `homeworkLocation` enum('Out of Class','In Class') DEFAULT NULL,
  `homeworkSubmission` enum('N','Y') NOT NULL,
  `homeworkSubmissionDateOpen` date DEFAULT NULL,
  `homeworkSubmissionDrafts` varchar(1) DEFAULT NULL,
  `homeworkSubmissionType` enum('','Link','File','Link/File') NOT NULL,
  `homeworkSubmissionRequired` enum('Optional','Required') DEFAULT NULL,
  `homeworkCrowdAssess` enum('N','Y') NOT NULL,
  `homeworkCrowdAssessOtherTeachersRead` enum('N','Y') NOT NULL,
  `homeworkCrowdAssessOtherParentsRead` enum('N','Y') NOT NULL,
  `homeworkCrowdAssessClassmatesParentsRead` enum('N','Y') NOT NULL,
  `homeworkCrowdAssessSubmitterParentsRead` enum('N','Y') NOT NULL,
  `homeworkCrowdAssessOtherStudentsRead` enum('N','Y') NOT NULL,
  `homeworkCrowdAssessClassmatesRead` enum('N','Y') NOT NULL,
  `viewableStudents` enum('Y','N') NOT NULL DEFAULT 'Y',
  `viewableParents` enum('Y','N') NOT NULL DEFAULT 'N',
  `gibbonPersonIDCreator` int(10) UNSIGNED ZEROFILL NOT NULL,
  `gibbonPersonIDLastEdit` int(10) UNSIGNED ZEROFILL NOT NULL,
  `fields` text
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonPlannerEntry`
  ADD PRIMARY KEY (`gibbonPlannerEntryID`),
  ADD KEY `gibbonCourseClassID` (`gibbonCourseClassID`);

-- [TABLE] gibbonPlannerEntryDiscuss | [TRIAGE] PLUGIN | [CATEGORY] Behaviour, Lesson Planner, Units & Co-Curricular
CREATE TABLE `gibbonPlannerEntryDiscuss` (
  `gibbonPlannerEntryDiscussID` int(16) UNSIGNED ZEROFILL NOT NULL,
  `gibbonPlannerEntryID` int(14) UNSIGNED ZEROFILL NOT NULL,
  `gibbonPersonID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `timestamp` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `comment` text NOT NULL,
  `gibbonPlannerEntryDiscussIDReplyTo` int(16) UNSIGNED ZEROFILL DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonPlannerEntryDiscuss`
  ADD PRIMARY KEY (`gibbonPlannerEntryDiscussID`);

-- [TABLE] gibbonPlannerEntryGuest | [TRIAGE] PLUGIN | [CATEGORY] Behaviour, Lesson Planner, Units & Co-Curricular
CREATE TABLE `gibbonPlannerEntryGuest` (
  `gibbonPlannerEntryGuestID` int(16) UNSIGNED ZEROFILL NOT NULL,
  `gibbonPlannerEntryID` int(14) UNSIGNED ZEROFILL NOT NULL,
  `gibbonPersonID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `role` enum('Guest Student','Guest Teacher','Guest Assistant','Guest Technician','Guest Parent','Other Guest') NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonPlannerEntryGuest`
  ADD PRIMARY KEY (`gibbonPlannerEntryGuestID`);

-- [TABLE] gibbonPlannerEntryHomework | [TRIAGE] PLUGIN | [CATEGORY] Behaviour, Lesson Planner, Units & Co-Curricular
CREATE TABLE `gibbonPlannerEntryHomework` (
  `gibbonPlannerEntryHomeworkID` int(16) UNSIGNED ZEROFILL NOT NULL,
  `gibbonPlannerEntryID` int(14) UNSIGNED ZEROFILL NOT NULL,
  `gibbonPersonID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `type` enum('Link','File') NOT NULL,
  `version` enum('Draft','Final') NOT NULL,
  `status` enum('On Time','Late','Exemption') NOT NULL,
  `location` varchar(255) DEFAULT NULL,
  `count` int(1) NOT NULL,
  `timestamp` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonPlannerEntryHomework`
  ADD PRIMARY KEY (`gibbonPlannerEntryHomeworkID`);

-- [TABLE] gibbonPlannerEntryOutcome | [TRIAGE] PLUGIN | [CATEGORY] Behaviour, Lesson Planner, Units & Co-Curricular
CREATE TABLE `gibbonPlannerEntryOutcome` (
  `gibbonPlannerEntryOutcomeID` int(16) UNSIGNED ZEROFILL NOT NULL,
  `gibbonPlannerEntryID` int(14) UNSIGNED ZEROFILL NOT NULL,
  `gibbonOutcomeID` int(8) UNSIGNED ZEROFILL NOT NULL,
  `sequenceNumber` int(4) NOT NULL,
  `content` text NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonPlannerEntryOutcome`
  ADD PRIMARY KEY (`gibbonPlannerEntryOutcomeID`);

-- [TABLE] gibbonPlannerEntryStudentHomework | [TRIAGE] PLUGIN | [CATEGORY] Behaviour, Lesson Planner, Units & Co-Curricular
CREATE TABLE `gibbonPlannerEntryStudentHomework` (
  `gibbonPlannerEntryStudentHomeworkID` int(14) UNSIGNED ZEROFILL NOT NULL,
  `gibbonPlannerEntryID` int(14) UNSIGNED ZEROFILL NOT NULL,
  `gibbonPersonID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `homeworkDueDateTime` datetime NOT NULL,
  `homeworkDetails` mediumtext NOT NULL,
  `homeworkComplete` enum('Y','N') NOT NULL DEFAULT 'N'
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COMMENT='Allows students to add homework deadlines themselves';
ALTER TABLE `gibbonPlannerEntryStudentHomework`
  ADD PRIMARY KEY (`gibbonPlannerEntryStudentHomeworkID`),
  ADD KEY `gibbonPlannerEntryID` (`gibbonPlannerEntryID`,`gibbonPersonID`);

-- [TABLE] gibbonPlannerEntryStudentTracker | [TRIAGE] PLUGIN | [CATEGORY] Behaviour, Lesson Planner, Units & Co-Curricular
CREATE TABLE `gibbonPlannerEntryStudentTracker` (
  `gibbonPlannerEntryStudentTrackerID` int(16) UNSIGNED ZEROFILL NOT NULL,
  `gibbonPlannerEntryID` int(14) UNSIGNED ZEROFILL NOT NULL,
  `gibbonPersonID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `homeworkComplete` enum('Y','N') NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonPlannerEntryStudentTracker`
  ADD PRIMARY KEY (`gibbonPlannerEntryStudentTrackerID`);

-- [TABLE] gibbonPlannerParentWeeklyEmailSummary | [TRIAGE] PLUGIN | [CATEGORY] Behaviour, Lesson Planner, Units & Co-Curricular
CREATE TABLE `gibbonPlannerParentWeeklyEmailSummary` (
  `gibbonPlannerParentWeeklyEmailSummaryID` int(14) UNSIGNED ZEROFILL NOT NULL,
  `gibbonSchoolYearID` int(3) UNSIGNED ZEROFILL NOT NULL,
  `gibbonPersonIDParent` int(10) UNSIGNED ZEROFILL NOT NULL,
  `gibbonPersonIDStudent` int(10) UNSIGNED ZEROFILL NOT NULL,
  `weekOfYear` int(2) NOT NULL,
  `key` varchar(40) NOT NULL,
  `confirmed` enum('N','Y') NOT NULL DEFAULT 'N'
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonPlannerParentWeeklyEmailSummary`
  ADD PRIMARY KEY (`gibbonPlannerParentWeeklyEmailSummaryID`),
  ADD UNIQUE KEY `key` (`key`);

-- [TABLE] gibbonResource | [TRIAGE] PLUGIN | [CATEGORY] Behaviour, Lesson Planner, Units & Co-Curricular
CREATE TABLE `gibbonResource` (
  `gibbonResourceID` int(14) UNSIGNED ZEROFILL NOT NULL,
  `name` varchar(60) NOT NULL,
  `description` text NOT NULL,
  `gibbonYearGroupIDList` varchar(255) NOT NULL,
  `type` enum('File','HTML','Link') NOT NULL,
  `category` varchar(255) NOT NULL,
  `purpose` varchar(255) NOT NULL,
  `tags` text NOT NULL,
  `content` text NOT NULL,
  `gibbonPersonID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `timestamp` timestamp NULL DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonResource`
  ADD PRIMARY KEY (`gibbonResourceID`);

-- [TABLE] gibbonResourceTag | [TRIAGE] PLUGIN | [CATEGORY] Behaviour, Lesson Planner, Units & Co-Curricular
CREATE TABLE `gibbonResourceTag` (
  `gibbonResourceTagID` int(12) UNSIGNED ZEROFILL NOT NULL,
  `tag` varchar(100) NOT NULL,
  `count` int(6) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonResourceTag`
  ADD PRIMARY KEY (`gibbonResourceTagID`),
  ADD UNIQUE KEY `tag` (`tag`),
  ADD KEY `tag_2` (`tag`);

-- [TABLE] gibbonUnit | [TRIAGE] PLUGIN | [CATEGORY] Behaviour, Lesson Planner, Units & Co-Curricular
CREATE TABLE `gibbonUnit` (
  `gibbonUnitID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `gibbonCourseID` int(8) UNSIGNED ZEROFILL NOT NULL,
  `name` varchar(40) NOT NULL,
  `active` enum('Y','N') NOT NULL DEFAULT 'Y',
  `description` text NOT NULL,
  `tags` text NOT NULL,
  `map` enum('Y','N') NOT NULL DEFAULT 'Y' COMMENT 'Should this unit be included in curriculum maps and other summaries?',
  `ordering` int(2) NOT NULL DEFAULT '0',
  `attachment` varchar(255) NOT NULL,
  `details` text NOT NULL,
  `license` varchar(50) DEFAULT NULL,
  `sharedPublic` enum('Y','N') DEFAULT NULL,
  `gibbonPersonIDCreator` int(10) UNSIGNED ZEROFILL NOT NULL,
  `gibbonPersonIDLastEdit` int(10) UNSIGNED ZEROFILL NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonUnit`
  ADD PRIMARY KEY (`gibbonUnitID`);

-- [TABLE] gibbonUnitBlock | [TRIAGE] PLUGIN | [CATEGORY] Behaviour, Lesson Planner, Units & Co-Curricular
CREATE TABLE `gibbonUnitBlock` (
  `gibbonUnitBlockID` int(12) UNSIGNED ZEROFILL NOT NULL,
  `gibbonUnitID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `title` varchar(100) NOT NULL,
  `type` varchar(50) NOT NULL,
  `length` varchar(3) NOT NULL,
  `contents` text NOT NULL,
  `teachersNotes` text NOT NULL,
  `sequenceNumber` int(4) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonUnitBlock`
  ADD PRIMARY KEY (`gibbonUnitBlockID`);

-- [TABLE] gibbonUnitClass | [TRIAGE] PLUGIN | [CATEGORY] Behaviour, Lesson Planner, Units & Co-Curricular
CREATE TABLE `gibbonUnitClass` (
  `gibbonUnitClassID` int(12) UNSIGNED ZEROFILL NOT NULL,
  `gibbonUnitID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `gibbonCourseClassID` int(8) UNSIGNED ZEROFILL NOT NULL,
  `running` enum('N','Y') NOT NULL DEFAULT 'N'
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonUnitClass`
  ADD PRIMARY KEY (`gibbonUnitClassID`);

-- [TABLE] gibbonUnitClassBlock | [TRIAGE] PLUGIN | [CATEGORY] Behaviour, Lesson Planner, Units & Co-Curricular
CREATE TABLE `gibbonUnitClassBlock` (
  `gibbonUnitClassBlockID` int(14) UNSIGNED ZEROFILL NOT NULL,
  `gibbonUnitClassID` int(12) UNSIGNED ZEROFILL NOT NULL,
  `gibbonPlannerEntryID` int(14) UNSIGNED ZEROFILL NOT NULL,
  `gibbonUnitBlockID` int(12) UNSIGNED ZEROFILL NOT NULL,
  `title` varchar(100) NOT NULL,
  `type` varchar(50) NOT NULL,
  `length` varchar(3) NOT NULL,
  `contents` text NOT NULL,
  `teachersNotes` text NOT NULL,
  `sequenceNumber` int(4) NOT NULL,
  `complete` enum('Y','N') NOT NULL DEFAULT 'N'
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonUnitClassBlock`
  ADD PRIMARY KEY (`gibbonUnitClassBlockID`);

-- [TABLE] gibbonUnitOutcome | [TRIAGE] PLUGIN | [CATEGORY] Behaviour, Lesson Planner, Units & Co-Curricular
CREATE TABLE `gibbonUnitOutcome` (
  `gibbonUnitOutcomeID` int(12) UNSIGNED ZEROFILL NOT NULL,
  `gibbonUnitID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `gibbonOutcomeID` int(8) UNSIGNED ZEROFILL NOT NULL,
  `sequenceNumber` int(4) NOT NULL,
  `content` text NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonUnitOutcome`
  ADD PRIMARY KEY (`gibbonUnitOutcomeID`);

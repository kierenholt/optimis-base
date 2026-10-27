-- ============================================================================
-- Gibbon Blueprint Domain File: 08_plugin_finance_ops_and_comms.sql
-- Tables in this file: 40
-- Note: Stripped of INSERT seed data for fast AI agent reading.
-- ============================================================================

-- [TABLE] gibbonAdmissionsAccount | [TRIAGE] PLUGIN | [CATEGORY] Finance, Admissions, Library & Comms
CREATE TABLE `gibbonAdmissionsAccount` (
  `gibbonAdmissionsAccountID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `gibbonPersonID` int(10) UNSIGNED ZEROFILL DEFAULT NULL,
  `gibbonFamilyID` int(7) UNSIGNED ZEROFILL DEFAULT NULL,
  `email` varchar(75) DEFAULT NULL,
  `ipAddress` varchar(40) DEFAULT NULL,
  `accessID` varchar(64) DEFAULT NULL,
  `accessToken` varchar(64) DEFAULT NULL,
  `timestampTokenExpire` timestamp NULL DEFAULT NULL,
  `timestampCreated` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `timestampActive` timestamp NULL DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonAdmissionsAccount`
  ADD PRIMARY KEY (`gibbonAdmissionsAccountID`),
  ADD UNIQUE KEY `gibbonPersonID` (`gibbonPersonID`),
  ADD UNIQUE KEY `email` (`email`),
  ADD UNIQUE KEY `accessID` (`accessID`);

-- [TABLE] gibbonAdmissionsApplication | [TRIAGE] PLUGIN | [CATEGORY] Finance, Admissions, Library & Comms
CREATE TABLE `gibbonAdmissionsApplication` (
  `gibbonAdmissionsApplicationID` int(12) UNSIGNED ZEROFILL NOT NULL,
  `gibbonFormID` int(6) UNSIGNED ZEROFILL DEFAULT NULL,
  `gibbonFormPageID` int(8) UNSIGNED ZEROFILL DEFAULT NULL,
  `gibbonSchoolYearID` int(3) UNSIGNED ZEROFILL DEFAULT NULL,
  `gibbonYearGroupID` int(3) UNSIGNED ZEROFILL DEFAULT NULL,
  `gibbonFormGroupID` int(5) UNSIGNED ZEROFILL DEFAULT NULL,
  `foreignTable` varchar(60) DEFAULT NULL,
  `foreignTableID` int(14) UNSIGNED ZEROFILL DEFAULT NULL,
  `status` enum('Incomplete','Pending','Waiting List','Accepted','Rejected','Withdrawn') NOT NULL DEFAULT 'Incomplete',
  `priority` int(1) NOT NULL DEFAULT '0',
  `owner` varchar(75) DEFAULT NULL,
  `identifier` varchar(255) DEFAULT NULL,
  `data` mediumtext,
  `result` text,
  `milestones` text,
  `gibbonPaymentIDSubmit` int(14) UNSIGNED ZEROFILL DEFAULT NULL,
  `gibbonPaymentIDProcess` int(14) UNSIGNED ZEROFILL DEFAULT NULL,
  `timestampCreated` timestamp NULL DEFAULT NULL,
  `timestampModified` timestamp NULL DEFAULT NULL,
  `gibbonPersonIDModified` int(10) UNSIGNED ZEROFILL DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonAdmissionsApplication`
  ADD PRIMARY KEY (`gibbonAdmissionsApplicationID`),
  ADD KEY `foreignTable` (`foreignTable`,`foreignTableID`);

-- [TABLE] gibbonAlarm | [TRIAGE] PLUGIN | [CATEGORY] Finance, Admissions, Library & Comms
CREATE TABLE `gibbonAlarm` (
  `gibbonAlarmID` int(5) UNSIGNED ZEROFILL NOT NULL,
  `type` enum('General','Lockdown','Custom') DEFAULT NULL,
  `status` enum('Current','Past') NOT NULL DEFAULT 'Past',
  `gibbonPersonID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `timestampStart` timestamp NULL DEFAULT NULL,
  `timestampEnd` timestamp NULL DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonAlarm`
  ADD PRIMARY KEY (`gibbonAlarmID`);

-- [TABLE] gibbonAlarmConfirm | [TRIAGE] PLUGIN | [CATEGORY] Finance, Admissions, Library & Comms
CREATE TABLE `gibbonAlarmConfirm` (
  `gibbonAlarmConfirmID` int(8) UNSIGNED ZEROFILL NOT NULL,
  `gibbonAlarmID` int(5) UNSIGNED ZEROFILL NOT NULL,
  `gibbonPersonID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `timestamp` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonAlarmConfirm`
  ADD PRIMARY KEY (`gibbonAlarmConfirmID`),
  ADD UNIQUE KEY `gibbonAlarmID` (`gibbonAlarmID`,`gibbonPersonID`);

-- [TABLE] gibbonApplicationForm | [TRIAGE] PLUGIN | [CATEGORY] Finance, Admissions, Library & Comms
CREATE TABLE `gibbonApplicationForm` (
  `gibbonApplicationFormID` int(12) UNSIGNED ZEROFILL NOT NULL,
  `gibbonApplicationFormHash` varchar(40) DEFAULT NULL,
  `surname` varchar(60) NOT NULL DEFAULT '',
  `firstName` varchar(60) NOT NULL DEFAULT '',
  `preferredName` varchar(60) NOT NULL DEFAULT '',
  `officialName` varchar(150) NOT NULL,
  `nameInCharacters` varchar(20) NOT NULL,
  `gender` enum('M','F','Other','Unspecified') NOT NULL DEFAULT 'Unspecified',
  `username` varchar(20) DEFAULT NULL,
  `status` enum('Pending','Waiting List','Accepted','Rejected','Withdrawn') NOT NULL DEFAULT 'Pending',
  `dob` date DEFAULT NULL,
  `email` varchar(75) DEFAULT NULL,
  `homeAddress` mediumtext,
  `homeAddressDistrict` varchar(255) DEFAULT NULL,
  `homeAddressCountry` varchar(255) DEFAULT NULL,
  `phone1Type` enum('','Mobile','Home','Work','Fax','Pager','Other') NOT NULL DEFAULT '',
  `phone1CountryCode` varchar(7) NOT NULL,
  `phone1` varchar(20) NOT NULL,
  `phone2Type` enum('','Mobile','Home','Work','Fax','Pager','Other') NOT NULL DEFAULT '',
  `phone2CountryCode` varchar(7) NOT NULL,
  `phone2` varchar(20) NOT NULL,
  `countryOfBirth` varchar(30) NOT NULL,
  `gibbonSchoolYearIDEntry` int(3) UNSIGNED ZEROFILL NOT NULL,
  `gibbonYearGroupIDEntry` int(3) UNSIGNED ZEROFILL NOT NULL,
  `dayType` varchar(255) DEFAULT NULL,
  `referenceEmail` varchar(100) DEFAULT NULL,
  `schoolName1` varchar(50) NOT NULL,
  `schoolAddress1` varchar(255) NOT NULL,
  `schoolGrades1` varchar(20) NOT NULL,
  `schoolLanguage1` varchar(50) NOT NULL,
  `schoolDate1` date DEFAULT NULL,
  `schoolName2` varchar(50) NOT NULL,
  `schoolAddress2` varchar(255) NOT NULL,
  `schoolGrades2` varchar(20) NOT NULL,
  `schoolLanguage2` varchar(50) NOT NULL,
  `schoolDate2` date DEFAULT NULL,
  `gibbonFamilyID` int(7) UNSIGNED ZEROFILL DEFAULT NULL,
  `siblingName1` varchar(50) NOT NULL,
  `siblingDOB1` date DEFAULT NULL,
  `siblingSchool1` varchar(50) NOT NULL,
  `siblingSchoolJoiningDate1` date DEFAULT NULL,
  `siblingName2` varchar(50) NOT NULL,
  `siblingDOB2` date DEFAULT NULL,
  `siblingSchool2` varchar(50) NOT NULL,
  `siblingSchoolJoiningDate2` date DEFAULT NULL,
  `siblingName3` varchar(50) NOT NULL,
  `siblingDOB3` date DEFAULT NULL,
  `siblingSchool3` varchar(50) NOT NULL,
  `siblingSchoolJoiningDate3` date DEFAULT NULL,
  `languageHomePrimary` varchar(30) NOT NULL,
  `languageHomeSecondary` varchar(30) NOT NULL,
  `languageFirst` varchar(30) NOT NULL,
  `languageSecond` varchar(30) NOT NULL,
  `languageThird` varchar(30) NOT NULL,
  `medicalInformation` text NOT NULL,
  `sen` enum('N','Y') DEFAULT NULL,
  `senDetails` text NOT NULL,
  `languageChoice` varchar(100) DEFAULT NULL,
  `languageChoiceExperience` text,
  `scholarshipInterest` enum('N','Y') NOT NULL DEFAULT 'N',
  `scholarshipRequired` enum('N','Y') NOT NULL DEFAULT 'N',
  `payment` enum('Family','Company') NOT NULL DEFAULT 'Family',
  `companyName` varchar(100) DEFAULT NULL,
  `companyContact` varchar(100) DEFAULT NULL,
  `companyAddress` varchar(255) DEFAULT NULL,
  `companyEmail` text,
  `companyCCFamily` enum('N','Y') DEFAULT NULL COMMENT 'When company is billed, should family receive a copy?',
  `companyPhone` varchar(20) DEFAULT NULL,
  `companyAll` enum('Y','N') DEFAULT NULL,
  `gibbonFinanceFeeCategoryIDList` text,
  `agreement` enum('N','Y') DEFAULT NULL,
  `parent1gibbonPersonID` int(10) UNSIGNED ZEROFILL DEFAULT NULL,
  `parent1title` varchar(5) DEFAULT NULL,
  `parent1surname` varchar(60) DEFAULT '',
  `parent1firstName` varchar(60) DEFAULT '',
  `parent1preferredName` varchar(60) DEFAULT '',
  `parent1officialName` varchar(150) DEFAULT NULL,
  `parent1nameInCharacters` varchar(20) DEFAULT NULL,
  `parent1gender` enum('M','F','Other','Unspecified') DEFAULT 'Unspecified',
  `parent1relationship` varchar(50) DEFAULT NULL,
  `parent1languageFirst` varchar(30) DEFAULT NULL,
  `parent1languageSecond` varchar(30) DEFAULT NULL,
  `parent1email` varchar(75) DEFAULT NULL,
  `parent1phone1Type` enum('','Mobile','Home','Work','Fax','Pager','Other') DEFAULT '',
  `parent1phone1CountryCode` varchar(7) DEFAULT NULL,
  `parent1phone1` varchar(20) DEFAULT NULL,
  `parent1phone2Type` enum('','Mobile','Home','Work','Fax','Pager','Other') DEFAULT '',
  `parent1phone2CountryCode` varchar(7) DEFAULT NULL,
  `parent1phone2` varchar(20) DEFAULT NULL,
  `parent1profession` varchar(30) DEFAULT NULL,
  `parent1employer` varchar(30) DEFAULT NULL,
  `parent2title` varchar(5) DEFAULT NULL,
  `parent2surname` varchar(60) DEFAULT '',
  `parent2firstName` varchar(60) DEFAULT '',
  `parent2preferredName` varchar(60) DEFAULT '',
  `parent2officialName` varchar(150) DEFAULT NULL,
  `parent2nameInCharacters` varchar(20) DEFAULT NULL,
  `parent2gender` enum('M','F','Other','Unspecified') DEFAULT 'Unspecified',
  `parent2relationship` varchar(50) DEFAULT NULL,
  `parent2languageFirst` varchar(30) DEFAULT NULL,
  `parent2languageSecond` varchar(30) DEFAULT NULL,
  `parent2email` varchar(75) DEFAULT NULL,
  `parent2phone1Type` enum('','Mobile','Home','Work','Fax','Pager','Other') DEFAULT '',
  `parent2phone1CountryCode` varchar(7) DEFAULT NULL,
  `parent2phone1` varchar(20) DEFAULT NULL,
  `parent2phone2Type` enum('','Mobile','Home','Work','Fax','Pager','Other') DEFAULT '',
  `parent2phone2CountryCode` varchar(7) DEFAULT NULL,
  `parent2phone2` varchar(20) DEFAULT NULL,
  `parent2profession` varchar(30) DEFAULT NULL,
  `parent2employer` varchar(30) DEFAULT NULL,
  `timestamp` timestamp NULL DEFAULT NULL,
  `priority` int(1) NOT NULL DEFAULT '0',
  `milestones` text NOT NULL,
  `notes` text NOT NULL,
  `dateStart` date DEFAULT NULL,
  `gibbonFormGroupID` int(5) UNSIGNED ZEROFILL DEFAULT NULL,
  `howDidYouHear` varchar(255) DEFAULT NULL,
  `howDidYouHearMore` varchar(255) DEFAULT NULL,
  `paymentMade` enum('N','Y','Exemption') NOT NULL DEFAULT 'N',
  `paymentMade2` enum('N','Y','Exemption') NOT NULL DEFAULT 'N',
  `gibbonPaymentID` int(14) UNSIGNED ZEROFILL DEFAULT NULL,
  `gibbonPaymentID2` int(14) UNSIGNED ZEROFILL DEFAULT NULL,
  `studentID` varchar(10) DEFAULT NULL,
  `privacy` text,
  `fields` text NOT NULL COMMENT 'Serialised array of custom field values',
  `parent1fields` text NOT NULL COMMENT 'Serialised array of custom field values',
  `parent2fields` text NOT NULL COMMENT 'Serialised array of custom field values'
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonApplicationForm`
  ADD PRIMARY KEY (`gibbonApplicationFormID`);

-- [TABLE] gibbonApplicationFormFile | [TRIAGE] PLUGIN | [CATEGORY] Finance, Admissions, Library & Comms
CREATE TABLE `gibbonApplicationFormFile` (
  `gibbonApplicationFormFileID` int(14) UNSIGNED ZEROFILL NOT NULL,
  `gibbonApplicationFormID` int(12) UNSIGNED ZEROFILL NOT NULL,
  `name` varchar(255) NOT NULL,
  `path` varchar(255) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonApplicationFormFile`
  ADD PRIMARY KEY (`gibbonApplicationFormFileID`);

-- [TABLE] gibbonApplicationFormLink | [TRIAGE] PLUGIN | [CATEGORY] Finance, Admissions, Library & Comms
CREATE TABLE `gibbonApplicationFormLink` (
  `gibbonApplicationFormLinkID` int(12) UNSIGNED NOT NULL,
  `gibbonApplicationFormID1` int(12) UNSIGNED ZEROFILL NOT NULL,
  `gibbonApplicationFormID2` int(12) UNSIGNED ZEROFILL NOT NULL,
  `timestamp` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonApplicationFormLink`
  ADD PRIMARY KEY (`gibbonApplicationFormLinkID`),
  ADD UNIQUE KEY `link` (`gibbonApplicationFormID1`,`gibbonApplicationFormID2`);

-- [TABLE] gibbonApplicationFormRelationship | [TRIAGE] PLUGIN | [CATEGORY] Finance, Admissions, Library & Comms
CREATE TABLE `gibbonApplicationFormRelationship` (
  `gibbonApplicationFormRelationshipID` int(14) UNSIGNED ZEROFILL NOT NULL,
  `gibbonApplicationFormID` int(12) UNSIGNED ZEROFILL NOT NULL,
  `gibbonPersonID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `relationship` varchar(50) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonApplicationFormRelationship`
  ADD PRIMARY KEY (`gibbonApplicationFormRelationshipID`);

-- [TABLE] gibbonEmailTemplate | [TRIAGE] PLUGIN | [CATEGORY] Finance, Admissions, Library & Comms
CREATE TABLE `gibbonEmailTemplate` (
  `gibbonEmailTemplateID` int(6) UNSIGNED ZEROFILL NOT NULL,
  `type` enum('Core','Additional','Custom') NOT NULL DEFAULT 'Core',
  `templateType` varchar(120) NOT NULL,
  `moduleName` varchar(30) NOT NULL,
  `templateName` varchar(120) NOT NULL,
  `templateSubject` varchar(255) DEFAULT NULL,
  `templateBody` text,
  `variables` text,
  `timestamp` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonEmailTemplate`
  ADD PRIMARY KEY (`gibbonEmailTemplateID`),
  ADD UNIQUE KEY `moduleTemplate` (`templateName`,`moduleName`) USING BTREE;

-- [TABLE] gibbonFinanceBillingSchedule | [TRIAGE] PLUGIN | [CATEGORY] Finance, Admissions, Library & Comms
CREATE TABLE `gibbonFinanceBillingSchedule` (
  `gibbonFinanceBillingScheduleID` int(6) UNSIGNED ZEROFILL NOT NULL,
  `gibbonSchoolYearID` int(3) UNSIGNED ZEROFILL NOT NULL,
  `name` varchar(100) NOT NULL,
  `description` text NOT NULL,
  `active` enum('Y','N') NOT NULL DEFAULT 'Y',
  `invoiceIssueDate` date DEFAULT NULL,
  `invoiceDueDate` date DEFAULT NULL,
  `gibbonPersonIDCreator` int(10) UNSIGNED ZEROFILL NOT NULL,
  `timestampCreator` timestamp NULL DEFAULT NULL,
  `gibbonPersonIDUpdate` int(10) UNSIGNED ZEROFILL DEFAULT NULL,
  `timestampUpdate` timestamp NULL DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonFinanceBillingSchedule`
  ADD PRIMARY KEY (`gibbonFinanceBillingScheduleID`);

-- [TABLE] gibbonFinanceBudget | [TRIAGE] PLUGIN | [CATEGORY] Finance, Admissions, Library & Comms
CREATE TABLE `gibbonFinanceBudget` (
  `gibbonFinanceBudgetID` int(4) UNSIGNED ZEROFILL NOT NULL,
  `name` varchar(30) NOT NULL,
  `nameShort` varchar(8) NOT NULL,
  `active` enum('Y','N') NOT NULL DEFAULT 'Y',
  `category` varchar(255) NOT NULL,
  `gibbonPersonIDCreator` int(10) UNSIGNED ZEROFILL NOT NULL,
  `timestampCreator` timestamp NULL DEFAULT NULL,
  `gibbonPersonIDUpdate` int(10) UNSIGNED ZEROFILL DEFAULT NULL,
  `timestampUpdate` timestamp NULL DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonFinanceBudget`
  ADD PRIMARY KEY (`gibbonFinanceBudgetID`),
  ADD UNIQUE KEY `name` (`name`),
  ADD UNIQUE KEY `nameShort` (`nameShort`);

-- [TABLE] gibbonFinanceBudgetCycle | [TRIAGE] PLUGIN | [CATEGORY] Finance, Admissions, Library & Comms
CREATE TABLE `gibbonFinanceBudgetCycle` (
  `gibbonFinanceBudgetCycleID` int(6) UNSIGNED ZEROFILL NOT NULL,
  `name` varchar(7) NOT NULL,
  `status` enum('Past','Current','Upcoming') NOT NULL DEFAULT 'Upcoming',
  `dateStart` date NOT NULL,
  `dateEnd` date NOT NULL,
  `sequenceNumber` int(6) NOT NULL,
  `gibbonPersonIDCreator` int(10) UNSIGNED ZEROFILL NOT NULL,
  `timestampCreator` timestamp NULL DEFAULT NULL,
  `gibbonPersonIDUpdate` int(10) UNSIGNED ZEROFILL DEFAULT NULL,
  `timestampUpdate` timestamp NULL DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonFinanceBudgetCycle`
  ADD PRIMARY KEY (`gibbonFinanceBudgetCycleID`),
  ADD UNIQUE KEY `name` (`name`);

-- [TABLE] gibbonFinanceBudgetCycleAllocation | [TRIAGE] PLUGIN | [CATEGORY] Finance, Admissions, Library & Comms
CREATE TABLE `gibbonFinanceBudgetCycleAllocation` (
  `gibbonFinanceBudgetCycleAllocationID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `gibbonFinanceBudgetID` int(4) UNSIGNED ZEROFILL NOT NULL,
  `gibbonFinanceBudgetCycleID` int(6) UNSIGNED ZEROFILL NOT NULL,
  `value` decimal(14,2) NOT NULL DEFAULT '0.00'
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonFinanceBudgetCycleAllocation`
  ADD PRIMARY KEY (`gibbonFinanceBudgetCycleAllocationID`);

-- [TABLE] gibbonFinanceBudgetPerson | [TRIAGE] PLUGIN | [CATEGORY] Finance, Admissions, Library & Comms
CREATE TABLE `gibbonFinanceBudgetPerson` (
  `gibbonFinanceBudgetPersonID` int(8) UNSIGNED ZEROFILL NOT NULL,
  `gibbonFinanceBudgetID` int(4) UNSIGNED ZEROFILL NOT NULL,
  `gibbonPersonID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `access` enum('Full','Write','Read') NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonFinanceBudgetPerson`
  ADD PRIMARY KEY (`gibbonFinanceBudgetPersonID`);

-- [TABLE] gibbonFinanceExpense | [TRIAGE] PLUGIN | [CATEGORY] Finance, Admissions, Library & Comms
CREATE TABLE `gibbonFinanceExpense` (
  `gibbonFinanceExpenseID` int(14) UNSIGNED ZEROFILL NOT NULL,
  `gibbonFinanceBudgetID` int(4) UNSIGNED ZEROFILL NOT NULL,
  `gibbonFinanceBudgetCycleID` int(6) UNSIGNED ZEROFILL NOT NULL,
  `title` varchar(60) NOT NULL,
  `body` text NOT NULL,
  `status` enum('Requested','Approved','Rejected','Cancelled','Ordered','Paid') NOT NULL,
  `cost` decimal(12,2) NOT NULL,
  `countAgainstBudget` enum('Y','N') NOT NULL DEFAULT 'Y',
  `purchaseBy` enum('School','Self') NOT NULL DEFAULT 'School',
  `purchaseDetails` text NOT NULL,
  `paymentMethod` enum('Cash','Cheque','Credit Card','Bank Transfer','Other') DEFAULT NULL,
  `paymentDate` date DEFAULT NULL,
  `paymentAmount` decimal(12,2) DEFAULT NULL,
  `gibbonPersonIDPayment` int(10) UNSIGNED ZEROFILL DEFAULT NULL,
  `paymentID` varchar(100) DEFAULT NULL,
  `paymentReimbursementReceipt` varchar(255) NOT NULL,
  `paymentReimbursementStatus` enum('Requested','Complete') DEFAULT NULL,
  `gibbonPersonIDCreator` int(10) UNSIGNED ZEROFILL NOT NULL,
  `timestampCreator` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `statusApprovalBudgetCleared` enum('N','Y') NOT NULL DEFAULT 'N'
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonFinanceExpense`
  ADD PRIMARY KEY (`gibbonFinanceExpenseID`);

-- [TABLE] gibbonFinanceExpenseApprover | [TRIAGE] PLUGIN | [CATEGORY] Finance, Admissions, Library & Comms
CREATE TABLE `gibbonFinanceExpenseApprover` (
  `gibbonFinanceExpenseApproverID` int(4) UNSIGNED ZEROFILL NOT NULL,
  `gibbonPersonID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `sequenceNumber` int(4) DEFAULT NULL,
  `gibbonPersonIDCreator` int(10) UNSIGNED ZEROFILL NOT NULL,
  `timestampCreator` timestamp NULL DEFAULT NULL,
  `gibbonPersonIDUpdate` int(10) UNSIGNED ZEROFILL DEFAULT NULL,
  `timestampUpdate` timestamp NULL DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonFinanceExpenseApprover`
  ADD PRIMARY KEY (`gibbonFinanceExpenseApproverID`);

-- [TABLE] gibbonFinanceExpenseLog | [TRIAGE] PLUGIN | [CATEGORY] Finance, Admissions, Library & Comms
CREATE TABLE `gibbonFinanceExpenseLog` (
  `gibbonFinanceExpenseLogID` int(16) UNSIGNED ZEROFILL NOT NULL,
  `gibbonFinanceExpenseID` int(14) UNSIGNED ZEROFILL NOT NULL,
  `gibbonPersonID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `timestamp` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `action` enum('Request','Approval - Partial - Budget','Approval - Partial - School','Approval - Final','Approval - Exempt','Rejection','Cancellation','Order','Payment','Reimbursement Request','Reimbursement Completion','Comment') NOT NULL,
  `comment` text NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonFinanceExpenseLog`
  ADD PRIMARY KEY (`gibbonFinanceExpenseLogID`);

-- [TABLE] gibbonFinanceFee | [TRIAGE] PLUGIN | [CATEGORY] Finance, Admissions, Library & Comms
CREATE TABLE `gibbonFinanceFee` (
  `gibbonFinanceFeeID` int(6) UNSIGNED ZEROFILL NOT NULL,
  `gibbonSchoolYearID` int(3) UNSIGNED ZEROFILL NOT NULL,
  `name` varchar(100) NOT NULL,
  `nameShort` varchar(6) NOT NULL,
  `description` text NOT NULL,
  `active` enum('Y','N') NOT NULL DEFAULT 'Y',
  `gibbonFinanceFeeCategoryID` int(4) UNSIGNED ZEROFILL NOT NULL,
  `fee` decimal(12,2) NOT NULL,
  `gibbonPersonIDCreator` int(10) UNSIGNED ZEROFILL NOT NULL,
  `timestampCreator` timestamp NULL DEFAULT NULL,
  `gibbonPersonIDUpdate` int(10) UNSIGNED ZEROFILL DEFAULT NULL,
  `timestampUpdate` timestamp NULL DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonFinanceFee`
  ADD PRIMARY KEY (`gibbonFinanceFeeID`);

-- [TABLE] gibbonFinanceFeeCategory | [TRIAGE] PLUGIN | [CATEGORY] Finance, Admissions, Library & Comms
CREATE TABLE `gibbonFinanceFeeCategory` (
  `gibbonFinanceFeeCategoryID` int(4) UNSIGNED ZEROFILL NOT NULL,
  `name` varchar(100) NOT NULL,
  `nameShort` varchar(6) NOT NULL,
  `description` text NOT NULL,
  `active` enum('Y','N') NOT NULL,
  `gibbonPersonIDCreator` int(10) UNSIGNED ZEROFILL NOT NULL,
  `timestampCreator` timestamp NULL DEFAULT NULL,
  `gibbonPersonIDUpdate` int(10) UNSIGNED ZEROFILL DEFAULT NULL,
  `timestampUpdate` timestamp NULL DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonFinanceFeeCategory`
  ADD PRIMARY KEY (`gibbonFinanceFeeCategoryID`);

-- [TABLE] gibbonFinanceInvoice | [TRIAGE] PLUGIN | [CATEGORY] Finance, Admissions, Library & Comms
CREATE TABLE `gibbonFinanceInvoice` (
  `gibbonFinanceInvoiceID` int(14) UNSIGNED ZEROFILL NOT NULL,
  `gibbonSchoolYearID` int(3) UNSIGNED ZEROFILL NOT NULL,
  `gibbonFinanceInvoiceeID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `invoiceTo` enum('Family','Company') NOT NULL DEFAULT 'Family',
  `billingScheduleType` enum('Scheduled','Ad Hoc') NOT NULL DEFAULT 'Ad Hoc',
  `separated` enum('N','Y') DEFAULT NULL COMMENT 'Has this invoice been separated from its schedule in gibbonFinanceBillingSchedule? Only applies to scheduled invoices. Separation takes place during invoice issueing.',
  `gibbonFinanceBillingScheduleID` int(6) UNSIGNED ZEROFILL DEFAULT NULL,
  `status` enum('Pending','Issued','Paid','Paid - Partial','Cancelled','Refunded') NOT NULL DEFAULT 'Pending',
  `gibbonFinanceFeeCategoryIDList` text,
  `invoiceIssueDate` date DEFAULT NULL,
  `invoiceDueDate` date DEFAULT NULL,
  `paidDate` date DEFAULT NULL,
  `paidAmount` decimal(13,2) DEFAULT NULL COMMENT 'The current running total amount paid to this invoice',
  `gibbonPaymentID` int(14) UNSIGNED ZEROFILL DEFAULT NULL,
  `reminderCount` int(3) NOT NULL DEFAULT '0',
  `notes` text NOT NULL,
  `key` varchar(40) NOT NULL,
  `gibbonPersonIDCreator` int(10) UNSIGNED ZEROFILL NOT NULL,
  `timestampCreator` timestamp NULL DEFAULT NULL,
  `gibbonPersonIDUpdate` int(10) UNSIGNED ZEROFILL DEFAULT NULL,
  `timestampUpdate` timestamp NULL DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonFinanceInvoice`
  ADD PRIMARY KEY (`gibbonFinanceInvoiceID`);

-- [TABLE] gibbonFinanceInvoiceFee | [TRIAGE] PLUGIN | [CATEGORY] Finance, Admissions, Library & Comms
CREATE TABLE `gibbonFinanceInvoiceFee` (
  `gibbonFinanceInvoiceFeeID` int(15) UNSIGNED ZEROFILL NOT NULL,
  `gibbonFinanceInvoiceID` int(14) UNSIGNED ZEROFILL NOT NULL,
  `feeType` enum('Standard','Ad Hoc') NOT NULL DEFAULT 'Ad Hoc',
  `gibbonFinanceFeeID` int(6) UNSIGNED ZEROFILL DEFAULT NULL,
  `separated` enum('N','Y') DEFAULT NULL COMMENT 'Has this fee been separated from its parent in gibbonFinanceFee? Only applies to Standard fees. Separation takes place during invoice issueing.',
  `name` varchar(100) DEFAULT NULL,
  `description` text,
  `gibbonFinanceFeeCategoryID` int(4) UNSIGNED ZEROFILL DEFAULT NULL,
  `fee` decimal(12,2) DEFAULT NULL,
  `sequenceNumber` int(10) DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonFinanceInvoiceFee`
  ADD PRIMARY KEY (`gibbonFinanceInvoiceFeeID`);

-- [TABLE] gibbonFinanceInvoicee | [TRIAGE] PLUGIN | [CATEGORY] Finance, Admissions, Library & Comms
CREATE TABLE `gibbonFinanceInvoicee` (
  `gibbonFinanceInvoiceeID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `gibbonPersonID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `invoiceTo` enum('Family','Company') NOT NULL,
  `companyName` varchar(100) DEFAULT NULL,
  `companyContact` varchar(100) DEFAULT NULL,
  `companyAddress` varchar(255) DEFAULT NULL,
  `companyEmail` text,
  `companyCCFamily` enum('N','Y') DEFAULT NULL COMMENT 'When company is billed, should family receive a copy?',
  `companyPhone` varchar(20) DEFAULT NULL,
  `companyAll` enum('Y','N') DEFAULT NULL COMMENT 'Should company pay all invoices?.',
  `gibbonFinanceFeeCategoryIDList` text COMMENT 'If companyAll is N, list category IDs for campany to pay here.'
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonFinanceInvoicee`
  ADD PRIMARY KEY (`gibbonFinanceInvoiceeID`);

-- [TABLE] gibbonFinanceInvoiceeUpdate | [TRIAGE] PLUGIN | [CATEGORY] Finance, Admissions, Library & Comms
CREATE TABLE `gibbonFinanceInvoiceeUpdate` (
  `gibbonFinanceInvoiceeUpdateID` int(12) UNSIGNED ZEROFILL NOT NULL,
  `gibbonSchoolYearID` int(3) UNSIGNED ZEROFILL DEFAULT NULL,
  `status` enum('Pending','Complete') NOT NULL DEFAULT 'Pending',
  `gibbonFinanceInvoiceeID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `invoiceTo` enum('Family','Company') NOT NULL,
  `companyName` varchar(100) DEFAULT NULL,
  `companyContact` varchar(100) DEFAULT NULL,
  `companyAddress` varchar(255) DEFAULT NULL,
  `companyEmail` text,
  `companyCCFamily` enum('N','Y') DEFAULT NULL COMMENT 'When company is billed, should family receive a copy?',
  `companyPhone` varchar(20) DEFAULT NULL,
  `companyAll` enum('Y','N') DEFAULT NULL COMMENT 'Should company pay all invoices?.',
  `gibbonFinanceFeeCategoryIDList` text COMMENT 'If companyAll is N, list category IDs for campany to pay here.',
  `gibbonPersonIDUpdater` int(10) UNSIGNED ZEROFILL NOT NULL,
  `timestamp` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonFinanceInvoiceeUpdate`
  ADD PRIMARY KEY (`gibbonFinanceInvoiceeUpdateID`),
  ADD KEY `gibbonInvoiceeIndex` (`gibbonFinanceInvoiceeID`,`gibbonSchoolYearID`);

-- [TABLE] gibbonForm | [TRIAGE] PLUGIN | [CATEGORY] Finance, Admissions, Library & Comms
CREATE TABLE `gibbonForm` (
  `gibbonFormID` int(6) UNSIGNED ZEROFILL NOT NULL,
  `name` varchar(90) NOT NULL,
  `description` varchar(255) DEFAULT NULL,
  `type` varchar(60) NOT NULL,
  `active` enum('Y','N') NOT NULL DEFAULT 'Y',
  `public` enum('Y','N') NOT NULL DEFAULT 'N',
  `gibbonYearGroupIDList` varchar(255) DEFAULT NULL,
  `config` text
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonForm`
  ADD PRIMARY KEY (`gibbonFormID`);

-- [TABLE] gibbonFormField | [TRIAGE] PLUGIN | [CATEGORY] Finance, Admissions, Library & Comms
CREATE TABLE `gibbonFormField` (
  `gibbonFormFieldID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `gibbonFormPageID` int(8) UNSIGNED ZEROFILL NOT NULL,
  `fieldName` varchar(90) NOT NULL,
  `fieldType` varchar(60) NOT NULL,
  `fieldGroup` varchar(60) NOT NULL,
  `required` enum('Y','N','X') NOT NULL DEFAULT 'N',
  `hidden` enum('Y','N') NOT NULL DEFAULT 'N',
  `prefill` enum('Y','N') NOT NULL DEFAULT 'N',
  `options` text,
  `label` varchar(90) DEFAULT NULL,
  `description` text,
  `conditional` text COMMENT 'json',
  `defaultValue` varchar(255) DEFAULT NULL,
  `sequenceNumber` int(3) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonFormField`
  ADD PRIMARY KEY (`gibbonFormFieldID`);

-- [TABLE] gibbonFormPage | [TRIAGE] PLUGIN | [CATEGORY] Finance, Admissions, Library & Comms
CREATE TABLE `gibbonFormPage` (
  `gibbonFormPageID` int(8) UNSIGNED ZEROFILL NOT NULL,
  `gibbonFormID` int(6) UNSIGNED ZEROFILL NOT NULL,
  `name` varchar(90) NOT NULL,
  `introduction` text,
  `postscript` text,
  `sequenceNumber` int(3) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonFormPage`
  ADD PRIMARY KEY (`gibbonFormPageID`);

-- [TABLE] gibbonFormSubmission | [TRIAGE] PLUGIN | [CATEGORY] Finance, Admissions, Library & Comms
CREATE TABLE `gibbonFormSubmission` (
  `gibbonFormSubmissionID` int(12) UNSIGNED ZEROFILL NOT NULL,
  `gibbonFormID` int(6) UNSIGNED ZEROFILL DEFAULT NULL,
  `gibbonFormPageID` int(8) UNSIGNED ZEROFILL DEFAULT NULL,
  `foreignTable` varchar(60) DEFAULT NULL,
  `foreignTableID` int(14) UNSIGNED ZEROFILL DEFAULT NULL,
  `status` enum('Incomplete','Pending','Processed','Withdrawn') NOT NULL DEFAULT 'Incomplete',
  `owner` varchar(75) DEFAULT NULL,
  `identifier` varchar(255) DEFAULT NULL,
  `data` mediumtext,
  `result` text,
  `timestampCreated` timestamp NULL DEFAULT NULL,
  `timestampModified` timestamp NULL DEFAULT NULL,
  `gibbonPersonIDModified` int(10) UNSIGNED ZEROFILL DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonFormSubmission`
  ADD PRIMARY KEY (`gibbonFormSubmissionID`),
  ADD KEY `foreignTable` (`foreignTable`,`foreignTableID`);

-- [TABLE] gibbonFormUpload | [TRIAGE] PLUGIN | [CATEGORY] Finance, Admissions, Library & Comms
CREATE TABLE `gibbonFormUpload` (
  `gibbonFormUploadID` int(12) UNSIGNED ZEROFILL NOT NULL,
  `gibbonFormID` int(6) UNSIGNED ZEROFILL DEFAULT NULL,
  `gibbonFormFieldID` int(10) UNSIGNED ZEROFILL DEFAULT NULL,
  `foreignTable` varchar(60) DEFAULT NULL,
  `foreignTableID` int(14) UNSIGNED ZEROFILL DEFAULT NULL,
  `name` varchar(255) DEFAULT NULL,
  `path` varchar(255) DEFAULT NULL,
  `confirmed` enum('Y','N') NOT NULL DEFAULT 'N',
  `timestamp` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonFormUpload`
  ADD PRIMARY KEY (`gibbonFormUploadID`),
  ADD KEY `foreignTable` (`foreignTable`,`foreignTableID`);

-- [TABLE] gibbonLibraryItem | [TRIAGE] PLUGIN | [CATEGORY] Finance, Admissions, Library & Comms
CREATE TABLE `gibbonLibraryItem` (
  `gibbonLibraryItemID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `gibbonLibraryItemIDParent` int(10) DEFAULT NULL,
  `gibbonLibraryTypeID` int(5) UNSIGNED ZEROFILL NOT NULL,
  `id` varchar(255) NOT NULL,
  `name` varchar(255) NOT NULL COMMENT 'Name for book, model for computer, etc.',
  `producer` varchar(255) NOT NULL COMMENT 'Author for book, manufacturer for computer, etc',
  `fields` text NOT NULL COMMENT 'JSON object',
  `vendor` varchar(100) NOT NULL,
  `purchaseDate` date DEFAULT NULL,
  `invoiceNumber` varchar(50) NOT NULL,
  `cost` decimal(10,2) DEFAULT NULL,
  `imageType` enum('','Link','File') NOT NULL DEFAULT '' COMMENT 'Type of image. Image should be 240px x 240px, or smaller.',
  `imageLocation` varchar(255) NOT NULL COMMENT 'URL or local FS path of image.',
  `comment` text NOT NULL,
  `gibbonSpaceID` int(5) UNSIGNED ZEROFILL DEFAULT NULL,
  `locationDetail` varchar(255) NOT NULL,
  `ownershipType` enum('School','Individual') NOT NULL DEFAULT 'School',
  `gibbonPersonIDOwnership` int(10) UNSIGNED ZEROFILL DEFAULT NULL COMMENT 'If owned by school, then this is the main user. If owned by individual, then this is that individual.',
  `gibbonDepartmentID` int(4) UNSIGNED ZEROFILL DEFAULT NULL COMMENT 'Who is responsible for managing this item? By default this will be the person who added the record, but it can be changed.',
  `replacement` enum('Y','N') NOT NULL DEFAULT 'Y',
  `replacementCost` decimal(10,2) DEFAULT NULL,
  `gibbonSchoolYearIDReplacement` int(3) UNSIGNED ZEROFILL DEFAULT NULL,
  `physicalCondition` enum('','As New','Lightly Worn','Moderately Worn','Damaged','Unusable') NOT NULL,
  `bookable` enum('N','Y') NOT NULL DEFAULT 'N',
  `borrowable` enum('Y','N') NOT NULL DEFAULT 'Y',
  `status` enum('Available','In Use','Decommissioned','Lost','On Loan','Repair','Reserved','On Order') NOT NULL DEFAULT 'Available' COMMENT 'The current status of the item.',
  `gibbonPersonIDStatusResponsible` int(10) UNSIGNED ZEROFILL DEFAULT NULL COMMENT 'The person who is responsible for the current status.',
  `gibbonPersonIDStatusRecorder` int(10) UNSIGNED ZEROFILL DEFAULT NULL COMMENT 'The person who recored the current status.',
  `timestampStatus` datetime DEFAULT NULL COMMENT 'The time the status was recorded',
  `returnExpected` date DEFAULT NULL COMMENT 'The time when the event expires.',
  `returnAction` enum('Make Available','Decommission','Repair','Reserve') DEFAULT NULL COMMENT 'What to do when the item is returned?',
  `gibbonPersonIDReturnAction` int(10) UNSIGNED ZEROFILL DEFAULT NULL,
  `gibbonPersonIDCreator` int(10) UNSIGNED ZEROFILL NOT NULL,
  `timestampCreator` datetime NOT NULL,
  `gibbonPersonIDUpdate` int(10) UNSIGNED ZEROFILL DEFAULT NULL,
  `timestampUpdate` datetime DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonLibraryItem`
  ADD PRIMARY KEY (`gibbonLibraryItemID`),
  ADD UNIQUE KEY `id` (`id`);

-- [TABLE] gibbonLibraryItemEvent | [TRIAGE] PLUGIN | [CATEGORY] Finance, Admissions, Library & Comms
CREATE TABLE `gibbonLibraryItemEvent` (
  `gibbonLibraryItemEventID` int(14) UNSIGNED ZEROFILL NOT NULL,
  `gibbonLibraryItemID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `type` enum('Decommission','Loss','Loan','Repair','Reserve','Other') NOT NULL DEFAULT 'Other' COMMENT 'This is maintained even after the item is returned, so we know what type of event it was.',
  `status` enum('Available','Decommissioned','Lost','On Loan','Repair','Reserved','Returned') NOT NULL DEFAULT 'Available',
  `gibbonPersonIDStatusResponsible` int(10) UNSIGNED ZEROFILL DEFAULT NULL COMMENT 'The person who was responsible for the event.',
  `gibbonPersonIDOut` int(10) UNSIGNED ZEROFILL DEFAULT NULL COMMENT 'The person who recored the event.',
  `timestampOut` datetime DEFAULT NULL COMMENT 'The time the event was recorded',
  `returnExpected` date DEFAULT NULL COMMENT 'The time when the event expires.',
  `returnAction` enum('Make Available','Decommission','Repair','Reserve') DEFAULT NULL COMMENT 'What to do when the item is returned?',
  `gibbonPersonIDReturnAction` int(10) UNSIGNED ZEROFILL DEFAULT NULL,
  `timestampReturn` datetime DEFAULT NULL,
  `gibbonPersonIDIn` int(10) UNSIGNED ZEROFILL DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonLibraryItemEvent`
  ADD PRIMARY KEY (`gibbonLibraryItemEventID`);

-- [TABLE] gibbonLibraryShelf | [TRIAGE] PLUGIN | [CATEGORY] Finance, Admissions, Library & Comms
CREATE TABLE `gibbonLibraryShelf` (
  `gibbonLibraryShelfID` int(6) UNSIGNED ZEROFILL NOT NULL,
  `name` varchar(90) NOT NULL,
  `active` enum('Y','N') NOT NULL DEFAULT 'N',
  `shuffle` enum('N','Y') NOT NULL DEFAULT 'N',
  `field` varchar(90) NOT NULL,
  `fieldValue` varchar(90) DEFAULT NULL,
  `type` enum('Manual','Automatic') NOT NULL DEFAULT 'Manual',
  `sequenceNumber` int(6) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonLibraryShelf`
  ADD PRIMARY KEY (`gibbonLibraryShelfID`);

-- [TABLE] gibbonLibraryShelfItem | [TRIAGE] PLUGIN | [CATEGORY] Finance, Admissions, Library & Comms
CREATE TABLE `gibbonLibraryShelfItem` (
  `gibbonLibraryShelfItemID` int(6) UNSIGNED ZEROFILL NOT NULL,
  `gibbonLibraryItemID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `gibbonLibraryShelfID` int(6) UNSIGNED ZEROFILL NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonLibraryShelfItem`
  ADD PRIMARY KEY (`gibbonLibraryShelfItemID`);

-- [TABLE] gibbonLibraryType | [TRIAGE] PLUGIN | [CATEGORY] Finance, Admissions, Library & Comms
CREATE TABLE `gibbonLibraryType` (
  `gibbonLibraryTypeID` int(5) UNSIGNED ZEROFILL NOT NULL,
  `name` varchar(30) NOT NULL,
  `active` enum('Y','N') NOT NULL DEFAULT 'Y',
  `fields` text NOT NULL COMMENT 'JSON object'
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonLibraryType`
  ADD PRIMARY KEY (`gibbonLibraryTypeID`);

-- [TABLE] gibbonMessenger | [TRIAGE] PLUGIN | [CATEGORY] Finance, Admissions, Library & Comms
CREATE TABLE `gibbonMessenger` (
  `gibbonMessengerID` int(12) UNSIGNED ZEROFILL NOT NULL,
  `gibbonSchoolYearID` int(3) UNSIGNED ZEROFILL DEFAULT NULL,
  `status` enum('Draft','Sending','Sent') NOT NULL DEFAULT 'Sent',
  `email` enum('N','Y') NOT NULL DEFAULT 'N',
  `messageWall` enum('N','Y') NOT NULL DEFAULT 'N',
  `messageWallPin` enum('N','Y') NOT NULL DEFAULT 'N',
  `messageWall_dateStart` date DEFAULT NULL,
  `messageWall_dateEnd` date DEFAULT NULL,
  `sms` enum('N','Y') NOT NULL DEFAULT 'N',
  `subject` varchar(60) NOT NULL,
  `body` text NOT NULL,
  `gibbonPersonID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `timestamp` timestamp NULL DEFAULT NULL,
  `emailFrom` varchar(75) DEFAULT NULL,
  `emailReplyTo` varchar(75) DEFAULT NULL,
  `emailReport` text NOT NULL,
  `emailReceipt` enum('N','Y') DEFAULT NULL,
  `emailReceiptText` text,
  `individualNaming` enum('N','Y') NOT NULL DEFAULT 'N',
  `includeSignature` enum('N','Y') NOT NULL DEFAULT 'N',
  `smsReport` text NOT NULL,
  `confidential` enum('N','Y') NOT NULL DEFAULT 'N',
  `enableSharingLink` enum('N','Y') NOT NULL DEFAULT 'N'
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonMessenger`
  ADD PRIMARY KEY (`gibbonMessengerID`);

-- [TABLE] gibbonMessengerCannedResponse | [TRIAGE] PLUGIN | [CATEGORY] Finance, Admissions, Library & Comms
CREATE TABLE `gibbonMessengerCannedResponse` (
  `gibbonMessengerCannedResponseID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `subject` varchar(30) NOT NULL,
  `body` text NOT NULL,
  `timestampCreator` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `gibbonPersonIDCreator` int(10) UNSIGNED ZEROFILL NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonMessengerCannedResponse`
  ADD PRIMARY KEY (`gibbonMessengerCannedResponseID`);

-- [TABLE] gibbonMessengerMailingList | [TRIAGE] PLUGIN | [CATEGORY] Finance, Admissions, Library & Comms
CREATE TABLE `gibbonMessengerMailingList` (
  `gibbonMessengerMailingListID` int(8) UNSIGNED ZEROFILL NOT NULL,
  `name` varchar(60) NOT NULL,
  `active` enum('Y','N') NOT NULL DEFAULT 'Y',
  `timestampCreated` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonMessengerMailingList`
  ADD PRIMARY KEY (`gibbonMessengerMailingListID`);

-- [TABLE] gibbonMessengerMailingListRecipient | [TRIAGE] PLUGIN | [CATEGORY] Finance, Admissions, Library & Comms
CREATE TABLE `gibbonMessengerMailingListRecipient` (
  `gibbonMessengerMailingListRecipientID` int(8) UNSIGNED ZEROFILL NOT NULL,
  `surname` varchar(60) DEFAULT NULL,
  `preferredName` varchar(60) DEFAULT NULL,
  `email` varchar(75) NOT NULL,
  `organisation` varchar(60) DEFAULT NULL,
  `gibbonMessengerMailingListIDList` text,
  `key` varchar(40) DEFAULT NULL,
  `timestampCreated` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonMessengerMailingListRecipient`
  ADD PRIMARY KEY (`gibbonMessengerMailingListRecipientID`),
  ADD UNIQUE KEY `email` (`email`);

-- [TABLE] gibbonMessengerReceipt | [TRIAGE] PLUGIN | [CATEGORY] Finance, Admissions, Library & Comms
CREATE TABLE `gibbonMessengerReceipt` (
  `gibbonMessengerReceiptID` int(14) UNSIGNED ZEROFILL NOT NULL,
  `gibbonMessengerID` int(12) UNSIGNED ZEROFILL NOT NULL,
  `gibbonPersonID` int(10) UNSIGNED ZEROFILL DEFAULT NULL,
  `sent` enum('N','Y') NOT NULL DEFAULT 'N',
  `targetType` enum('Class','Course','Form Group','Year Group','Activity','Role','Applicants','Individuals','Houses','Role Category','Transport','Attendance','Group','Mailing List') DEFAULT NULL,
  `targetID` varchar(30) NOT NULL,
  `contactType` enum('Email','SMS') DEFAULT NULL,
  `contactDetail` varchar(255) DEFAULT NULL,
  `key` varchar(40) DEFAULT NULL,
  `confirmed` enum('N','Y') DEFAULT NULL,
  `confirmedTimestamp` timestamp NULL DEFAULT NULL,
  `gibbonPersonIDListStudent` text,
  `nameListStudent` text,
  `unsubscribeKey` varchar(50) DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonMessengerReceipt`
  ADD PRIMARY KEY (`gibbonMessengerReceiptID`);

-- [TABLE] gibbonMessengerTarget | [TRIAGE] PLUGIN | [CATEGORY] Finance, Admissions, Library & Comms
CREATE TABLE `gibbonMessengerTarget` (
  `gibbonMessengerTargetID` int(14) UNSIGNED ZEROFILL NOT NULL,
  `gibbonMessengerID` int(12) UNSIGNED ZEROFILL NOT NULL,
  `type` enum('Class','Course','Form Group','Year Group','Activity','Role','Applicants','Individuals','Houses','Role Category','Transport','Attendance','Group','Mailing List') DEFAULT NULL,
  `id` varchar(30) NOT NULL,
  `parents` enum('N','Y') NOT NULL DEFAULT 'N',
  `students` enum('N','Y') NOT NULL DEFAULT 'N',
  `staff` enum('N','Y') NOT NULL DEFAULT 'N'
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonMessengerTarget`
  ADD PRIMARY KEY (`gibbonMessengerTargetID`),
  ADD KEY `gibbonMessengerID` (`gibbonMessengerID`,`gibbonMessengerTargetID`);

-- [TABLE] gibbonPayment | [TRIAGE] PLUGIN | [CATEGORY] Finance, Admissions, Library & Comms
CREATE TABLE `gibbonPayment` (
  `gibbonPaymentID` int(14) UNSIGNED ZEROFILL NOT NULL,
  `foreignTable` varchar(50) NOT NULL,
  `foreignTableID` int(14) UNSIGNED ZEROFILL NOT NULL,
  `gibbonPersonID` int(10) UNSIGNED ZEROFILL DEFAULT NULL COMMENT 'Person recording the transaction',
  `type` varchar(60) NOT NULL DEFAULT 'Online',
  `status` enum('Complete','Partial','Final','Failure') NOT NULL DEFAULT 'Complete' COMMENT 'Complete means paid in one go, partial is part of a set of payments, and final is last in a set of payments.',
  `amount` decimal(13,2) NOT NULL,
  `gateway` varchar(30) DEFAULT NULL,
  `onlineTransactionStatus` enum('Success','Failure') DEFAULT NULL,
  `paymentToken` varchar(50) DEFAULT NULL,
  `paymentPayerID` varchar(50) DEFAULT NULL,
  `paymentTransactionID` varchar(50) DEFAULT NULL,
  `paymentReceiptID` varchar(50) DEFAULT NULL,
  `timestamp` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonPayment`
  ADD PRIMARY KEY (`gibbonPaymentID`);

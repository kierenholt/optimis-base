-- ============================================================================
-- Gibbon Blueprint Domain File: 09_system_lookups_and_legacy.sql
-- Tables in this file: 21
-- Note: Stripped of INSERT seed data for fast AI agent reading.
-- ============================================================================

-- [TABLE] gibbonAction | [TRIAGE] DROP | [CATEGORY] Legacy Gibbon PHP Framework / UI Internals
CREATE TABLE `gibbonAction` (
  `gibbonActionID` int(7) UNSIGNED ZEROFILL NOT NULL,
  `gibbonModuleID` int(4) UNSIGNED ZEROFILL NOT NULL,
  `name` varchar(50) NOT NULL COMMENT 'The action name should be unqiue to the module that it is related to',
  `precedence` int(2) NOT NULL,
  `category` varchar(20) NOT NULL,
  `description` varchar(255) NOT NULL,
  `helpURL` varchar(255) DEFAULT NULL,
  `URLList` text NOT NULL COMMENT 'Comma seperated list of all URLs that make up this action',
  `entryURL` varchar(255) NOT NULL,
  `entrySidebar` enum('Y','N') NOT NULL DEFAULT 'Y',
  `menuShow` enum('Y','N') NOT NULL DEFAULT 'Y',
  `defaultPermissionAdmin` enum('N','Y') NOT NULL DEFAULT 'N',
  `defaultPermissionTeacher` enum('N','Y') NOT NULL DEFAULT 'N',
  `defaultPermissionStudent` enum('N','Y') NOT NULL DEFAULT 'N',
  `defaultPermissionParent` enum('N','Y') NOT NULL DEFAULT 'N',
  `defaultPermissionSupport` enum('N','Y') NOT NULL DEFAULT 'N',
  `categoryPermissionStaff` enum('Y','N') NOT NULL DEFAULT 'Y',
  `categoryPermissionStudent` enum('Y','N') NOT NULL DEFAULT 'Y',
  `categoryPermissionParent` enum('Y','N') NOT NULL DEFAULT 'Y',
  `categoryPermissionOther` enum('Y','N') NOT NULL DEFAULT 'Y'
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonAction`
  ADD PRIMARY KEY (`gibbonActionID`),
  ADD UNIQUE KEY `moduleActionName` (`name`,`gibbonModuleID`),
  ADD KEY `gibbonModuleID` (`gibbonModuleID`);

-- [TABLE] gibbonCountry | [TRIAGE] CORE_LOOKUP | [CATEGORY] Reference Lookups & GDPR Retention
CREATE TABLE `gibbonCountry` (
  `printable_name` varchar(80) NOT NULL,
  `iddCountryCode` varchar(7) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonCountry`
  ADD PRIMARY KEY (`printable_name`);

-- [TABLE] gibbonCustomField | [TRIAGE] DROP | [CATEGORY] Legacy Gibbon PHP Framework / UI Internals
CREATE TABLE `gibbonCustomField` (
  `gibbonCustomFieldID` int(4) UNSIGNED ZEROFILL NOT NULL,
  `context` varchar(60) NOT NULL DEFAULT 'Person',
  `name` varchar(50) NOT NULL,
  `active` enum('Y','N') NOT NULL DEFAULT 'Y',
  `description` varchar(255) NOT NULL,
  `type` enum('varchar','text','date','time','url','select','checkboxes','radio','yesno','editor','color','number','image','file','code') NOT NULL,
  `options` text NOT NULL COMMENT 'Field length for varchar, rows for text, comma-separate list for select/checkbox.',
  `required` enum('N','Y') NOT NULL DEFAULT 'N',
  `hidden` enum('Y','N') DEFAULT 'N',
  `heading` varchar(90) NOT NULL,
  `sequenceNumber` int(4) NOT NULL,
  `activePersonStudent` tinyint(1) NOT NULL DEFAULT '0',
  `activePersonStaff` tinyint(1) NOT NULL DEFAULT '0',
  `activePersonParent` tinyint(1) NOT NULL DEFAULT '0',
  `activePersonOther` tinyint(1) NOT NULL DEFAULT '0',
  `activeApplicationForm` tinyint(1) NOT NULL DEFAULT '0',
  `activeDataUpdater` tinyint(1) NOT NULL DEFAULT '0',
  `activePublicRegistration` tinyint(1) NOT NULL DEFAULT '0'
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonCustomField`
  ADD PRIMARY KEY (`gibbonCustomFieldID`);

-- [TABLE] gibbonDataRetention | [TRIAGE] CORE_LOOKUP | [CATEGORY] Reference Lookups & GDPR Retention
CREATE TABLE `gibbonDataRetention` (
  `gibbonDataRetentionID` int(11) UNSIGNED ZEROFILL NOT NULL,
  `gibbonPersonID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `tables` text NOT NULL,
  `status` enum('Success','Partial Failure') DEFAULT NULL,
  `timestamp` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `gibbonPersonIDOperator` int(10) UNSIGNED ZEROFILL NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonDataRetention`
  ADD PRIMARY KEY (`gibbonDataRetentionID`),
  ADD UNIQUE KEY `gibbonPersonID` (`gibbonPersonID`);

-- [TABLE] gibbonDistrict | [TRIAGE] CORE_LOOKUP | [CATEGORY] Reference Lookups & GDPR Retention
CREATE TABLE `gibbonDistrict` (
  `gibbonDistrictID` int(6) UNSIGNED ZEROFILL NOT NULL,
  `name` varchar(30) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonDistrict`
  ADD PRIMARY KEY (`gibbonDistrictID`);

-- [TABLE] gibbonFileExtension | [TRIAGE] DROP | [CATEGORY] Legacy Gibbon PHP Framework / UI Internals
CREATE TABLE `gibbonFileExtension` (
  `gibbonFileExtensionID` int(4) UNSIGNED ZEROFILL NOT NULL,
  `type` enum('Document','Spreadsheet','Presentation','Graphics/Design','Video','Audio','Other') NOT NULL DEFAULT 'Other',
  `extension` varchar(7) NOT NULL,
  `name` varchar(50) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonFileExtension`
  ADD PRIMARY KEY (`gibbonFileExtensionID`);

-- [TABLE] gibbonHook | [TRIAGE] DROP | [CATEGORY] Legacy Gibbon PHP Framework / UI Internals
CREATE TABLE `gibbonHook` (
  `gibbonHookID` int(4) UNSIGNED ZEROFILL NOT NULL,
  `name` varchar(50) NOT NULL,
  `type` enum('Public Home Page','Student Profile','Parental Dashboard','Staff Dashboard','Student Dashboard','Report Writing','Staff Profile','Lesson Planner') DEFAULT NULL,
  `options` text NOT NULL,
  `gibbonModuleID` int(4) UNSIGNED ZEROFILL NOT NULL COMMENT 'The module which installed this hook.'
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonHook`
  ADD PRIMARY KEY (`gibbonHookID`),
  ADD UNIQUE KEY `name` (`name`,`type`);

-- [TABLE] gibbonLanguage | [TRIAGE] CORE_LOOKUP | [CATEGORY] Reference Lookups & GDPR Retention
CREATE TABLE `gibbonLanguage` (
  `gibbonLanguageID` int(4) UNSIGNED ZEROFILL NOT NULL,
  `name` varchar(30) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonLanguage`
  ADD PRIMARY KEY (`gibbonLanguageID`);

-- [TABLE] gibbonLog | [TRIAGE] DROP | [CATEGORY] Legacy Gibbon PHP Framework / UI Internals
CREATE TABLE `gibbonLog` (
  `gibbonLogID` int(16) UNSIGNED ZEROFILL NOT NULL,
  `gibbonModuleID` int(4) UNSIGNED ZEROFILL DEFAULT NULL,
  `gibbonPersonID` int(10) UNSIGNED ZEROFILL DEFAULT NULL,
  `gibbonSchoolYearID` int(3) UNSIGNED ZEROFILL NOT NULL,
  `timestamp` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `title` varchar(50) NOT NULL,
  `serialisedArray` text,
  `ip` varchar(15) DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonLog`
  ADD PRIMARY KEY (`gibbonLogID`);

-- [TABLE] gibbonMigration | [TRIAGE] DROP | [CATEGORY] Legacy Gibbon PHP Framework / UI Internals
CREATE TABLE `gibbonMigration` (
  `gibbonMigrationID` int(8) UNSIGNED ZEROFILL NOT NULL,
  `name` varchar(60) NOT NULL,
  `version` varchar(8) NOT NULL,
  `timestamp` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonMigration`
  ADD PRIMARY KEY (`gibbonMigrationID`);

-- [TABLE] gibbonModule | [TRIAGE] DROP | [CATEGORY] Legacy Gibbon PHP Framework / UI Internals
CREATE TABLE `gibbonModule` (
  `gibbonModuleID` int(4) UNSIGNED ZEROFILL NOT NULL COMMENT 'This number is assigned at install, and is only unique to the installation',
  `name` varchar(30) NOT NULL DEFAULT '' COMMENT 'This name should be globally unique preferably, but certainly locally unique',
  `description` text NOT NULL,
  `entryURL` varchar(255) NOT NULL DEFAULT 'index.php',
  `type` enum('Core','Additional') NOT NULL DEFAULT 'Core',
  `active` enum('Y','N') NOT NULL DEFAULT 'Y',
  `category` varchar(12) NOT NULL,
  `version` varchar(8) NOT NULL,
  `author` varchar(40) NOT NULL,
  `url` varchar(255) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonModule`
  ADD PRIMARY KEY (`gibbonModuleID`),
  ADD UNIQUE KEY `gibbonModuleName` (`name`);

-- [TABLE] gibbonNotification | [TRIAGE] DROP | [CATEGORY] Legacy Gibbon PHP Framework / UI Internals
CREATE TABLE `gibbonNotification` (
  `gibbonNotificationID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `gibbonPersonID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `status` enum('New','Archived') NOT NULL DEFAULT 'New',
  `gibbonModuleID` int(4) UNSIGNED ZEROFILL DEFAULT NULL,
  `count` int(4) NOT NULL DEFAULT '1',
  `text` text NOT NULL,
  `actionLink` varchar(255) NOT NULL COMMENT 'Relative to absoluteURL, start with a forward slash',
  `timestamp` datetime NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonNotification`
  ADD PRIMARY KEY (`gibbonNotificationID`),
  ADD KEY `gibbonPersonID` (`gibbonPersonID`);

-- [TABLE] gibbonNotificationEvent | [TRIAGE] DROP | [CATEGORY] Legacy Gibbon PHP Framework / UI Internals
CREATE TABLE `gibbonNotificationEvent` (
  `gibbonNotificationEventID` int(6) UNSIGNED ZEROFILL NOT NULL,
  `event` varchar(90) NOT NULL,
  `moduleName` varchar(30) NOT NULL,
  `actionName` varchar(50) NOT NULL,
  `type` enum('Core','Additional','CLI') NOT NULL DEFAULT 'Core',
  `scopes` varchar(255) NOT NULL DEFAULT 'All',
  `active` enum('Y','N') NOT NULL DEFAULT 'Y'
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonNotificationEvent`
  ADD PRIMARY KEY (`gibbonNotificationEventID`),
  ADD UNIQUE KEY `event` (`event`,`moduleName`);

-- [TABLE] gibbonNotificationListener | [TRIAGE] DROP | [CATEGORY] Legacy Gibbon PHP Framework / UI Internals
CREATE TABLE `gibbonNotificationListener` (
  `gibbonNotificationListenerID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `gibbonNotificationEventID` int(6) UNSIGNED ZEROFILL DEFAULT NULL,
  `gibbonPersonID` int(10) UNSIGNED ZEROFILL DEFAULT NULL,
  `scopeType` varchar(30) DEFAULT NULL,
  `scopeID` int(20) UNSIGNED DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonNotificationListener`
  ADD PRIMARY KEY (`gibbonNotificationListenerID`);

-- [TABLE] gibbonPermission | [TRIAGE] DROP | [CATEGORY] Legacy Gibbon PHP Framework / UI Internals
CREATE TABLE `gibbonPermission` (
  `permissionID` int(10) UNSIGNED ZEROFILL NOT NULL,
  `gibbonRoleID` int(3) UNSIGNED ZEROFILL NOT NULL,
  `gibbonActionID` int(7) UNSIGNED ZEROFILL NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonPermission`
  ADD PRIMARY KEY (`permissionID`),
  ADD KEY `gibbonRoleID` (`gibbonRoleID`),
  ADD KEY `gibbonActionID` (`gibbonActionID`);

-- [TABLE] gibbonSession | [TRIAGE] DROP | [CATEGORY] Legacy Gibbon PHP Framework / UI Internals
CREATE TABLE `gibbonSession` (
  `gibbonSessionID` varchar(40) NOT NULL,
  `gibbonPersonID` int(10) UNSIGNED ZEROFILL DEFAULT NULL,
  `gibbonActionID` int(7) UNSIGNED ZEROFILL DEFAULT NULL,
  `sessionData` mediumtext,
  `sessionStatus` varchar(20) DEFAULT NULL,
  `timestampCreated` timestamp NULL DEFAULT NULL,
  `timestampModified` timestamp NULL DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonSession`
  ADD PRIMARY KEY (`gibbonSessionID`);

-- [TABLE] gibbonSetting | [TRIAGE] DROP | [CATEGORY] Legacy Gibbon PHP Framework / UI Internals
CREATE TABLE `gibbonSetting` (
  `gibbonSettingID` int(5) UNSIGNED ZEROFILL NOT NULL,
  `scope` varchar(50) NOT NULL,
  `name` varchar(50) NOT NULL,
  `nameDisplay` varchar(60) NOT NULL,
  `description` varchar(255) NOT NULL,
  `value` text NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonSetting`
  ADD PRIMARY KEY (`gibbonSettingID`),
  ADD UNIQUE KEY `scope` (`scope`,`nameDisplay`),
  ADD UNIQUE KEY `scope_2` (`scope`,`name`);

-- [TABLE] gibbonString | [TRIAGE] DROP | [CATEGORY] Legacy Gibbon PHP Framework / UI Internals
CREATE TABLE `gibbonString` (
  `gibbonStringID` int(8) UNSIGNED ZEROFILL NOT NULL,
  `original` varchar(255) NOT NULL,
  `replacement` varchar(255) NOT NULL,
  `mode` enum('Whole','Partial') NOT NULL,
  `caseSensitive` enum('Y','N') NOT NULL,
  `priority` int(2) NOT NULL DEFAULT '0'
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonString`
  ADD PRIMARY KEY (`gibbonStringID`);

-- [TABLE] gibbonTheme | [TRIAGE] DROP | [CATEGORY] Legacy Gibbon PHP Framework / UI Internals
CREATE TABLE `gibbonTheme` (
  `gibbonThemeID` int(4) UNSIGNED ZEROFILL NOT NULL,
  `name` varchar(30) NOT NULL,
  `description` varchar(100) NOT NULL,
  `active` enum('N','Y') NOT NULL DEFAULT 'N',
  `version` varchar(6) NOT NULL,
  `author` varchar(40) NOT NULL,
  `url` varchar(255) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibbonTheme`
  ADD PRIMARY KEY (`gibbonThemeID`);

-- [TABLE] gibbonUsernameFormat | [TRIAGE] DROP | [CATEGORY] Legacy Gibbon PHP Framework / UI Internals
CREATE TABLE `gibbonUsernameFormat` (
  `gibbonUsernameFormatID` int(3) UNSIGNED ZEROFILL NOT NULL,
  `gibbonRoleIDList` varchar(255) COLLATE utf8_unicode_ci DEFAULT NULL,
  `format` varchar(255) COLLATE utf8_unicode_ci DEFAULT NULL,
  `isDefault` enum('Y','N') COLLATE utf8_unicode_ci NOT NULL DEFAULT 'N',
  `isNumeric` enum('Y','N') COLLATE utf8_unicode_ci NOT NULL DEFAULT 'N',
  `numericValue` int(12) UNSIGNED NOT NULL DEFAULT '0',
  `numericIncrement` int(3) UNSIGNED NOT NULL DEFAULT '1',
  `numericSize` int(3) UNSIGNED NOT NULL DEFAULT '4'
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_unicode_ci;
ALTER TABLE `gibbonUsernameFormat`
  ADD PRIMARY KEY (`gibbonUsernameFormatID`);

-- [TABLE] gibboni18n | [TRIAGE] DROP | [CATEGORY] Legacy Gibbon PHP Framework / UI Internals
CREATE TABLE `gibboni18n` (
  `gibboni18nID` int(4) UNSIGNED ZEROFILL NOT NULL,
  `code` varchar(5) NOT NULL,
  `name` varchar(100) NOT NULL,
  `version` varchar(10) DEFAULT NULL,
  `active` enum('Y','N') NOT NULL DEFAULT 'Y',
  `installed` enum('Y','N') NOT NULL DEFAULT 'N',
  `systemDefault` enum('Y','N') NOT NULL DEFAULT 'N',
  `dateFormat` varchar(20) NOT NULL,
  `dateFormatRegEx` text NOT NULL,
  `dateFormatPHP` varchar(20) NOT NULL,
  `rtl` enum('Y','N') NOT NULL DEFAULT 'N'
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
ALTER TABLE `gibboni18n`
  ADD PRIMARY KEY (`gibboni18nID`);

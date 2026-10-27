-- SPDX-License-Identifier: Apache-2.0
-- Licensed to 1EdTech Consortium, Inc. under one or more agreements.
-- 1EdTech Consortium, Inc. licenses this file to you under the Apache License, Version 2.0.
-- See the LICENSE and NOTICES files in the project root for more information.

drop index if exists oneroster12.enrollments_sourcedid;
drop materialized view if exists oneroster12.enrollments;
--
create materialized view oneroster12.enrollments as
with staff_section_associations as (
    select * from edfi.staffSectionAssociation
),
student_section_associations as (
    select * from edfi.studentSectionAssociation
),
sections as (
    select * from edfi.section
),
staff_enrollments_formatted as (
    select
        md5(concat(
            'STA-', lower(staff.staffUniqueId)::varchar,
            '-', lower(sections.localcoursecode)::varchar,
            '-', sections.schoolid::varchar,
            '-', sections.schoolyear::varchar,
            '-', lower(sections.sectionidentifier)::varchar,
            '-', lower(sections.sessionname)::varchar,
            '-', beginDate::varchar
        )) as "sourcedId", -- unique ID constructed from natural key of Ed-Fi StaffSectionAssociations
        'active' as "status",
        ssa.lastmodifieddate as "dateLastModified",
        json_build_object(
            'href', concat('/classes/', md5(concat(
                lower(sections.localcoursecode)::varchar,
                '-', sections.schoolid::varchar,
                '-', sections.schoolyear::varchar,
                '-', l


-- SPDX-License-Identifier: Apache-2.0
-- Licensed to 1EdTech Consortium, Inc. under one or more agreements.
-- 1EdTech Consortium, Inc. licenses this file to you under the Apache License, Version 2.0.
-- See the LICENSE and NOTICES files in the project root for more information.

drop index if exists oneroster12.courses_sourcedid;
drop materialized view if exists oneroster12.courses;
--
create materialized view if not exists oneroster12.courses as
with course as (
    select * from edfi.course
),
course_offerings as (
    -- one offering row per course (latest school year wins) so a course with
    -- offerings in multiple years still yields a single courses row. Keyed by
    -- the course natural key (org + code) so courses sharing a code across
    -- organizations don't pick up each other's offerings.
    select educationorganizationid, coursecode, max(schoolyear) as schoolyear
    from edfi.courseoffering
    group by educationorganizationid, coursecode
)
-- property documentation at
-- https://www.imsglobal.org/sites/default/files/spec/oneroster/v1p2/rostering-restbinding/OneRosterv1p2RosteringService_RESTBindv1p0.html#Main6p8p2
select
    md5(concat(
                crs.educationOrganizationId::varchar,
                '-', crs.courseCode::varchar
            )) as "sourcedId", -- unique ID constructed from natural key of Ed-Fi Courses
    'active' as "status",
    crs.lastmodifieddate as 


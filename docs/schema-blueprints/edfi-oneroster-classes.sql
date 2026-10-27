-- SPDX-License-Identifier: Apache-2.0
-- Licensed to 1EdTech Consortium, Inc. under one or more agreements.
-- 1EdTech Consortium, Inc. licenses this file to you under the Apache License, Version 2.0.
-- See the LICENSE and NOTICES files in the project root for more information.

drop index if exists oneroster12.classes_sourcedid;
drop materialized view if exists oneroster12.classes;
--
create materialized view if not exists oneroster12.classes as
with section as (
    select * from edfi.section
),
courseoffering as (
    -- avoid column ambiguity in next step
    select
        off.*,
        sch.localEducationAgencyId
    from edfi.courseoffering off
        join edfi.school sch
            on off.schoolid = sch.schoolid
),
periods as (
    select
        sectionidentifier, localcoursecode, schoolid, schoolyear, sessionname,
        jsonb_agg(distinct classperiodname) as periods
    from edfi.sectionclassperiod
    group by sectionidentifier, localcoursecode, schoolid, schoolyear, sessionname
),
-- property documentation at
-- https://www.imsglobal.org/sites/default/files/spec/oneroster/v1p2/rostering-restbinding/OneRosterv1p2RosteringService_RESTBindv1p0.html#Main6p6p2
classes as (
	select
	    md5(concat(
            lower(section.localcoursecode)::varchar,
            '-', section.schoolid::varchar,
            '-', section.schoolyear::varchar,
            '-', lower(section.sectionidentifier)::varchar,
            '-', l


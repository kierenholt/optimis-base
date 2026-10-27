-- SPDX-License-Identifier: Apache-2.0
-- Licensed to 1EdTech Consortium, Inc. under one or more agreements.
-- 1EdTech Consortium, Inc. licenses this file to you under the Apache License, Version 2.0.
-- See the LICENSE and NOTICES files in the project root for more information.

drop index if exists oneroster12.demographics_sourcedid;
drop materialized view if exists oneroster12.demographics;
--
create materialized view if not exists oneroster12.demographics as
with student as (
    select * from edfi.student
),
-- One row per (student, school), mirroring the users view's anchor so the
-- school-keyed sourcedId always matches a user. Race/ethnicity is resolved
-- per-student (below) and replicated onto each of the student's school rows.
student_school as (
    select distinct
        ssa.studentusi,
        ssa.schoolid
    from edfi.studentschoolassociation ssa
    join edfi.school s on ssa.schoolid = s.schoolid
),
-- Hispanic/ethnicity is a person-level fact, not an org-level one. Aggregate
-- across ALL of the student's demographic records regardless of the org level
-- (school, LEA, ESC, SEA) they were recorded at, so higher-org data is never dropped.
-- DS 6.1: hispaniclatinoethnicity moved off studenteducationorganizationassociation
-- onto edfi.studentdemographic, which is keyed the same way on
-- (educationorganizationid, studentusi), so the roll-up below is unchanged.
student_edorg as (
    select
        se.studentusi,
        bool_or(se.hispaniclatinoethnicity) as hispaniclatinoethnicity,
        max(se.lastmodifieddate) as edorg_lmdate
    from edfi.studentdemographic se
    group by se.studentusi
),
-- Race is likewise person-level: union the mapped race descriptors across all
-- of the student's demographic records.
student_race as (
    select
        seoar.studentusi,
        array_remove(array_agg(distinct mappedracedescriptor.mappedvalue::text), null) as race_array
    from edfi.studentdemographicrace seoar
    joi


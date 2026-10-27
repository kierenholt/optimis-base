-- SPDX-License-Identifier: Apache-2.0
-- Licensed to 1EdTech Consortium, Inc. under one or more agreements.
-- 1EdTech Consortium, Inc. licenses this file to you under the Apache License, Version 2.0.
-- See the LICENSE and NOTICES files in the project root for more information.

drop index if exists oneroster12.users_sourcedid;
drop index if exists oneroster12.users_participantusi;
drop materialized view if exists oneroster12.users;
--
create materialized view if not exists oneroster12.users as
with student as (
    select * from edfi.student
),
student_school as (
    select * from edfi.studentSchoolAssociation
),
school as (
    select * from edfi.school
),
staff as (
    select * from edfi.staff
),
staff_school as (
    select * from edfi.staffschoolassociation
),
staff_edorg_assign as (
    select * from edfi.staffeducationorganizationassignmentassociation
),
student_ids as (
    select
        seoa_sid.studentusi,
        seoa_sid.educationOrganizationId,
        json_agg(
            json_build_object(
                'type', studentIDsystemDescriptor.codeValue,
                'identifier', identificationcode
            )
            order by studentIDsystemDescriptor.codeValue
        ) as ids
    from edfi.studentidentificationcode seoa_sid
        join edfi.descriptor studentIDsystemDescriptor
            on seoa_sid.studentIdentificationSystemDescriptorId=studentIDsystemDescriptor.descriptorId
    group by 1,2
),
-- DS 6.1: studenteducationorganizationassociationelectronicmail was renamed to
-- edfi.studentdirectoryelectronicmail; the key is unchanged.
student_email as (
    select x.*
    from (
        select
            seoa_et.*,
            emailtypedescriptor.codevalue = 'Home/Personal' as is_preferred,
            row_number() over(
                partition by studentusi
                order by (emailtypedescriptor.codevalue = 'Home/Personal') desc nulls last,
                         emailtypedescriptor.codevalue
            ) as seq
        from edfi.studentdirectoryelectronicmail as seoa_et
            join edfi.descriptor emailtypedescriptor
                on seoa_et.electronicMailTypeDescriptorId=emailtypedescriptor.descriptorid
        -- exclude do-not-publish addresses before ranking, so a suppressed preferred
        -- address falls through to the next publishable one instead of leaving no email
        where donotpublishindicator is null or not donotpublishindicator
    ) x
    where seq = 1
),
student_orgs as (
    select
        studentusi,
        school.localEducationAgencyId,
        school.schoolId,
        md5(school.schoolId::text) as sourcedid,
        student_school.primarySchool,
        student_school.entryDate
    from student_school
        join school
            on student_school.schoolId = school.schoolId
),
student_primary_org as (
    select studentusi, schoolid
    from (
        select
            studentusi,
            schoolid,
            row_number() over (
                partition by studentusi
                order by (case when primarySchool then 1 else 0 end) desc,
                         entryDate desc,
                         schoolid
            ) as seq
        from student_orgs
    ) ranked
    where seq = 1
),
student_orgs_agg as (
    select
        student_orgs.studentusi,
        json_agg(
            json_build_object(
                'roleType', case
                    when student_primary_org.schoolid is not null
                         and student_orgs.schoolid = student_primary_org.schoolid then 'primary'
                    else 'secondary'
                end,
                'role', 'student',
                'org', json_build_object(
                    'href', concat('/orgs/', student_orgs.sourcedid::text),
                    'sourcedId', student_orgs.sourcedid,
                    'type', 'org'
                )
            )
            -- deterministic order so the array matches the MSSQL artifact
            order by student_orgs.schoolid
        ) AS "roles"
    -- dedup to one row per (student, school): a re-enrollment at the same school
    -- must not duplicate that org in the roles array.
    from (select distinct studentusi, schoolid, sourcedid from student_orgs) student_orgs
        left join student_primary_org
            on student_orgs.studentusi = student_primary_org.studentusi
    group by student_orgs.studentusi
),
student_grade as (
    select x.*
    from (
        select
            studentusi,
            gradeleveldescriptor.codevalue as grade_level,
            row_number() over(
                -- partition by student alone (not student, schoolyear) so a
                -- student with enrolments in multiple years yields one user row
                partition by studentusi
                order by
                    entrydate desc,
                    exitwithdrawdate desc nulls first,
                    gradeleveldescriptor.codevalue desc
            ) as seq
        from student_school
            join edfi.descriptor gradeleveldescriptor
                on student_


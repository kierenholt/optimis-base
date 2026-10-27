-- SPDX-License-Identifier: Apache-2.0
-- Licensed to 1EdTech Consortium, Inc. under one or more agreements.
-- 1EdTech Consortium, Inc. licenses this file to you under the Apache License, Version 2.0.
-- See the LICENSE and NOTICES files in the project root for more information.

drop index if exists oneroster12.orgs_sourcedid;
drop materialized view if exists oneroster12.orgs;
--
create materialized view if not exists oneroster12.orgs as
with schools as (
    select
        school.*,
    	schoolOrg.*,
        leaOrg.id as leaId
    from edfi.school
        join edfi.educationOrganization schoolOrg
            on schoolOrg.educationOrganizationId = school.schoolId
        left join edfi.educationOrganization leaOrg
            on leaOrg.educationOrganizationId = school.localEducationAgencyId
),
leas as (
    select
        localEducationAgency.*,
        leaOrg.*,
        seaOrg.id as seaId
    from edfi.localEducationAgency
        join edfi.educationOrganization leaOrg
            on leaOrg.educationOrganizationId = localEducationAgency.localEducationAgencyId
        left join edfi.educationOrganization seaOrg
            on seaOrg.educationOrganizationId = localEducationAgency.stateEducationAgencyId
),
seas as (
    select
    	stateEducationAgency.*,
    	seaOrg.*
    from edfi.stateEducationAg


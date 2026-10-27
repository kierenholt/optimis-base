-- SPDX-License-Identifier: Apache-2.0
-- Licensed to 1EdTech Consortium, Inc. under one or more agreements.
-- 1EdTech Consortium, Inc. licenses this file to you under the Apache License, Version 2.0.
-- See the LICENSE and NOTICES files in the project root for more information.

drop index if exists oneroster12.academicsessions_sourcedid;
drop materialized view if exists oneroster12.academicsessions;
--
create materialized view if not exists oneroster12.academicsessions as
with sessions as (
    select ses.*, sch.localEducationAgencyid
    from edfi.session ses
        join edfi.school sch
            on ses.schoolid = sch.schoolid
),
calendar_windows as (
    select
        cd.schoolid,
        cd.schoolyear,
        cd.calendarcode,
        min(cd.date) as first_school_day,
        max(cd.date) as last_school_day,
        array_agg(distinct eventdescriptor.codevalue) as eventdescriptors
    from edfi.calendardate cd
    	join edfi.calendardatecalendarevent cdce
    		on cd.schoolid=cdce.schoolid
    			and cd.date=cdce.date
    			and cd.schoolyear=cdce.schoolyear
    			and cd.calendarcode=cdce.calendarcode
        join edfi.descriptor eventdescriptor
            on cdce.calendareventdescriptorid=eventdescriptor.descriptorid
        join edfi.descriptormapping mappedeventdescriptor
            on mappedeventdescriptor.value=eventdescriptor.codevalue
                and mappedeventdescriptor.namespace=eventdescriptor.namespace
                and mappedeventdescriptor.mappednamespace='uri://1edtech.org/oneroster12/CalendarEventDescriptor'
    where mappedeventdescriptor.mappedvalue = 'TRUE' -- IS a school/instructional day
    group by grouping sets (
        -- school level
   


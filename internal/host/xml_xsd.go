package host

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	lua "github.com/yuin/gopher-lua"
)

var (
	xmlTagNameRegex    = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_\-\.]*$`)
	isoDateRegex       = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	urnSixDigitRegex   = regexp.MustCompile(`^\d{6}$`)
	upnThirteenRegex   = regexp.MustCompile(`^[A-Z][0-9]{12}$`)
	ulnTenDigitRegex   = regexp.MustCompile(`^[0-9]{10}$`)
)

func escapeXMLText(raw string) string {
	var buf bytes.Buffer
	_ = xml.EscapeText(&buf, []byte(raw))
	return buf.String()
}

func serializeLuaXMLNode(L *lua.LState, nodeTbl *lua.LTable, buf *bytes.Buffer) error {
	tagVal := nodeTbl.RawGetString("tag")
	if tagVal == lua.LNil {
		return fmt.Errorf("ctx.xml.build node is missing mandatory 'tag' field")
	}
	tag := strings.TrimSpace(tagVal.String())
	if !xmlTagNameRegex.MatchString(tag) {
		return fmt.Errorf("ctx.xml.build invalid XML tag name: %q", tag)
	}

	buf.WriteByte('<')
	buf.WriteString(tag)

	if attrsVal := nodeTbl.RawGetString("attrs"); attrsVal != lua.LNil {
		attrsTbl, ok := attrsVal.(*lua.LTable)
		if !ok {
			return fmt.Errorf("ctx.xml.build 'attrs' on <%s> must be a table", tag)
		}
		var attrKeys []string
		attrMap := make(map[string]string)
		attrsTbl.ForEach(func(k, v lua.LValue) {
			key := strings.TrimSpace(k.String())
			if key != "" && v != lua.LNil {
				attrKeys = append(attrKeys, key)
				attrMap[key] = v.String()
			}
		})
		sort.Strings(attrKeys)
		for _, k := range attrKeys {
			if !xmlTagNameRegex.MatchString(k) {
				return fmt.Errorf("ctx.xml.build invalid XML attribute name %q on <%s>", k, tag)
			}
			buf.WriteByte(' ')
			buf.WriteString(k)
			buf.WriteString(`="`)
			buf.WriteString(escapeXMLText(attrMap[k]))
			buf.WriteByte('"')
		}
	}

	buf.WriteByte('>')

	if childrenVal := nodeTbl.RawGetString("children"); childrenVal != lua.LNil {
		childrenTbl, ok := childrenVal.(*lua.LTable)
		if !ok {
			return fmt.Errorf("ctx.xml.build 'children' on <%s> must be a table", tag)
		}
		maxn := childrenTbl.MaxN()
		for i := 1; i <= maxn; i++ {
			childVal := childrenTbl.RawGetInt(i)
			childTbl, ok := childVal.(*lua.LTable)
			if !ok {
				return fmt.Errorf("ctx.xml.build child[%d] of <%s> must be a table", i, tag)
			}
			if err := serializeLuaXMLNode(L, childTbl, buf); err != nil {
				return err
			}
		}
	}

	if textVal := nodeTbl.RawGetString("text"); textVal != lua.LNil {
		buf.WriteString(escapeXMLText(textVal.String()))
	}

	buf.WriteString("</")
	buf.WriteString(tag)
	buf.WriteByte('>')
	return nil
}

type dfeCensusXMLMessage struct {
	XMLName xml.Name `xml:"Message"`
	Header  struct {
		Collection string `xml:"Collection"`
		Term       string `xml:"Term"`
		CensusDate string `xml:"CensusDate"`
		Source     struct {
			LEA        string `xml:"LEA"`
			Estab      string `xml:"Estab"`
			URN        string `xml:"URN"`
			SchoolName string `xml:"SchoolName"`
		} `xml:"Source"`
	} `xml:"Header"`
	Pupils struct {
		PupilList []struct {
			PupilIdentifiers struct {
				UPN      string `xml:"UPN"`
				ULN      string `xml:"ULN"`
				Surname  string `xml:"Surname"`
				Forename string `xml:"Forename"`
				DOB      string `xml:"DOB"`
				Sex      string `xml:"Sex"`
			} `xml:"PupilIdentifiers"`
			PupilCharacteristics struct {
				Ethnicity    string `xml:"Ethnicity"`
				Language     string `xml:"Language"`
				SENProvision string `xml:"SENProvision"`
				SENNeed      string `xml:"SENNeed"`
				LookedAfter  string `xml:"LookedAfter"`
				FSMPeriods   *struct {
					Periods []struct {
						FSMStartDate string `xml:"FSMStartDate"`
						FSMEndDate   string `xml:"FSMEndDate"`
						Country      string `xml:"Country"`
					} `xml:"FSMPeriod"`
				} `xml:"FSMPeriods"`
			} `xml:"PupilCharacteristics"`
			Enrolment struct {
				NCYearActual    string `xml:"NCYearActual"`
				EnrolmentStatus string `xml:"EnrolmentStatus"`
				EntryDate       string `xml:"EntryDate"`
				LeavingDate     string `xml:"LeavingDate"`
			} `xml:"Enrolment"`
			Attendance struct {
				PossibleSessions            string `xml:"PossibleSessions"`
				SessionsPresent             string `xml:"SessionsPresent"`
				AuthorisedAbsenceSessions   string `xml:"AuthorisedAbsenceSessions"`
				UnauthorisedAbsenceSessions string `xml:"UnauthorisedAbsenceSessions"`
			} `xml:"Attendance"`
		} `xml:"Pupil"`
	} `xml:"Pupils"`
}

func ValidateDfeSchoolCensusXML(xmlStr string) (bool, []string, *dfeCensusXMLMessage) {
	trimmed := strings.TrimSpace(xmlStr)
	if trimmed == "" {
		return false, []string{"empty XML document"}, nil
	}

	var msg dfeCensusXMLMessage
	if err := xml.Unmarshal([]byte(trimmed), &msg); err != nil {
		return false, []string{fmt.Sprintf("malformed XML: %v", err)}, nil
	}

	var errs []string
	if msg.XMLName.Local != "Message" {
		errs = append(errs, fmt.Sprintf("root element must be <Message>, got <%s>", msg.XMLName.Local))
	}
	if strings.TrimSpace(msg.Header.Collection) != "SchoolCensus" {
		errs = append(errs, fmt.Sprintf("<Header><Collection> must be 'SchoolCensus', got %q", msg.Header.Collection))
	}
	term := strings.TrimSpace(msg.Header.Term)
	if term != "AUTUMN" && term != "SPRING" && term != "SUMMER" {
		errs = append(errs, fmt.Sprintf("<Header><Term> must be AUTUMN, SPRING, or SUMMER, got %q", term))
	}
	if !isoDateRegex.MatchString(strings.TrimSpace(msg.Header.CensusDate)) {
		errs = append(errs, fmt.Sprintf("<Header><CensusDate> must match YYYY-MM-DD, got %q", msg.Header.CensusDate))
	}
	if !urnSixDigitRegex.MatchString(strings.TrimSpace(msg.Header.Source.URN)) {
		errs = append(errs, fmt.Sprintf("<Header><Source><URN> must be a 6-digit URN, got %q", msg.Header.Source.URN))
	}

	if len(msg.Pupils.PupilList) == 0 {
		errs = append(errs, "<Pupils> must contain at least one <Pupil> record")
	}

	for idx, p := range msg.Pupils.PupilList {
		upn := strings.TrimSpace(p.PupilIdentifiers.UPN)
		if !upnThirteenRegex.MatchString(upn) {
			errs = append(errs, fmt.Sprintf("Pupil[%d] <UPN> %q does not match statutory 13-character format ^[A-Z][0-9]{12}$", idx+1, upn))
		}
		uln := strings.TrimSpace(p.PupilIdentifiers.ULN)
		if uln != "" && !ulnTenDigitRegex.MatchString(uln) {
			errs = append(errs, fmt.Sprintf("Pupil[%d] <ULN> %q does not match 10-digit format ^[0-9]{10}$", idx+1, uln))
		}
		if strings.TrimSpace(p.PupilIdentifiers.Surname) == "" {
			errs = append(errs, fmt.Sprintf("Pupil[%d] <Surname> is required", idx+1))
		}
		if strings.TrimSpace(p.PupilIdentifiers.Forename) == "" {
			errs = append(errs, fmt.Sprintf("Pupil[%d] <Forename> is required", idx+1))
		}
		if !isoDateRegex.MatchString(strings.TrimSpace(p.PupilIdentifiers.DOB)) {
			errs = append(errs, fmt.Sprintf("Pupil[%d] <DOB> %q must match YYYY-MM-DD", idx+1, p.PupilIdentifiers.DOB))
		}
		sex := strings.TrimSpace(p.PupilIdentifiers.Sex)
		if sex != "M" && sex != "F" && sex != "U" && sex != "X" {
			errs = append(errs, fmt.Sprintf("Pupil[%d] <Sex> %q must be M, F, U, or X", idx+1, sex))
		}

		senProv := strings.TrimSpace(p.PupilCharacteristics.SENProvision)
		if senProv != "N" && senProv != "K" && senProv != "E" {
			errs = append(errs, fmt.Sprintf("Pupil[%d] <SENProvision> %q must be N, K, or E", idx+1, senProv))
		}
		lac := strings.TrimSpace(p.PupilCharacteristics.LookedAfter)
		if lac != "0" && lac != "1" && lac != "Y" && lac != "N" {
			errs = append(errs, fmt.Sprintf("Pupil[%d] <LookedAfter> %q must be 0, 1, Y, or N", idx+1, lac))
		}

		if strings.TrimSpace(p.Enrolment.NCYearActual) == "" {
			errs = append(errs, fmt.Sprintf("Pupil[%d] <NCYearActual> is required", idx+1))
		}
		enrStatus := strings.TrimSpace(p.Enrolment.EnrolmentStatus)
		if enrStatus != "C" && enrStatus != "M" && enrStatus != "S" && enrStatus != "F" && enrStatus != "O" {
			errs = append(errs, fmt.Sprintf("Pupil[%d] <EnrolmentStatus> %q must be C, M, S, F, or O", idx+1, enrStatus))
		}
		if !isoDateRegex.MatchString(strings.TrimSpace(p.Enrolment.EntryDate)) {
			errs = append(errs, fmt.Sprintf("Pupil[%d] <EntryDate> %q must match YYYY-MM-DD", idx+1, p.Enrolment.EntryDate))
		}

		for _, attCheck := range []struct {
			name string
			val  string
		}{
			{"PossibleSessions", p.Attendance.PossibleSessions},
			{"SessionsPresent", p.Attendance.SessionsPresent},
			{"AuthorisedAbsenceSessions", p.Attendance.AuthorisedAbsenceSessions},
			{"UnauthorisedAbsenceSessions", p.Attendance.UnauthorisedAbsenceSessions},
		} {
			n, convErr := strconv.Atoi(strings.TrimSpace(attCheck.val))
			if convErr != nil || n < 0 {
				errs = append(errs, fmt.Sprintf("Pupil[%d] <%s> %q must be a non-negative integer", idx+1, attCheck.name, attCheck.val))
			}
		}
	}

	return len(errs) == 0, errs, &msg
}

func (h *Host) buildLuaXMLTable(L *lua.LState) *lua.LTable {
	xmlTbl := L.NewTable()

	L.SetField(xmlTbl, "build", L.NewFunction(func(L *lua.LState) int {
		nodeTbl, ok := L.Get(1).(*lua.LTable)
		if !ok {
			L.RaiseError("ctx.xml.build requires a root node table as its first argument")
			return 0
		}

		var buf bytes.Buffer
		buf.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
		if err := serializeLuaXMLNode(L, nodeTbl, &buf); err != nil {
			L.RaiseError("%s", err.Error())
			return 0
		}

		L.Push(lua.LString(buf.String()))
		return 1
	}))

	L.SetField(xmlTbl, "validate_xsd", L.NewFunction(func(L *lua.LState) int {
		xmlStr := strings.TrimSpace(L.OptString(1, ""))
		schemaName := strings.TrimSpace(L.OptString(2, ""))
		if xmlStr == "" {
			L.RaiseError("ctx.xml.validate_xsd requires a non-empty XML string")
			return 0
		}
		if schemaName == "" {
			L.RaiseError("ctx.xml.validate_xsd requires a non-empty XSD schema name")
			return 0
		}
		if schemaName != "dfe_school_census_v1.xsd" {
			L.RaiseError("XSD_SCHEMA_NOT_FOUND: unknown or unsupported XSD schema %q", schemaName)
			return 0
		}

		valid, errs, _ := ValidateDfeSchoolCensusXML(xmlStr)
		errsTbl := L.NewTable()
		markLuaTableAsArray(L, errsTbl)
		for i, e := range errs {
			errsTbl.RawSetInt(i+1, lua.LString(e))
		}

		resTbl := L.NewTable()
		L.SetField(resTbl, "valid", lua.LBool(valid))
		L.SetField(resTbl, "errors", errsTbl)
		L.Push(resTbl)
		return 1
	}))

	return xmlTbl
}

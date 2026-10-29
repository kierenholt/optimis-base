package dal

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

const (
	EventCorePupilEnrolled               = "core.pupil.enrolled"
	EventCoreGroupMembershipChanged      = "core.group.membership_changed"
	EventCoreAttendanceSessionRecorded   = "core.attendance.session_recorded"
	EventCoreSafeguardingPostboxCreated  = "core.safeguarding.postbox_created"
	EventCoreGDPRPupilPurged             = "core.gdpr.pupil_purged"
	EventCoreParentDMZSubmissionReceived = "core.parent_dmz.submission_received"
)

var standardCoreEventCatalog = []string{
	EventCorePupilEnrolled,
	EventCoreGroupMembershipChanged,
	EventCoreAttendanceSessionRecorded,
	EventCoreSafeguardingPostboxCreated,
	EventCoreGDPRPupilPurged,
	EventCoreParentDMZSubmissionReceived,
}

type CoreDomainEvent struct {
	Name    string         `json:"name"`
	Payload map[string]any `json:"payload"`
	Hop     uint8          `json:"hop"`
}

type PostCommitEventHook func(evt CoreDomainEvent)

type eventHopCtxKey struct{}

func WithEventHop(ctx context.Context, hop uint8) context.Context {
	return context.WithValue(ctx, eventHopCtxKey{}, hop)
}

func EventHopFromContext(ctx context.Context) uint8 {
	if ctx == nil {
		return 0
	}
	if v, ok := ctx.Value(eventHopCtxKey{}).(uint8); ok {
		return v
	}
	return 0
}

func CoreEventCatalog() []string {
	out := make([]string, len(standardCoreEventCatalog))
	copy(out, standardCoreEventCatalog)
	return out
}

func IsValidCoreEventName(name string) bool {
	trimmed := strings.TrimSpace(name)
	for _, cat := range standardCoreEventCatalog {
		if trimmed == cat {
			return true
		}
	}
	return false
}

func ValidateEventSubscriptionName(name string) error {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return errors.New("event name cannot be empty")
	}
	if strings.HasPrefix(trimmed, "core.") && !IsValidCoreEventName(trimmed) {
		return fmt.Errorf("unknown Core domain event %q: must be declared in the Standard Core Domain Event Catalog", trimmed)
	}
	return nil
}

func (d *CoreDAL) SetPostCommitEventHook(hook PostCommitEventHook) {
	d.attachMu.Lock()
	defer d.attachMu.Unlock()
	d.postCommitEventHook = hook
}

func (d *CoreDAL) emitPostCommitEvent(ctx context.Context, name string, payload map[string]any) {
	d.attachMu.RLock()
	hook := d.postCommitEventHook
	d.attachMu.RUnlock()
	if hook == nil {
		return
	}
	parentHop := EventHopFromContext(ctx)
	nextHop := parentHop + 1
	if nextHop == 0 {
		nextHop = 1
	}
	if payload == nil {
		payload = map[string]any{}
	}
	hook(CoreDomainEvent{
		Name:    name,
		Payload: payload,
		Hop:     nextHop,
	})
}

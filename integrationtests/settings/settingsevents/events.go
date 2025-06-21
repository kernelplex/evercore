package settingsevents

import (
	"github.com/kernelplex/evercore/base"
	"github.com/kernelplex/evercore/integrationtests/generated/events"
)

// evercore:event
type SettingUpdatedEvent struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func (s *SettingUpdatedEvent) AsEvent() evercore.EventState {
	return s
}

func (s *SettingUpdatedEvent) GetEventType() string {
	return generated_events.SettingUpdatedEventType
}

func (s *SettingUpdatedEvent) Serialize() string {
	return evercore.SerializeToJson(s)
}

package settings

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/kernelplex/evercore/base"
	"github.com/kernelplex/evercore/integrationtests/settings/settingsevents"
)

// evercore:aggregate
type SystemSettingsAggregate struct {
	Id       int64             `json:"id"`
	Sequence int64             `json:"sequence"`
	Settings map[string]string `json:"settings"`
}

func NewSystemSettingsAggregate() *SystemSettingsAggregate {
	return &SystemSettingsAggregate{
		Settings: make(map[string]string),
	}
}

func (s *SystemSettingsAggregate) AsAggregate() evercore.Aggregate {
	return s
}

func (s *SystemSettingsAggregate) GetId() int64 {
	return s.Id
}

func (s *SystemSettingsAggregate) SetId(id int64) {
	s.Id = id
}

func (s *SystemSettingsAggregate) GetSequence() int64 {
	return s.Sequence
}

func (s *SystemSettingsAggregate) SetSequence(seq int64) {
	s.Sequence = seq
}

func (s *SystemSettingsAggregate) GetAggregateType() string {
	return "SystemSettingsAggregate"
}

func (s *SystemSettingsAggregate) GetSnapshotFrequency() int64 {
	return 10 // Take snapshot every 10 events
}

func (s *SystemSettingsAggregate) GetSnapshotState() (*string, error) {
	data, err := json.Marshal(s.Settings)
	if err != nil {
		return nil, err
	}
	state := string(data)
	return &state, nil
}

func (s *SystemSettingsAggregate) ApplyEventState(eventState evercore.EventState, eventTime time.Time, reference string) error {
	switch ev := eventState.(type) {
	case *settingsevents.SettingUpdatedEvent:
		s.Settings[ev.Key] = ev.Value
	default:
		return fmt.Errorf("unknown event type: %T", eventState)
	}
	return nil
}

func (s *SystemSettingsAggregate) ApplySnapshot(snapshot *evercore.Snapshot) error {
	return json.Unmarshal([]byte(snapshot.State), &s.Settings)
}

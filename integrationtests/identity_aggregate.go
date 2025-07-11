package integrationtests

import (
	"context"
	"testing"
	"time"

	"github.com/kernelplex/evercore/base"
	"github.com/kernelplex/evercore/integrationtests/settings"
	"github.com/kernelplex/evercore/integrationtests/settings/settingsevents"
	"github.com/stretchr/testify/assert"
)

var systemSettingsAggregateType = "system_settings"

func (s *IntegrationTestSuite) loadIdentitySystemSettings(t *testing.T) {
	ctx := context.Background()
	eventStore := s.eventStore
	aggregate := settings.NewSystemSettingsAggregate()
	created := false
	err := eventStore.WithContext(ctx, func(etx evercore.EventStoreContext) error {
		var err error
		created, err = etx.LoadOrCreateAggregate(aggregate, systemSettingsAggregateType)
		if err != nil {
			t.Errorf("Failed to load user: %v", err)
			return err
		}
		var event = settingsevents.SettingUpdatedEvent{
			Key:   SettingsAdminEmail,
			Value: sampleUser.Email,
		}
		err = etx.ApplyEventTo(aggregate, event, time.Now(), "test_suite")
		if err != nil {
			t.Errorf("Failed to apply event: %v", err)
			return err
		}

		return nil
	})
	if err != nil {
		t.Errorf("Failed to load identity system settings: %v", err)
	}
	assert.True(t, created, "identity system settings should have been created")
	assert.Equal(t, sampleUser.Email, aggregate.Settings[SettingsAdminEmail])
}

func (s *IntegrationTestSuite) reloadIdentitySystemSettings(t *testing.T) {
	ctx := context.Background()

	// First create/update the settings using write context
	err := s.eventStore.WithContext(ctx, func(etx evercore.EventStoreContext) error {
		aggregate := settings.NewSystemSettingsAggregate()
		created, err := etx.LoadOrCreateAggregate(aggregate, systemSettingsAggregateType)
		if err != nil {
			return err
		}
		assert.Equal(t, sampleUser.Email, aggregate.Settings[SettingsAdminEmail])
		assert.False(t, created)

		return nil
	})
	assert.NoError(t, err)
}

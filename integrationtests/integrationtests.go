package integrationtests

import (
	"context"
	"testing"
	"time"

	evercore "github.com/kernelplex/evercore/base"
	_ "github.com/kernelplex/evercore/integrationtests/generated"
	"github.com/kernelplex/evercore/integrationtests/user"
	"github.com/stretchr/testify/assert"
)

type IntegrationTestSuite struct {
	storageEngine  evercore.StorageEngine
	eventStore     *evercore.EventStore
	existingUserId int64
}

var sampleUser = user.UserState{
	Email:        "chavez@example.com",
	PasswordHash: "3c4a8a2f9d5e3e5f4d6c7b8a9",
	FirstName:    "Juan",
	LastName:     "Chavez",
	DisplayName:  "Juan Chavez",
}

func NewIntegrationTestSuite(storageEngine evercore.StorageEngine) *IntegrationTestSuite {
	eventStore := evercore.NewEventStore(storageEngine)
	return &IntegrationTestSuite{
		storageEngine: storageEngine,
		eventStore:    eventStore,
	}
}

func (s *IntegrationTestSuite) RunTests(t *testing.T) {
	t.Run("Create a new user", s.createNewUser)
	t.Run("Update existing user", s.updateExistingUser)
	t.Run("Create user with existing email fails", s.createNewUserWithExistingEmail)
}

func (s *IntegrationTestSuite) createNewUser(t *testing.T) {
	ctx := context.Background()
	eventStore := s.eventStore

	uat, err := evercore.InContext(ctx, eventStore, func(etx evercore.EventStoreContext) (*user.UserAggregate, error) {

		aggregate := user.UserAggregate{}
		err := etx.CreateAggregateWithKeyInto(&aggregate, sampleUser.Email)

		if err != nil {
			t.Errorf("Failed to create user: %v", err)
			return nil, err
		}

		event := evercore.NewStateEvent(user.UserCreatedEvent{
			Email:        &sampleUser.Email,
			PasswordHash: &sampleUser.PasswordHash,
			FirstName:    &sampleUser.FirstName,
			LastName:     &sampleUser.LastName,
			DisplayName:  &sampleUser.DisplayName,
		})
		err = etx.ApplyEventTo(&aggregate, event, time.Now(), "test_suite")

		if err != nil {
			t.Errorf("Failed to apply event: %v", err)
			return nil, err
		}

		return &aggregate, nil
	})

	assert.NoError(t, err)
	assert.Greater(t, uat.GetId(), (int64)(0))
	assert.Equal(t, uat.State.Email, sampleUser.Email)
	assert.Equal(t, uat.State.FirstName, sampleUser.FirstName)
	assert.Equal(t, uat.State.LastName, sampleUser.LastName)
	assert.Equal(t, uat.State.DisplayName, sampleUser.DisplayName)
	s.existingUserId = uat.GetId()
}

func (s *IntegrationTestSuite) updateExistingUser(t *testing.T) {
	ctx := context.Background()
	eventStore := s.eventStore

	newFirstName := "Carlos"
	newDisplayName := "Carlos Chavez"

	uat, err := evercore.InContext(ctx, eventStore, func(etx evercore.EventStoreContext) (*user.UserAggregate, error) {
		aggregate := user.UserAggregate{}

		// Load existing user
		err := etx.LoadStateInto(&aggregate, s.existingUserId)
		if err != nil {
			t.Errorf("Failed to load user: %v", err)
			return nil, err
		}

		// Apply update event
		event := evercore.NewStateEvent(user.UserUpdatedEvent{
			FirstName:   &newFirstName,
			DisplayName: &newDisplayName,
		})
		err = etx.ApplyEventTo(&aggregate, event, time.Now(), "test_suite")
		if err != nil {
			t.Errorf("Failed to apply update event: %v", err)
			return nil, err
		}

		return &aggregate, nil
	})

	assert.NoError(t, err)
	assert.Equal(t, s.existingUserId, uat.GetId())
	assert.Equal(t, newFirstName, uat.State.FirstName)
	assert.Equal(t, newDisplayName, uat.State.DisplayName)
	// Verify unchanged fields remain the same
	assert.Equal(t, sampleUser.LastName, uat.State.LastName)
	assert.Equal(t, sampleUser.Email, uat.State.Email)
	assert.Equal(t, sampleUser.PasswordHash, uat.State.PasswordHash)
}

func (s *IntegrationTestSuite) createNewUserWithExistingEmail(t *testing.T) {
	ctx := context.Background()
	eventStore := s.eventStore

	_, err := evercore.InContext(ctx, eventStore, func(etx evercore.EventStoreContext) (*user.UserAggregate, error) {
		aggregate := user.UserAggregate{}
		err := etx.CreateAggregateWithKeyInto(&aggregate, sampleUser.Email)

		if err == nil {
			t.Error("Expected error when creating user with existing email, but got none")
			return nil, nil
		}

		return nil, err
	})

	var storageErr *evercore.StorageEngineError
	assert.ErrorAs(t, err, &storageErr)

	// Log the type of the error
	assert.Equal(t, evercore.ErrorTypeConstraintViolation, storageErr.ErrorType)
}

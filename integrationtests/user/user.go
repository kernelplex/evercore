package user

import (
	"github.com/kernelplex/evercore/base"
)

type UserState struct {
	Email                 string  `json:"email"`
	PasswordHash          string  `json:"passwordHash"`
	FirstName             string  `json:"firstName"`
	LastName              string  `json:"lastName"`
	DisplayName           string  `json:"displayName"`
	ResetToken            *string `json:"resetToken,omitempty"`
	LastLogin             int64   `json:"lastLogin,omitempty"`
	LastLoginAttempt      int64   `json:"lastLoginAttempt,omitempty"`
	FailedLoginAttempts   int64   `json:"failedLoginAttempts,omitempty"`
	TwoFactorSharedSecret *string `json:"twoFactorSharedSecret,omitempty"`
	Roles                 []int64 `json:"roles,omitempty"`
}

// evercore:aggregate
type UserAggregate struct {
	evercore.StateAggregate[UserState]
}

// evercore:state-event
type UserCreatedEvent struct {
	Email        *string `json:"email"`
	PasswordHash *string `json:"passwordHash"`
	FirstName    *string `json:"firstName"`
	LastName     *string `json:"lastName"`
	DisplayName  *string `json:"displayName"`
}

// evercore:state-event
type UserUpdatedEvent struct {
	PasswordHash *string `json:"passwordHash"`
	FirstName    *string `json:"firstName"`
	LastName     *string `json:"lastName"`
	DisplayName  *string `json:"displayName"`
}

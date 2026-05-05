package common

import (
	"encoding/base64"
	"fmt"
)

type Base64Key [SymmetricKeyLen]byte

var EnvironmentConfig Environment //nolint:gochecknoglobals

type Environment struct {
	MaxRequests        int        `env:"MAX_REQUESTS" envDefault:"0"`
	CurrentEnvironment string     `env:"ENVIRONMENT" envDefault:"production"`
	Database           string     `env:"DATABASE_DSN"`
	PasetoKey          *Base64Key `env:"PASETO_KEY"`

	// WebhookDebounceMS is the delay in milliseconds before a
	// webhook is dispatched after the last write. Set to 0 to disable
	// debouncing entirely. Note: debouncing is per replica.
	WebhookDebounceMS int `env:"WEBHOOK_DEBOUNCE_MS" envDefault:"1000"`

	// WebhookDebounceMaxMS is the hard cap in milliseconds on how long a
	// webhook can be deferred by repeated resets of the debounce timer.
	// Set to 0 to disable the cap. Ignored when WebhookDebounceMS is 0.
	WebhookDebounceMaxMS int `env:"WEBHOOK_DEBOUNCE_MAX_MS" envDefault:"10000"`
}

func (k *Base64Key) UnmarshalText(text []byte) error {
	if len(text) == 0 {
		return nil
	}

	key, err := base64.StdEncoding.DecodeString(string(text))
	if err != nil {
		return fmt.Errorf("can't base64-decode PASETO_KEY environment variable: %w", err)
	}

	if len(key) != SymmetricKeyLen {
		return ErrKeyLen
	}

	*k = *(*[SymmetricKeyLen]byte)(key)

	return nil
}

func (e *Environment) IsTest() bool {
	return e.CurrentEnvironment == "test"
}

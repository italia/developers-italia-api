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

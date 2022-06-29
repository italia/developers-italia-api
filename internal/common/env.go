package common

type Environment struct {
	MaxRequests        int    `env:"MAX_REQUESTS" envDefault:"20"`
	CurrentEnvironment string `env:"ENVIRONMENT" envDefault:"local"`
	Database           string `env:"DATABASE_DSN"`
}

func (e *Environment) IsTest() bool {
	return e.CurrentEnvironment == "test"
}

package log

import (
	"time"

	"github.com/mustafaturan/sser/internal/servicer/config"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type (
	Params struct {
		Config config.Servicer
	}

	Servicer interface {
	}

	servicer struct {
	}
)

func New(p Params) (Servicer, error) {
	zerolog.TimestampFunc = func() time.Time {
		return time.Now().UTC()
	}
	zerolog.DurationFieldUnit = time.Millisecond
	log.Logger = log.With().
		Str("name", p.Config.App()).
		Str("version", p.Config.Version()).
		Str("env", p.Config.Env()).
		Logger()

	return &servicer{}, nil
}

package idgen

import (
	"math/rand"
	"regexp"
	"time"

	"github.com/hasmcp/sser/internal/servicer/config"
	"github.com/mustafaturan/monoflake"
	zlog "github.com/rs/zerolog/log"
)

type (
	Params struct {
		Config config.Servicer
	}

	idgenConfig struct {
		Node               uint16 `yaml:"node"`
		EpochTimeInSeconds int64  `yaml:"epochTimeInSeconds"`
		NodeBits           int    `yaml:"nodeBits"`
	}

	Servicer interface {
		Next() int64
		NextString() string
		ValidStringID(string) bool
	}

	servicer struct {
		monoflake *monoflake.MonoFlake
	}
)

const (
	_logPrefix = "[idgen] "

	cfgKey  = "idgen"
	pattern = "^[0-9a-zA-Z]{11}$"
)

var (
	regex = regexp.MustCompile(pattern)
)

// New inits a new id generator
func New(p Params) (Servicer, error) {
	var cfg idgenConfig
	if err := p.Config.Populate(cfgKey, &cfg); err != nil {
		return nil, err
	}

	if cfg.Node == 0 {
		cfg.Node = uint16(rand.Intn(1 << 8))
		zlog.Info().Uint16("node", uint16(cfg.Node)).Msg(_logPrefix + "node id is set randomly")
	}

	epoch := time.Unix(cfg.EpochTimeInSeconds, 0)
	f, err := monoflake.New(cfg.Node, monoflake.WithEpoch(epoch), monoflake.WithNodeBits(cfg.NodeBits))
	if err != nil {
		zlog.Error().Str("epoch", epoch.Format(time.RFC3339)).Err(err).Msg(_logPrefix + "failed to init monoflake")
		return nil, err
	}
	zlog.Info().Any("monoflake.cfg", cfg).Msg(_logPrefix + "node is initialized")

	return &servicer{
		monoflake: f,
	}, nil
}

func (s *servicer) Next() int64 {
	return s.monoflake.Next().Int64()
}

func (s *servicer) NextString() string {
	return s.monoflake.Next().String()
}

func (s *servicer) ValidStringID(id string) bool {
	return regex.Match([]byte(id))
}

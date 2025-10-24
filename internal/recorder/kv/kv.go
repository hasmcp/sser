package kv

import (
	"context"
	"errors"
	"time"

	"github.com/mustafaturan/sser/internal/servicer/config"
	zlog "github.com/rs/zerolog/log"
	"go.etcd.io/bbolt"
)

type (
	Recorder interface {
		ListKeys(ctx context.Context) ([][]byte, error)
		Get(ctx context.Context, key []byte) ([]byte, error)
		Set(ctx context.Context, key, val []byte) error
		Delete(ctx context.Context, key []byte) error
		Close() error
	}

	recorder struct {
		db *bbolt.DB
	}

	Params struct {
		Config config.Servicer
	}

	bboltCfg struct {
		Enabled bool   `yaml:"enabled"`
		DSN     string `yaml:"dsn"`
	}

	err string
)

const (
	cfgKey = "kv"

	logPrefix = "[kv] "

	ErrNotEnabled err = "kv is not enabled"
	ErrNotFound   err = "not found"
)

var (
	_defaultBucket = []byte("_d")
)

func New(p Params) (Recorder, error) {
	var cfg bboltCfg
	err := p.Config.Populate(cfgKey, &cfg)
	if err != nil {
		return nil, err
	}

	if !cfg.Enabled {
		return nil, ErrNotEnabled
	}

	db, err := bbolt.Open(cfg.DSN, 0600, &bbolt.Options{
		Timeout: time.Second,
	})
	if err != nil {
		return nil, err
	}

	err = db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(_defaultBucket)
		if b == nil {
			var err error
			b, err = tx.CreateBucketIfNotExists(_defaultBucket)
			if err != nil {
				return err
			}
		}
		if b == nil {
			return errors.New("bucket is nil")
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	zlog.Info().Msg(logPrefix + "initialized")

	return &recorder{db: db}, nil
}

func (r *recorder) ListKeys(ctx context.Context) ([][]byte, error) {
	if r == nil {
		return nil, ErrNotEnabled
	}
	var keys [][]byte
	err := r.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(_defaultBucket)
		c := b.Cursor()

		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			keys = append(keys, k)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return keys, nil
}

func (r *recorder) Get(ctx context.Context, key []byte) ([]byte, error) {
	if r == nil {
		return nil, ErrNotFound
	}
	var val []byte
	err := r.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(_defaultBucket)
		val = b.Get(key)
		return nil
	})
	if err != nil {
		return nil, err
	}
	if val == nil {
		return nil, ErrNotFound
	}
	return val, nil
}

func (r *recorder) Set(ctx context.Context, key, val []byte) error {
	if r == nil {
		return ErrNotEnabled
	}
	return r.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(_defaultBucket)
		return b.Put(key, val)
	})
}

func (r *recorder) Delete(ctx context.Context, key []byte) error {
	if r == nil {
		return ErrNotEnabled
	}
	return r.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(_defaultBucket)
		return b.Delete(key)
	})
}

func (r *recorder) Close() error {
	zlog.Info().Msg(logPrefix + "closing")
	return r.db.Close()
}

func (e err) Error() string {
	return string(e)
}

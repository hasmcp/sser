package app

import (
	"context"
	"errors"

	"github.com/mustafaturan/sser/internal/controller/pubsub"
	"github.com/mustafaturan/sser/internal/handler/http"
	"github.com/mustafaturan/sser/internal/recorder/kv"
	"github.com/mustafaturan/sser/internal/servicer/config"
	"github.com/mustafaturan/sser/internal/servicer/idgen"
	"github.com/mustafaturan/sser/internal/servicer/log"
	"github.com/mustafaturan/sser/internal/servicer/server"
)

type (
	App struct {
		Config config.Servicer
		Log    log.Servicer
		Server server.Servicer
		KV     kv.Recorder
	}
)

func New() (*App, error) {
	config, err := config.New()
	if err != nil {
		return nil, err
	}

	log, err := log.New(log.Params{
		Config: config,
	})
	if err != nil {
		return nil, err
	}

	idgen, err := idgen.New(idgen.Params{
		Config: config,
	})
	if err != nil {
		return nil, err
	}

	kvrecorder, err := kv.New(kv.Params{
		Config: config,
	})
	if err != nil && !errors.Is(err, kv.ErrNotEnabled) {
		return nil, err
	}

	pubsub, err := pubsub.New(pubsub.Params{
		Config: config,
		IDGen:  idgen,
		KV:     kvrecorder,
	})
	if err != nil {
		return nil, err
	}

	httpHandler, err := http.New(http.Params{
		PubSub: pubsub,
	})
	if err != nil {
		return nil, err
	}

	server, err := server.New(server.Params{
		Config:  config,
		Handler: httpHandler.Handle,
	})
	if err != nil {
		return nil, err
	}

	return &App{
		Config: config,
		Log:    log,
		Server: server,
		KV:     kvrecorder,
	}, nil
}

func (a *App) Start(ctx context.Context) error {
	err := a.Server.ListenAndServe()
	if err != nil {
		return err
	}
	return nil
}

func (a *App) Stop(ctx context.Context) error {
	err := a.Server.Shutdown()
	if err != nil {
		return err
	}

	if a.KV != nil {
		if err := a.KV.Close(); err != nil {
			return err
		}
	}
	return nil
}

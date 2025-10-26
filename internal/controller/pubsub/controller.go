package pubsub

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/hasmcp/sser/internal/_data/entity"
	"github.com/hasmcp/sser/internal/recorder/kv"
	"github.com/hasmcp/sser/internal/servicer/config"
	"github.com/hasmcp/sser/internal/servicer/idgen"
	"github.com/mustafaturan/monoflake"
	zlog "github.com/rs/zerolog/log"
)

type (
	Controller interface {
		Create(ctx context.Context, req entity.CreatePubSubRequest) (*entity.CreatePubSubResponse, error)
		Delete(ctx context.Context, req entity.DeletePubSubRequest) error
		Publish(ctx context.Context, req entity.PublishRequest) (*entity.PublishResponse, error)
		Subscribe(ctx context.Context, req entity.SubscribeRequest) (*entity.SubscribeResponse, error)
		Unsubscribe(ctx context.Context, req entity.UnsubscribeRequest) error
		GetMetrics(ctx context.Context, req entity.GetMetricsRequest) (*entity.GetMetricsResponse, error)
	}

	controller struct {
		cfg     pubsubConfig
		idgen   idgen.Servicer
		kv      kv.Recorder
		pubsubs sync.Map
		metrics *metrics
	}

	Params struct {
		Config config.Servicer
		IDGen  idgen.Servicer
		KV     kv.Recorder
	}

	pubsub struct {
		id          int64
		static      bool
		subscribers []subscriber
		mutex       sync.RWMutex
		token       []byte
	}

	subscriber struct {
		channel chan []byte
		id      int64
	}

	pubsubConfig struct {
		ApiAccessToken                    string               `yaml:"apiAccessToken"`
		MetricsAccessToken                string               `yaml:"metricsAccessToken"`
		MaxDurationForSubscriberToReceive time.Duration        `yaml:"maxDurationForSubscriberToReceive"`
		TickFrequency                     time.Duration        `yaml:"tickFrequency"`
		StaticPubSubs                     []StaticPubSubConfig `yaml:"staticPubSubs"`
	}

	StaticPubSubConfig struct {
		ID    int64  `yaml:"id"`
		Name  string `yaml:"name"`
		Token string `yaml:"token"`
	}
)

const (
	cfgKey = "pubsub"

	logPrefix = "[pubsubctrl] "
)

func New(p Params) (Controller, error) {
	var cfg pubsubConfig
	err := p.Config.Populate(cfgKey, &cfg)
	if err != nil {
		return nil, err
	}

	c := &controller{
		cfg:     cfg,
		idgen:   p.IDGen,
		kv:      p.KV,
		pubsubs: sync.Map{},
		metrics: newMetrics(),
	}

	err = c.registerStaticPubSubs()
	if err != nil {
		return nil, err
	}

	err = c.registerPersistentPubSubs()
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (c *controller) Create(ctx context.Context, req entity.CreatePubSubRequest) (*entity.CreatePubSubResponse, error) {
	if req.ApiAccessToken != c.cfg.ApiAccessToken {
		return nil, entity.Err{
			Code:    401,
			Message: "API access token mismatch",
			Details: map[string]any{
				"token": req.ApiAccessToken,
			},
		}
	}

	defer c.inc(metricTopics)
	defer c.inc(metricActiveTopics)

	id := c.idgen.Next()

	token, err := generateRandom64()
	if err != nil {
		return nil, entity.Err{
			Code:    500,
			Message: "Couldn't generate random token",
			Details: map[string]any{
				"err": err.Error(),
			},
		}
	}

	if req.Persist {
		if c.kv == nil {
			return nil, entity.Err{
				Code:    400,
				Message: "Persistent store is not available",
			}
		}

		err := c.kv.Set(ctx, monoflake.ID(id).BigEndianBytes(), []byte(token))
		if err != nil {
			return nil, entity.Err{
				Code:    500,
				Message: "Couldn't persist to store",
				Details: map[string]any{
					"err": err.Error(),
				},
			}
		}
	}

	c.pubsubs.Store(id, &pubsub{
		id:          id,
		subscribers: make([]subscriber, 0, 1),
		mutex:       sync.RWMutex{},
		token:       []byte(token),
	})

	return &entity.CreatePubSubResponse{
		ID:    id,
		Token: []byte(token),
	}, nil
}

func (c *controller) Delete(ctx context.Context, req entity.DeletePubSubRequest) error {
	if req.ApiAccessToken != c.cfg.ApiAccessToken {
		return entity.Err{
			Code:    401,
			Message: "API access token mismatch",
			Details: map[string]any{
				"token": req.ApiAccessToken,
			},
		}
	}

	t, ok := c.pubsubs.Load(req.ID)
	if !ok {
		return nil
	}
	pubsub, ok := t.(*pubsub)
	if !ok {
		return entity.Err{
			Code:    500,
			Message: "malformed pubsub type",
			Details: map[string]any{
				"id": req.ID,
			},
		}
	}

	if pubsub.static {
		return entity.Err{
			Code:    400,
			Message: "static pubsubs can't be deleted",
			Details: map[string]any{
				"id": req.ID,
			},
		}
	}

	if c.kv != nil {
		err := c.kv.Delete(context.Background(), monoflake.ID(req.ID).BigEndianBytes())
		if err != nil {
			return entity.Err{
				Code:    500,
				Message: "Couldn't delete the pubsub from storage",
				Details: map[string]any{
					"id": req.ID,
				},
			}
		}
	}

	defer c.dec(metricActiveTopics)

	pubsub.mutex.Lock()
	for _, s := range pubsub.subscribers {
		close(s.channel)
	}
	c.pubsubs.Delete(req.ID)
	pubsub.mutex.Unlock()
	return nil
}

func (c *controller) Publish(ctx context.Context, req entity.PublishRequest) (*entity.PublishResponse, error) {
	if req.ApiAccessToken != c.cfg.ApiAccessToken {
		return nil, entity.Err{
			Code:    401,
			Message: "API access token mismatch",
			Details: map[string]any{
				"token": req.ApiAccessToken,
			},
		}
	}

	cnt, err := c.publish(req.PubSubID, req.Message)
	if err != nil {
		return nil, err
	}
	defer c.inc(metricMessageReceived)
	defer c.incBy(metricMessageSent, int64(cnt))

	return &entity.PublishResponse{
		ID: c.idgen.Next(),
	}, nil
}

func (c *controller) Subscribe(ctx context.Context, req entity.SubscribeRequest) (*entity.SubscribeResponse, error) {
	t, ok := c.pubsubs.Load(req.PubSubID)
	if !ok {
		return nil, entity.Err{
			Code:    404,
			Message: "pubsub not found",
			Details: map[string]any{
				"id": req.PubSubID,
			},
		}
	}

	pubsub, ok := t.(*pubsub)
	if !ok {
		return nil, entity.Err{
			Code:    500,
			Message: "malformed pubsub",
			Details: map[string]any{
				"id": req.PubSubID,
			},
		}
	}

	if !bytes.Equal(pubsub.token, req.Token) {
		return nil, entity.Err{
			Code:    401,
			Message: "token mismatch for the pubsub",
			Details: map[string]any{
				"token": string(req.Token),
			},
		}
	}

	id := c.idgen.Next()

	subscriber := subscriber{
		channel: make(chan []byte),
		id:      id,
	}

	pubsub.mutex.Lock()
	pubsub.subscribers = append(pubsub.subscribers, subscriber)
	pubsub.mutex.Unlock()

	defer c.inc(metricActiveSubscribers)
	defer c.inc(metricSubscribers)

	return &entity.SubscribeResponse{
		ID:            subscriber.id,
		Events:        subscriber.channel,
		TickFrequency: c.cfg.TickFrequency,
	}, nil
}

func (c *controller) Unsubscribe(ctx context.Context, req entity.UnsubscribeRequest) error {
	t, ok := c.pubsubs.Load(req.PubSubID)
	if !ok {
		return entity.Err{
			Code:    404,
			Message: "pubsub not found",
			Details: map[string]any{
				"id": req.PubSubID,
			},
		}
	}

	pubsub, ok := t.(*pubsub)
	if !ok {
		return entity.Err{
			Code:    500,
			Message: "malformed pubsub",
			Details: map[string]any{
				"id": req.PubSubID,
			},
		}
	}

	if !bytes.Equal(pubsub.token, req.Token) {
		return entity.Err{
			Code:    401,
			Message: "token mismatch for the pubsub",
			Details: map[string]any{
				"token": string(req.Token[:]),
			},
		}
	}

	pubsub.mutex.Lock()
	for i := 0; i < len(pubsub.subscribers); i++ {
		if pubsub.subscribers[i].id == req.ID {
			pubsub.subscribers[i], pubsub.subscribers[len(pubsub.subscribers)-1] = pubsub.subscribers[len(pubsub.subscribers)-1], pubsub.subscribers[i]
			pubsub.subscribers = pubsub.subscribers[0 : len(pubsub.subscribers)-1]
			break
		}
	}
	pubsub.mutex.Unlock()
	defer c.dec(metricActiveSubscribers)
	return nil
}

func (c *controller) GetMetrics(ctx context.Context, req entity.GetMetricsRequest) (*entity.GetMetricsResponse, error) {
	if req.MetricsAccessToken != c.cfg.MetricsAccessToken {
		return nil, entity.Err{
			Code:    401,
			Message: "API access token mismatch",
			Details: map[string]any{
				"token": req.MetricsAccessToken,
			},
		}
	}

	metrics := make([]entity.Metric, 0, len(c.metrics.vals))
	for k := range c.metrics.vals {
		metrics = append(metrics, entity.Metric{
			Name:  k.String(),
			Value: float64(c.get(k)),
		})
	}

	return &entity.GetMetricsResponse{
		Metrics: metrics,
	}, nil
}

func (c *controller) registerPersistentPubSubs() error {
	if c.kv == nil {
		zlog.Warn().Msg(logPrefix + "persistant storage is not available, skipping loads")
		return nil
	}

	keys, err := c.kv.ListKeys(context.Background())
	if err != nil {
		return err
	}
	ctx := context.Background()
	cnt := int64(0)
	for _, k := range keys {
		id := monoflake.IDFromBigEndianBytes(k).Int64()
		token, err := c.kv.Get(ctx, k)
		if err != nil {
			zlog.Error().Err(err).Int64("id", id).Msg(logPrefix + "failed to load pubsub from storage; going on with the next one.")
			continue
		}
		c.pubsubs.Store(id, &pubsub{
			id:          id,
			subscribers: make([]subscriber, 0),
			mutex:       sync.RWMutex{},
			token:       token,
		})
		cnt++
	}
	c.incBy(metricTopics, cnt)
	c.incBy(metricActiveTopics, cnt)
	return nil
}

func (c *controller) registerStaticPubSubs() error {
	// it is used for publishing system metrics (do not override!)
	c.pubsubs.Store(int64(0), &pubsub{
		id:          0, // reserved id
		static:      true,
		subscribers: make([]subscriber, 0),
		mutex:       sync.RWMutex{},
		token:       []byte(c.cfg.MetricsAccessToken),
	})

	for _, ps := range c.cfg.StaticPubSubs {
		if ps.ID == 0 {
			return fmt.Errorf("[pubsub] id for static token must be >= 1 (name: %s)", ps.Name)
		}

		token := []byte(ps.Token)
		if len(token) < 1 {
			return fmt.Errorf("[pubsub] token must be >= 1 chars (name: %s)", ps.Name)
		}
		c.pubsubs.Store(ps.ID, &pubsub{
			id:          ps.ID,
			static:      true,
			subscribers: make([]subscriber, 0),
			mutex:       sync.RWMutex{},
			token:       []byte(token),
		})
	}

	c.incBy(metricTopics, int64(len(c.cfg.StaticPubSubs)+1))
	c.incBy(metricActiveTopics, int64(len(c.cfg.StaticPubSubs)+1))
	c.incBy(metricStaticTopics, int64(len(c.cfg.StaticPubSubs)+1))
	return nil
}

func (c *controller) publish(id int64, msg []byte) (int, error) {
	t, ok := c.pubsubs.Load(id)
	if !ok {
		return 0, entity.Err{
			Code:    404,
			Message: "pubsub not found",
			Details: map[string]any{
				"id": id,
			},
		}
	}

	pubsub, ok := t.(*pubsub)
	if !ok {
		return 0, entity.Err{
			Code:    500,
			Message: "malformed pubsub, please create another pubsub",
			Details: map[string]any{
				"id": id,
			},
		}
	}

	pubsub.mutex.RLock()
	subscribers := pubsub.subscribers
	pubsub.mutex.RUnlock()

	go func(msg []byte, subscribers []subscriber) {
		timeoutDuration := c.cfg.MaxDurationForSubscriberToReceive
		wg := sync.WaitGroup{}
		for _, s := range subscribers {
			wg.Add(1)
			go func(ch chan []byte) {
				defer wg.Done()
				err := publishWithTimeout(ch, msg, timeoutDuration)
				if err != nil {
					zlog.Error().Err(err).Dur("timeout", timeoutDuration).
						Msg(logPrefix + "failed to send message to subscriber within the given timeout duration")
				}
			}(s.channel)
		}
		wg.Wait()
	}(msg, subscribers)

	return len(subscribers), nil
}

func (c *controller) inc(k metric) {
	msg := fmt.Sprintf(`{"val": 1, "metric": "%s"}`, k.String())
	_, _ = c.publish(0, []byte(msg))
	c.metrics.inc(k)
}

func (c *controller) incBy(k metric, val int64) {
	msg := fmt.Sprintf(`{"val": %d, "metric": "%s"}`, val, k.String())
	_, _ = c.publish(0, []byte(msg))
	c.metrics.incBy(k, val)
}

func (c *controller) dec(k metric) {
	msg := fmt.Sprintf(`{"val": -1, "metric": "%s"}`, k.String())
	_, _ = c.publish(0, []byte(msg))
	c.metrics.dec(k)
}

func (c *controller) get(k metric) int64 {
	return c.metrics.get(k)
}

// independent functions

func generateRandom64() (string, error) {
	b := make([]byte, 64)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	num := new(big.Int).SetBytes(b)
	return num.Text(62)[:64], nil
}

func publishWithTimeout(ch chan []byte, msg []byte, timeout time.Duration) error {
	select {
	case ch <- msg:
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("send to channel timed out after %v", timeout)
	}
}

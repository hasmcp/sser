package entity

import "time"

type (
	CreatePubSubRequest struct {
		ApiAccessToken string
		Persist        bool
	}

	CreatePubSubResponse struct {
		ID    int64
		Token []byte
	}

	DeletePubSubRequest struct {
		ApiAccessToken string
		ID             int64
	}

	PublishRequest struct {
		ApiAccessToken string
		PubSubID       int64
		EventID        string
		EventType      string
		Message        []byte
	}

	PublishResponse struct {
		ID int64
	}

	SubscribeRequest struct {
		PubSubID int64
		Token    []byte
	}

	SubscribeResponse struct {
		ID            int64
		Events        chan *Event
		TickFrequency time.Duration
	}

	UnsubscribeRequest struct {
		PubSubID int64
		ID       int64
		Token    []byte
	}

	GetMetricsRequest struct {
		MetricsAccessToken string
	}

	GetMetricsResponse struct {
		Metrics []Metric
	}

	Metric struct {
		Name  string
		Value float64
	}

	Event struct {
		ID   string
		Type string
		Data []byte
	}
)

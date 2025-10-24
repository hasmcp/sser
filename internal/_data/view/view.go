package view

type (
	CreatePubSubRequest struct {
		Persist bool `yaml:"persist"`
	}

	CreatePubSubResponse struct {
		ID    string `json:"id"`
		Token string `json:"token"`
	}

	PublishRequest struct {
		Message string `json:"message"`
	}

	PublishResponse struct {
		ID string `json:"id"`
	}

	SubscribeRequest struct {
		Token string `json:"token"`
	}

	GetMetricsResponse struct {
		Metrics []Metric `json:"metrics"`
	}

	Metric struct {
		Name  string  `json:"name"`
		Value float64 `json:"value"`
	}
)

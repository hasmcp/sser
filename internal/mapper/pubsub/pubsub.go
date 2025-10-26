package pubsub

import (
	"encoding/json"
	"strings"

	"github.com/hasmcp/sser/internal/_data/entity"
	"github.com/hasmcp/sser/internal/_data/view"
	"github.com/mustafaturan/monoflake"
	"github.com/valyala/fasthttp"
)

const (
	payloadPubSubNamespace      string = "pubsub"
	payloadPubSubEventNamespace string = "event"
)

func FromHttpRequestToCreatePubSubRequest(ctx *fasthttp.RequestCtx) *entity.CreatePubSubRequest {
	var req map[string]view.CreatePubSubRequest

	err := json.Unmarshal(ctx.Request.Body(), &req)
	if err != nil {
		return nil
	}
	return &entity.CreatePubSubRequest{
		ApiAccessToken: fromHttpRequestToAccessToken(ctx),
		Persist:        req[payloadPubSubNamespace].Persist,
	}
}

func FromCreatePubSubResponseToHttpResponse(res entity.CreatePubSubResponse) []byte {
	payload := map[string]view.CreatePubSubResponse{
		payloadPubSubNamespace: {
			ID:    monoflake.ID(res.ID).String(),
			Token: string(res.Token[:]),
		},
	}

	data, _ := json.Marshal(payload)
	return data
}

func FromHttpRequestToDeletePubSubRequest(ctx *fasthttp.RequestCtx) *entity.DeletePubSubRequest {
	return &entity.DeletePubSubRequest{
		ApiAccessToken: fromHttpRequestToAccessToken(ctx),
		ID:             fromHttpRequestToPubSubID(ctx),
	}
}

func FromHttpRequestToPublishRequest(ctx *fasthttp.RequestCtx) *entity.PublishRequest {
	id := fromHttpRequestToPubSubID(ctx)
	var req map[string]view.PublishRequest

	err := json.Unmarshal(ctx.Request.Body(), &req)
	if err != nil {
		return nil
	}

	return &entity.PublishRequest{
		ApiAccessToken: fromHttpRequestToAccessToken(ctx),
		PubSubID:       id,
		Message:        []byte(req[payloadPubSubEventNamespace].Message),
	}
}

func FromPublishResponseToHttpResponse(res entity.PublishResponse) []byte {
	payload := map[string]view.PublishResponse{
		payloadPubSubEventNamespace: {
			ID: monoflake.ID(res.ID).String(),
		},
	}

	data, _ := json.Marshal(payload)
	return data
}

func FromHttpRequestToSubscribeRequest(ctx *fasthttp.RequestCtx) *entity.SubscribeRequest {
	id := fromHttpRequestToPubSubID(ctx)
	token := fromHttpRequestToAccessToken(ctx)
	if token == "" {
		token = string(ctx.QueryArgs().Peek("access_token"))
	}

	return &entity.SubscribeRequest{
		PubSubID: id,
		Token:    []byte(token),
	}
}

func FromHttpRequestToGetMetricsRequest(ctx *fasthttp.RequestCtx) *entity.GetMetricsRequest {
	return &entity.GetMetricsRequest{
		MetricsAccessToken: fromHttpRequestToAccessToken(ctx),
	}
}

func FromGetMetricsResponseToHttpResponse(res entity.GetMetricsResponse) []byte {
	metrics := make([]view.Metric, len(res.Metrics))
	for i, m := range res.Metrics {
		metrics[i] = fromMetricEntityMetricView(m)
	}

	payload := view.GetMetricsResponse{
		Metrics: metrics,
	}

	data, _ := json.Marshal(payload)
	return data
}

func fromMetricEntityMetricView(e entity.Metric) view.Metric {
	return view.Metric{
		Name:  e.Name,
		Value: e.Value,
	}
}

func fromHttpRequestToPubSubID(ctx *fasthttp.RequestCtx) int64 {
	path := string(ctx.Path())
	paths := strings.Split(path, "/")
	if len(paths) < 5 {
		return -1
	}
	id := paths[4]
	return monoflake.IDFromBase62(id).Int64()
}

func fromHttpRequestToAccessToken(ctx *fasthttp.RequestCtx) string {
	authorization := string(ctx.Request.Header.Peek("Authorization"))
	apiAccessToken := strings.Replace(authorization, "Bearer ", "", 1)
	return apiAccessToken
}

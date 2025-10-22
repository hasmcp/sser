package http

import (
	"bufio"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mustafaturan/sser/internal/_data/entity"
	"github.com/mustafaturan/sser/internal/controller/pubsub"
	errmapper "github.com/mustafaturan/sser/internal/mapper/err"
	pubsubmapper "github.com/mustafaturan/sser/internal/mapper/pubsub"
	zlog "github.com/rs/zerolog/log"
	"github.com/valyala/fasthttp"
)

type (
	Handler interface {
		Handle(ctx *fasthttp.RequestCtx)
	}

	handler struct {
		pubsub pubsub.Controller
	}

	Params struct {
		PubSub pubsub.Controller
	}
)

const (
	cfgKey = "http"
)

const (
	pathBase    string = "/api/v1"
	pathMetrics string = pathBase + "/metrics"
	pathPubSubs string = pathBase + "/pubsubs"
)

var (
	_httpPayloadInvalidRequest = []byte(`{"error": {"message":"Invalid request payload", "code":400}}`)
	_httpPayloadNotFound       = []byte(`{"error": {"code": 404, "message": "Not found"}}`)
)

func New(p Params) (Handler, error) {
	return &handler{
		pubsub: p.PubSub,
	}, nil
}

func (h *handler) Handle(ctx *fasthttp.RequestCtx) {
	path := string(ctx.Path())
	if path == "/" {
		fasthttp.ServeFile(ctx, "./public/index.html")
		return
	}
	if path == "/assets/chart.js" || path == "/assets/main.css" {
		fasthttp.ServeFile(ctx, "./public"+path)
		return
	}
	if strings.HasPrefix(path, pathPubSubs) {
		h.handlePubSub(ctx)
		return
	}
	if strings.HasPrefix(path, pathMetrics) {
		h.handleMetrics(ctx)
		return
	}
	notfound(ctx)
}

func notfound(ctx *fasthttp.RequestCtx) {
	ctx.SetConnectionClose()
	ctx.SetContentType("application/json")
	ctx.SetStatusCode(fasthttp.StatusNotFound)
	ctx.SetBody(_httpPayloadNotFound)
}

func badrequest(ctx *fasthttp.RequestCtx) {
	ctx.SetConnectionClose()
	ctx.SetContentType("application/json")
	ctx.SetStatusCode(fasthttp.StatusBadRequest)
	ctx.SetBody(_httpPayloadInvalidRequest)
}

func (h *handler) allowOrigin(ctx *fasthttp.RequestCtx) {
	origin := string(ctx.Request.Header.Peek("origin"))
	if origin == "" {
		origin = "*"
	}
	ctx.Response.Header.Set("access-control-allow-origin", origin)
	ctx.Response.Header.Set("access-control-allow-methods", "GET, POST, PUT, DELETE, OPTIONS")
	ctx.Response.Header.Set("access-control-allow-headers", "*")
	ctx.Response.Header.Set("access-control-allow-credentials", "true")
	ctx.Response.Header.Set("access-control-max-Age", "86400")
	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.Write([]byte{})
}

func (h *handler) handleMetrics(ctx *fasthttp.RequestCtx) {
	method := string(ctx.Method())
	path := string(ctx.Path())

	// Get /metrics
	if path == pathMetrics && method == fasthttp.MethodGet {
		h.getMetrics(ctx)
		return
	}

	notfound(ctx)
}

func (h *handler) handlePubSub(ctx *fasthttp.RequestCtx) {
	method := string(ctx.Method())
	path := string(ctx.Path())
	path = strings.Replace(path, pathPubSubs, "", -1)
	pathParts := strings.Split(path, "/")

	// POST /pubsubs
	if len(pathParts) == 1 {
		switch method {
		case fasthttp.MethodPost:
			h.createPubSub(ctx)
		default:
			notfound(ctx)
		}
		return
	}

	// DELETE /pubsubs/:id
	if len(pathParts) == 2 {
		switch method {
		case fasthttp.MethodDelete:
			h.deletePubSub(ctx)
		default:
			notfound(ctx)
		}
		return
	}

	// POST /pubsubs/:id/events
	// GET  /pubsubs/:id/events
	if len(pathParts) == 3 && method == fasthttp.MethodPost {
		switch pathParts[2] {
		case "events":
			h.publishToPubSub(ctx)
		default:
			notfound(ctx)
		}
		return
	}

	if len(pathParts) == 3 && method == fasthttp.MethodGet {
		switch pathParts[2] {
		case "events":
			h.subscribeToPubSub(ctx)
		default:
			notfound(ctx)
		}
		return
	}

	// OPTIONS /pubsubs/:id/events
	if len(pathParts) == 3 && pathParts[2] == "events" && method == fasthttp.MethodOptions {
		h.allowOrigin(ctx)
		return
	}

	notfound(ctx)
}

func (h *handler) createPubSub(ctx *fasthttp.RequestCtx) {
	req := pubsubmapper.FromHttpRequestToCreatePubSubRequest(ctx)
	if req == nil {
		badrequest(ctx)
		return
	}

	freshCtx := context.Background()
	res, err := h.pubsub.Create(freshCtx, *req)
	if err != nil {
		msg, code := errmapper.FromErrorToHttpResponse(err)
		ctx.SetStatusCode(code)
		ctx.SetBody(msg)
		return
	}

	body := pubsubmapper.FromCreatePubSubResponseToHttpResponse(*res)

	ctx.SetStatusCode(fasthttp.StatusCreated)
	ctx.SetBody(body)
}

func (h *handler) deletePubSub(ctx *fasthttp.RequestCtx) {
	req := pubsubmapper.FromHttpRequestToDeletePubSubRequest(ctx)
	if req == nil {
		badrequest(ctx)
		return
	}

	freshCtx := context.Background()
	err := h.pubsub.Delete(freshCtx, *req)
	if err != nil {
		msg, code := errmapper.FromErrorToHttpResponse(err)
		ctx.SetStatusCode(code)
		ctx.SetBody(msg)
		return
	}

	ctx.SetStatusCode(fasthttp.StatusNoContent)
	ctx.SetBody([]byte{})
}

func (h *handler) publishToPubSub(ctx *fasthttp.RequestCtx) {
	req := pubsubmapper.FromHttpRequestToPublishRequest(ctx)
	if req == nil {
		badrequest(ctx)
		return
	}

	freshCtx := context.Background()
	res, err := h.pubsub.Publish(freshCtx, *req)
	if err != nil {
		msg, code := errmapper.FromErrorToHttpResponse(err)
		ctx.SetStatusCode(code)
		ctx.SetBody(msg)
		return
	}

	body := pubsubmapper.FromPublishResponseToHttpResponse(*res)

	ctx.SetStatusCode(fasthttp.StatusCreated)
	ctx.SetBody(body)
}

func (h *handler) subscribeToPubSub(ctx *fasthttp.RequestCtx) {
	req := pubsubmapper.FromHttpRequestToSubscribeRequest(ctx)
	if req == nil {
		badrequest(ctx)
		return
	}

	freshCtx := context.Background()
	res, err := h.pubsub.Subscribe(freshCtx, *req)
	if err != nil {
		msg, code := errmapper.FromErrorToHttpResponse(err)
		ctx.SetStatusCode(code)
		ctx.SetBody(msg)
		return
	}

	origin := string(ctx.Request.Header.Peek("origin"))
	if origin == "" {
		origin = "*"
	}
	ctx.SetContentType("text/event-stream")
	ctx.SetConnectionClose()
	ctx.Response.Header.Set("cache-control", "no-cache")
	ctx.Response.Header.Set("connection", "keep-alive")
	ctx.Response.Header.Set("transfer-encoding", "chunked")
	ctx.Response.Header.Set("access-control-allow-origin", origin)
	ctx.Response.Header.Set("access-control-allow-headers", "cache-control")
	ctx.Response.Header.Set("access-control-allow-credentials", "true")

	ctx.SetBodyStreamWriter(fasthttp.StreamWriter(func(w *bufio.Writer) {
		zlog.Info().Int64("id", res.ID).Dur("tickFrequency", res.TickFrequency).Msg("sse conn opened by user")
		ticker := time.NewTicker(res.TickFrequency)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				zlog.Info().Int64("pubsubID", req.PubSubID).Int64("id", res.ID).Msg("sse conn closed by user")
				err := h.pubsub.Unsubscribe(freshCtx, entity.UnsubscribeRequest{
					PubSubID: req.PubSubID,
					ID:       res.ID,
					Token:    req.Token,
				})
				if err != nil {
					zlog.Warn().Err(err).Int64("pubsubID", req.PubSubID).Int64("id", res.ID).Msg("failed to unsubscribe from topic on ctx done")
				}
				return
			case <-ticker.C:
				fmt.Fprintf(w, "data: {\"status\": \"tick\"}\n\n")
				if err := w.Flush(); err != nil {
					zlog.Warn().Err(err).Int64("pubsubID", req.PubSubID).Msg("failed to flush on tick")
					err := h.pubsub.Unsubscribe(freshCtx, entity.UnsubscribeRequest{
						PubSubID: req.PubSubID,
						ID:       res.ID,
						Token:    req.Token,
					})
					if err != nil {
						zlog.Warn().Err(err).Int64("pubsubID", req.PubSubID).Int64("id", res.ID).Msg("failed to unsubscribe on tick flush failure")
					}
					return
				}
			case event, ok := <-res.Events:
				if !ok {
					zlog.Info().Int64("id", res.ID).Msg("sse conn closed")

					fmt.Fprintf(w, "data: {\"status\": \"closed\"}\n\n")
					if err := w.Flush(); err != nil {
						zlog.Warn().Err(err).Int64("pubsubID", req.PubSubID).Msg("failed to flush on closed event")
						return
					}
					return
				}
				fmt.Fprintf(w, "data: %s\n\n", string(event))
				if err := w.Flush(); err != nil {
					zlog.Error().Err(err).Int64("pubsubID", req.PubSubID).Msg("failed to flush on event")
					err := h.pubsub.Unsubscribe(freshCtx, entity.UnsubscribeRequest{
						PubSubID: req.PubSubID,
						ID:       res.ID,
						Token:    req.Token,
					})
					if err != nil {
						zlog.Warn().Err(err).Int64("pubsubID", req.PubSubID).Int64("id", res.ID).Msg("failed to unsubscribe on message flush failure")
					}
					return
				}
			}
		}
	}))
}

func (h *handler) getMetrics(ctx *fasthttp.RequestCtx) {
	req := pubsubmapper.FromHttpRequestToGetMetricsRequest(ctx)
	if req == nil {
		badrequest(ctx)
		return
	}

	freshCtx := context.Background()
	res, err := h.pubsub.GetMetrics(freshCtx, *req)
	if err != nil {
		msg, code := errmapper.FromErrorToHttpResponse(err)
		ctx.SetStatusCode(code)
		ctx.SetBody(msg)
		return
	}

	body := pubsubmapper.FromGetMetricsResponseToHttpResponse(*res)

	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetBody(body)
}

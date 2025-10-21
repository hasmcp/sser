package pubsub

import "sync/atomic"

type (
	metrics struct {
		vals map[metric]*int64
	}

	metric uint8
)

const (
	metricInvalid metric = iota
	metricTopics
	metricStaticTopics
	metricActiveTopics
	metricSubscribers
	metricActiveSubscribers
	metricMessageReceived
	metricMessageSent
)

func newMetrics() *metrics {
	return &metrics{
		vals: map[metric]*int64{
			metricTopics:            ptrInt64(0),
			metricStaticTopics:      ptrInt64(0),
			metricActiveTopics:      ptrInt64(0),
			metricSubscribers:       ptrInt64(0),
			metricActiveSubscribers: ptrInt64(0),
			metricMessageReceived:   ptrInt64(0),
			metricMessageSent:       ptrInt64(0),
		},
	}
}

func (m metric) String() string {
	switch m {
	case metricTopics:
		return "topics"
	case metricStaticTopics:
		return "static_topics"
	case metricActiveTopics:
		return "active_topics"
	case metricSubscribers:
		return "subscribers"
	case metricActiveSubscribers:
		return "active_subscribers"
	case metricMessageReceived:
		return "message_received"
	case metricMessageSent:
		return "message_sent"
	}
	return ""
}

func (m *metrics) inc(k metric) {
	v := m.vals[k]
	atomic.AddInt64(v, 1)
}

func (m *metrics) incBy(k metric, val int64) {
	v := m.vals[k]
	atomic.AddInt64(v, val)
}

func (m *metrics) dec(k metric) {
	v := m.vals[k]
	atomic.AddInt64(v, -1)
}

func (m *metrics) get(k metric) int64 {
	v := m.vals[k]
	return atomic.LoadInt64(v)
}

func ptrInt64(v int64) *int64 {
	return &v
}

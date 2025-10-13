package mqx

import (
	"context"

	"github.com/ecodeclub/mq-api"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const instrumentationName = "internal/pkg/mq/tracing"

// 给发送消息打点
type TraceMq struct {
	mq.MQ
	tracer trace.Tracer
}

func NewTraceMq(mq mq.MQ) *TraceMq {
	return &TraceMq{MQ: mq, tracer: otel.GetTracerProvider().Tracer(instrumentationName)}
}

func (t TraceMq) Producer(topic string) (mq.Producer, error) {
	pro, err := t.MQ.Producer(topic)
	if err != nil {
		return nil, err
	}
	return NewTraceProducer(pro, t.tracer), nil
}

type TraceProducer struct {
	mq.Producer
	tracer trace.Tracer
}

func NewTraceProducer(producer mq.Producer, tracer trace.Tracer) *TraceProducer {
	return &TraceProducer{
		Producer: producer,
		tracer:   tracer,
	}
}

func (t *TraceProducer) Produce(ctx context.Context, m *mq.Message) (*mq.ProducerResult, error) {
	ctx, span := t.tracer.Start(ctx, "mq.produce", trace.WithSpanKind(trace.SpanKindProducer))
	defer span.End()
	setSpanAttributes(span, m)

	res, err := t.Producer.Produce(ctx, m)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	span.SetStatus(codes.Ok, "")
	return res, nil
}

func (t *TraceProducer) ProduceWithPartition(ctx context.Context, m *mq.Message, partition int) (*mq.ProducerResult, error) {
	ctx, span := t.tracer.Start(ctx, "mq.produce_with_partition", trace.WithSpanKind(trace.SpanKindProducer))
	defer span.End()
	setSpanAttributes(span, m)

	res, err := t.Producer.ProduceWithPartition(ctx, m, partition)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	span.SetStatus(codes.Ok, "")
	return res, nil
}

// setSpanAttributes 统一设置 MQ 发送相关的通用属性
func setSpanAttributes(span trace.Span, m *mq.Message) {
	attrs := []attribute.KeyValue{
		attribute.String("messaging.system", "mq"),
		attribute.String("messaging.operation", "produce"),
	}
	if m != nil {
		// 兼容常见字段：topic 与消息长度
		if m.Topic != "" {
			attrs = append(attrs, attribute.String("messaging.topic", m.Topic))
		}
		if m.Value != nil {
			attrs = append(attrs, attribute.Int("messaging.message_length", len(m.Value)))
		}
	}
	span.SetAttributes(attrs...)
}

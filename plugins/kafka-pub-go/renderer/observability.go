// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import "path/filepath"

func (r *Renderer) metricsSource() (source string) {

	return `package ` + filepath.Base(r.outDir) + `

import (
	"errors"
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type metrics struct {
	calls *prometheus.CounterVec
	records *prometheus.CounterVec
	bytes *prometheus.CounterVec
	duration *prometheus.HistogramVec
	batch *prometheus.HistogramVec
	inflight prometheus.Gauge
	info prometheus.Gauge
}

func newMetrics(reg prometheus.Registerer) (result *metrics, err error) {

	result = &metrics{
		calls: prometheus.NewCounterVec(prometheus.CounterOpts{Namespace: "tgp_kafka", Name: "produce_calls_total", Help: "Число вызовов публикации."}, []string{"contract", "method", "topic", "result", "cause"}),
		records: prometheus.NewCounterVec(prometheus.CounterOpts{Namespace: "tgp_kafka", Name: "produce_records_total", Help: "Число опубликованных записей."}, []string{"contract", "method", "topic", "result", "cause"}),
		bytes: prometheus.NewCounterVec(prometheus.CounterOpts{Namespace: "tgp_kafka", Name: "produce_bytes_total", Help: "Размер успешно опубликованных тел."}, []string{"contract", "method", "topic"}),
		duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{Namespace: "tgp_kafka", Name: "produce_duration_seconds", Help: "Длительность публикации.", Buckets: prometheus.DefBuckets}, []string{"contract", "method", "topic", "result"}),
		batch: prometheus.NewHistogramVec(prometheus.HistogramOpts{Namespace: "tgp_kafka", Name: "produce_batch_records", Help: "Число записей в вызове.", Buckets: []float64{1, 2, 5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000}}, []string{"topic"}),
		inflight: prometheus.NewGauge(prometheus.GaugeOpts{Namespace: "tgp_kafka", Name: "produce_inflight", Help: "Число записей, ожидающих completion."}),
		info: prometheus.NewGauge(prometheus.GaugeOpts{Namespace: "tgp_kafka", Name: "client_info", Help: "Версия Kafka-клиента.", ConstLabels: prometheus.Labels{"version": VersionASTg}}),
	}
	if result.calls, err = registerOrGet(reg, result.calls); err != nil {
		return nil, err
	}
	if result.records, err = registerOrGet(reg, result.records); err != nil {
		return nil, err
	}
	if result.bytes, err = registerOrGet(reg, result.bytes); err != nil {
		return nil, err
	}
	if result.duration, err = registerOrGet(reg, result.duration); err != nil {
		return nil, err
	}
	if result.batch, err = registerOrGet(reg, result.batch); err != nil {
		return nil, err
	}
	if result.inflight, err = registerOrGet(reg, result.inflight); err != nil {
		return nil, err
	}
	if result.info, err = registerOrGet(reg, result.info); err != nil {
		return nil, err
	}
	result.info.Set(1)
	return result, nil
}

func registerOrGet[T prometheus.Collector](reg prometheus.Registerer, collector T) (result T, err error) {

	if err = reg.Register(collector); err == nil {
		return collector, nil
	}
	var registered prometheus.AlreadyRegisteredError
	if !errors.As(err, &registered) {
		return result, err
	}
	var ok bool
	result, ok = registered.ExistingCollector.(T)
	if !ok {
		return result, fmt.Errorf("registered collector has type %T, want %T", registered.ExistingCollector, collector)
	}
	return result, nil
}

func (c *Client) observeProduceCall(contract string, method string, topic string, records int, duration time.Duration, callErr error, cause string) {

	if c.metrics == nil {
		return
	}
	result := "ok"
	if callErr != nil {
		result = "error"
	} else {
		cause = "none"
	}
	c.metrics.calls.WithLabelValues(contract, method, topic, result, cause).Inc()
	c.metrics.duration.WithLabelValues(contract, method, topic, result).Observe(duration.Seconds())
	if records > 0 && cause != "encode" {
		c.metrics.batch.WithLabelValues(topic).Observe(float64(records))
	}
}

func (c *Client) observeProduceRecords(contract string, method string, topic string, outcomes []produceOutcome) {

	if c.metrics == nil {
		return
	}
	for _, outcome := range outcomes {
		result := "ok"
		cause := "none"
		if outcome.err != nil {
			result = "error"
			cause = "produce"
		} else {
			c.metrics.bytes.WithLabelValues(contract, method, topic).Add(float64(outcome.bytes))
		}
		c.metrics.records.WithLabelValues(contract, method, topic, result, cause).Inc()
	}
}

func (c *Client) produceHooks() (hooks *produceHooks) {

	if c.metrics == nil {
		return nil
	}
	return &produceHooks{
		onQueued: func(records int) {

			c.metrics.inflight.Add(float64(records))
		},
		onFinished: func() {

			c.metrics.inflight.Dec()
		},
	}
}
`
}

func (r *Renderer) tracingSource() (source string) {

	return `package ` + filepath.Base(r.outDir) + `

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

func (c *Client) startProduceSpan(ctx context.Context, contract string, method string, topic string, records int) (spanContext context.Context, finish func(err error)) {

	if c.tracer == nil {
		return ctx, func(err error) {}
	}
	spanContext, span := c.tracer.Start(ctx, "kafka.produce/"+contract+"."+method)
	span.SetAttributes(
		attribute.String("messaging.system", "kafka"),
		attribute.String("messaging.destination", topic),
		attribute.String("tgp.contract", contract),
		attribute.String("tgp.method", method),
		attribute.Int("tgp.records", records),
	)
	return spanContext, func(err error) {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}
}
`
}

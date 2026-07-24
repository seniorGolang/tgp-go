// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import "path/filepath"

func (r *Renderer) metricsSource() (source string) {

	return `package ` + filepath.Base(r.outDir) + `

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type metrics struct {
	polls           *prometheus.CounterVec
	pollDuration    *prometheus.HistogramVec
	pollActive      prometheus.Gauge
	records         *prometheus.CounterVec
	bytes           *prometheus.CounterVec
	decodeDuration  *prometheus.HistogramVec
	handlerDuration *prometheus.HistogramVec
	lag             *prometheus.GaugeVec
	lagScrapeErrors *prometheus.CounterVec
}

func newMetrics(registerer prometheus.Registerer) (result *metrics, err error) {

	result = &metrics{
		polls: prometheus.NewCounterVec(prometheus.CounterOpts{Namespace: "tgp_kafka", Name: "consume_polls_total", Help: "Число циклов чтения Kafka."}, []string{"result", "cause"}),
		pollDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{Namespace: "tgp_kafka", Name: "consume_poll_duration_seconds", Help: "Длительность опроса и обработки."}, []string{"result"}),
		pollActive: prometheus.NewGauge(prometheus.GaugeOpts{Namespace: "tgp_kafka", Name: "consume_poll_active", Help: "Признак активной обработки опроса."}),
		records: prometheus.NewCounterVec(prometheus.CounterOpts{Namespace: "tgp_kafka", Name: "consume_records_total", Help: "Число обработанных записей."}, []string{"contract", "method", "topic", "result", "cause"}),
		bytes: prometheus.NewCounterVec(prometheus.CounterOpts{Namespace: "tgp_kafka", Name: "consume_bytes_total", Help: "Размер тел обработанных записей."}, []string{"contract", "method", "topic", "result"}),
		decodeDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{Namespace: "tgp_kafka", Name: "consume_decode_duration_seconds", Help: "Длительность разбора записей."}, []string{"contract", "method", "topic", "result"}),
		handlerDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{Namespace: "tgp_kafka", Name: "consume_handler_duration_seconds", Help: "Длительность обработчиков."}, []string{"contract", "method", "topic", "result"}),
		lag: prometheus.NewGaugeVec(prometheus.GaugeOpts{Namespace: "tgp_kafka", Name: "consume_lag_records", Help: "Лаг чтения по partition."}, []string{"topic", "partition"}),
		lagScrapeErrors: prometheus.NewCounterVec(prometheus.CounterOpts{Namespace: "tgp_kafka", Name: "consume_lag_scrape_errors_total", Help: "Ошибки обновления лага."}, []string{"cause"}),
	}
	for _, collector := range []prometheus.Collector{result.polls, result.pollDuration, result.pollActive, result.records, result.bytes, result.decodeDuration, result.handlerDuration, result.lag, result.lagScrapeErrors} {
		if err = registerer.Register(collector); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func (client *Client) observeDecode(contract string, method string, topic string, bytes int, duration time.Duration, err error) {

	if client.metrics == nil {
		return
	}
	result := "ok"
	if err != nil {
		result = "error"
		client.metrics.records.WithLabelValues(contract, method, topic, result, "decode").Inc()
		client.metrics.bytes.WithLabelValues(contract, method, topic, result).Add(float64(bytes))
	}
	client.metrics.decodeDuration.WithLabelValues(contract, method, topic, result).Observe(duration.Seconds())
}

func (client *Client) observeHandler(contract string, method string, topic string, records int, bytes int, duration time.Duration, err error) {

	if client.metrics == nil {
		return
	}
	result := "ok"
	cause := "none"
	if err != nil {
		result = "error"
		cause = "handler"
	}
	client.metrics.handlerDuration.WithLabelValues(contract, method, topic, result).Observe(duration.Seconds())
	client.metrics.records.WithLabelValues(contract, method, topic, result, cause).Add(float64(records))
	client.metrics.bytes.WithLabelValues(contract, method, topic, result).Add(float64(bytes))
}
`
}

func (r *Renderer) tracingSource() (source string) {

	return `package ` + filepath.Base(r.outDir) + `

import (
	"context"

	"go.opentelemetry.io/otel/codes"
)

func (client *Client) startConsumeSpan(ctx context.Context, contract string, method string, topic string, records int) (spanContext context.Context, finish func(err error)) {

	if client.tracer == nil {
		return ctx, func(err error) {}
	}
	spanContext, span := client.tracer.Start(ctx, "kafka.consume/"+contract+"."+method)
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

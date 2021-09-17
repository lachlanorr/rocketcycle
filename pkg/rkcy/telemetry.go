// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
	"context"
	"encoding/hex"
	"fmt"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpgrpc"
	"go.opentelemetry.io/otel/metric/global"
	controller "go.opentelemetry.io/otel/sdk/metric/controller/basic"
	processor "go.opentelemetry.io/otel/sdk/metric/processor/basic"
	"go.opentelemetry.io/otel/sdk/metric/selector/simple"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

type Telemetry struct {
	Ctx      context.Context
	Exporter *otlp.Exporter
	Tp       *sdktrace.TracerProvider
	Pusher   *controller.Controller
}

func NewTelemetry(ctx context.Context) (*Telemetry, error) {
	var err error
	telem := Telemetry{Ctx: ctx}

	// Set different endpoints for the metrics and traces collectors
	metricsDriver := otlpgrpc.NewDriver(
		otlpgrpc.WithInsecure(),
		otlpgrpc.WithEndpoint(gSettings.OtelcolEndpoint),
	)
	tracesDriver := otlpgrpc.NewDriver(
		otlpgrpc.WithInsecure(),
		otlpgrpc.WithEndpoint(gSettings.OtelcolEndpoint),
	)
	splitCfg := otlp.SplitConfig{
		ForMetrics: metricsDriver,
		ForTraces:  tracesDriver,
	}
	driver := otlp.NewSplitDriver(splitCfg)
	telem.Exporter, err = otlp.NewExporter(ctx, driver)
	if err != nil {
		return nil, err
	}

	telem.Tp = sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithBatcher(
			telem.Exporter,
			// add following two options to ensure flush
			sdktrace.WithBatchTimeout(5*time.Second),
			sdktrace.WithMaxExportBatchSize(10),
		),
	)

	otel.SetTracerProvider(telem.Tp)

	telem.Pusher = controller.New(
		processor.New(
			simple.NewWithExactDistribution(),
			telem.Exporter,
		),
		controller.WithExporter(telem.Exporter),
		controller.WithCollectPeriod(2*time.Second),
	)
	global.SetMeterProvider(telem.Pusher.MeterProvider())

	if err := telem.Pusher.Start(ctx); err != nil {
		telem.Exporter.Shutdown(ctx)
		return nil, err
	}

	return &telem, nil
}

func (telem *Telemetry) Close() {
	telem.Pusher.Stop(telem.Ctx)
	telem.Tp.Shutdown(telem.Ctx)
	telem.Exporter.Shutdown(telem.Ctx)
}

func SpanContext(ctx context.Context, traceId string) context.Context {
	var (
		traceIdBytes trace.TraceID
		err          error
	)
	traceIdBytes, err = trace.TraceIDFromHex(traceId)
	if err != nil {
		randTraceId := NewTraceId()
		log.Error().Err(err).Msgf("Telemetry.Start: Invalid traceId '%s', using new random one '%s'", traceId, randTraceId)
		traceIdBytes, err = trace.TraceIDFromHex(randTraceId)
		if err != nil {
			panic(err)
		}
	}

	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceIdBytes,
		TraceFlags: 0,
	})
	return trace.ContextWithRemoteSpanContext(ctx, sc)
}

var gTraceParentRe *regexp.Regexp = regexp.MustCompile("([0-9a-f]{2})-([0-9a-f]{32})-([0-9a-f]{16})-([0-9a-f]{2})")

func TraceParentIsValid(traceParent string) bool {
	return gTraceParentRe.MatchString(traceParent)
}

func TraceParentParts(traceParent string) []string {
	matches := gTraceParentRe.FindAllStringSubmatch(traceParent, -1)
	if len(matches) != 1 || len(matches[0]) != 5 {
		return make([]string, 0)
	}
	return matches[0][1:]
}

func TraceIdFromTraceParent(traceParent string) string {
	parts := TraceParentParts(traceParent)
	if len(parts) != 4 {
		log.Warn().Msgf("TraceIdFromTraceParent with invalid traceParent: '%s'", traceParent)
		return ""
	}
	return parts[1]
}

func ExtractTraceParent(ctx context.Context) string {
	sc := trace.SpanContextFromContext(ctx)
	if !sc.IsValid() {
		return ""
	}
	tp := fmt.Sprintf(
		"00-%s-%s-%s",
		sc.TraceID(),
		sc.SpanID(),
		sc.TraceFlags(),
	)
	return tp
}

func InjectTraceParent(ctx context.Context, traceParent string) context.Context {
	parts := TraceParentParts(traceParent)
	if len(parts) != 4 {
		log.Error().Msgf("InjectTraceParent re match failure, traceParent='%s'", traceParent)
		return ctx
	}

	traceIdBytes, err := trace.TraceIDFromHex(parts[1])
	if err != nil {
		log.Error().Err(err).Msgf("InjectTraceParent TraceIDFromHex failure, traceParent='%s'", traceParent)
		return ctx
	}
	spanIdBytes, err := trace.SpanIDFromHex(parts[2])
	if err != nil {
		log.Error().Err(err).Msgf("InjectTraceParent SpanIDFromHex failure, traceParent='%s'", traceParent)
		return ctx
	}
	flagBytes, err := hex.DecodeString(parts[3])
	if err != nil {
		log.Error().Err(err).Msgf("InjectTraceParent decode flags failure, traceParent='%s'", traceParent)
		return ctx
	}

	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceIdBytes,
		SpanID:     spanIdBytes,
		TraceFlags: trace.TraceFlags(flagBytes[0]),
		Remote:     true,
	})
	return trace.ContextWithRemoteSpanContext(ctx, sc)
}

func Tracer() trace.Tracer {
	return otel.Tracer("rocketcycle")
}

func funcName(skip int) string {
	pc, _, _, ok := runtime.Caller(skip + 1)
	var funcName string
	if ok {
		fn := runtime.FuncForPC(pc)
		if fn != nil {
			funcName = fn.Name()
		}
	}
	lastSlash := strings.LastIndex(funcName, "/")
	if lastSlash != -1 {
		funcName = funcName[lastSlash+1:]
	}

	return funcName
}

func (telem *Telemetry) Start(ctx context.Context, name string) (context.Context, trace.Span) {
	return Tracer().Start(ctx, name)
}

func (telem *Telemetry) StartFunc(ctx context.Context) (context.Context, trace.Span) {
	return Tracer().Start(ctx, funcName(1))
}

func (telem *Telemetry) StartRequest(ctx context.Context) (context.Context, string, trace.Span) {
	traceId := NewTraceId()

	ctx = SpanContext(ctx, traceId)
	ctx, span := Tracer().Start(
		ctx,
		funcName(1),
	)
	return ctx, traceId, span
}

func (telem *Telemetry) StartStep(ctx context.Context, rtxn *rtApecsTxn) (context.Context, trace.Span, *ApecsTxn_Step) {
	sc := trace.SpanContextFromContext(ctx)
	if !sc.IsValid() {
		ctx = SpanContext(ctx, rtxn.txn.TraceId)
	}

	step := rtxn.currentStep()
	spanName := fmt.Sprintf("Step %s %d %s", rtxn.txn.DirectionName(), rtxn.txn.CurrentStepIdx, step.Command)

	ctx, span := Tracer().Start(
		ctx,
		spanName,
		trace.WithAttributes(
			attribute.String("rkcy.command", step.Command),
			attribute.String("rkcy.key", step.Key),
		),
	)

	return ctx, span, step
}

func (telem *Telemetry) RecordProduceError(name string, traceId string, topic string, partition int32, err error) {
	ctx := SpanContext(context.Background(), traceId)
	ctx, span := Tracer().Start(
		ctx,
		"ProduceError: "+name,
		trace.WithAttributes(
			attribute.String("rkcy.topic", topic),
			attribute.String("rkcy.error", err.Error()),
			attribute.Int("rkcy.partition", int(partition)),
		),
	)
	span.End()
}

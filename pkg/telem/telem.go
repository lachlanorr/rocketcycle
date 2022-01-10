// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package telem

import (
	"context"
	"encoding/hex"
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/metric/global"
	controller "go.opentelemetry.io/otel/sdk/metric/controller/basic"
	processor "go.opentelemetry.io/otel/sdk/metric/processor/basic"
	"go.opentelemetry.io/otel/sdk/metric/selector/simple"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"

	"github.com/lachlanorr/rocketcycle/pkg/rkcy"
	"github.com/lachlanorr/rocketcycle/pkg/rkcypb"
)

var (
	gIsInitialized  bool
	gTraceExporter  *otlptrace.Exporter
	gMetricExporter *otlpmetric.Exporter
	gTp             *sdktrace.TracerProvider
	gMetricPusher   *controller.Controller
)

func initialize(
	ctx context.Context,
	traceClient otlptrace.Client,
	metricClient otlpmetric.Client,
) error {
	if gIsInitialized {
		return nil
	}

	gIsInitialized = true

	var err error

	// traces
	gTraceExporter, err = otlptrace.New(ctx, traceClient)
	if err != nil {
		Shutdown(ctx)
		return err
	}

	gTp = sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithBatcher(
			gTraceExporter,
			// add following two options to ensure flush
			sdktrace.WithBatchTimeout(5*time.Second),
			sdktrace.WithMaxExportBatchSize(10),
		),
	)

	otel.SetTracerProvider(gTp)

	// metrics
	gMetricExporter, err = otlpmetric.New(ctx, metricClient)
	if err != nil {
		Shutdown(ctx)
		return err
	}

	gMetricPusher = controller.New(
		processor.NewFactory(
			simple.NewWithHistogramDistribution(),
			gMetricExporter,
		),
		controller.WithExporter(gMetricExporter),
		controller.WithCollectPeriod(2*time.Second),
	)
	global.SetMeterProvider(gMetricPusher)

	if err := gMetricPusher.Start(ctx); err != nil {
		Shutdown(ctx)
		return err
	}

	return nil
}

func Initialize(ctx context.Context, otelcolEndpoint string) error {
	if otelcolEndpoint != "offline" {
		traceClient := otlptracegrpc.NewClient(
			otlptracegrpc.WithInsecure(),
			otlptracegrpc.WithEndpoint(otelcolEndpoint),
			otlptracegrpc.WithReconnectionPeriod(50*time.Millisecond),
		)

		metricClient := otlpmetricgrpc.NewClient(
			otlpmetricgrpc.WithInsecure(),
			otlpmetricgrpc.WithEndpoint(otelcolEndpoint),
			otlpmetricgrpc.WithReconnectionPeriod(50*time.Millisecond),
		)

		return initialize(ctx, traceClient, metricClient)
	} else {
		return InitializeOffline(ctx)
	}
}

func InitializeOffline(ctx context.Context) error {
	return initialize(ctx, &stubTraceClient{}, &stubMetricClient{})
}

func Shutdown(ctx context.Context) {
	// metrics
	if gMetricPusher != nil {
		gMetricPusher.Stop(ctx)
	}
	if gMetricExporter != nil {
		gMetricExporter.Shutdown(ctx)
	}

	// traces
	if gTp != nil {
		gTp.Shutdown(ctx)
	}
	if gTraceExporter != nil {
		gTraceExporter.Shutdown(ctx)
	}
}

func SpanContext(ctx context.Context, traceId string) context.Context {
	var (
		traceIdBytes trace.TraceID
		err          error
	)
	traceIdBytes, err = trace.TraceIDFromHex(traceId)
	if err != nil {
		randTraceId := rkcy.NewTraceId()
		log.Error().Err(err).Msgf("Telem.Start: Invalid traceId '%s', using new random one '%s'", traceId, randTraceId)
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

func RecordSpanError(span trace.Span, err error) {
	span.SetStatus(codes.Error, err.Error())
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
	parts := rkcy.TraceParentParts(traceParent)
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

func Start(ctx context.Context, name string) (context.Context, trace.Span) {
	return Tracer().Start(ctx, name)
}

func StartFunc(ctx context.Context) (context.Context, trace.Span) {
	return Tracer().Start(ctx, funcName(1))
}

func StartRequest(ctx context.Context) (context.Context, string, trace.Span) {
	traceId := rkcy.NewTraceId()

	ctx = SpanContext(ctx, traceId)
	ctx, span := Tracer().Start(
		ctx,
		funcName(1),
	)
	return ctx, traceId, span
}

func StartStep(ctx context.Context, rtxn *rkcy.RtApecsTxn) (context.Context, trace.Span, *rkcypb.ApecsTxn_Step) {
	sc := trace.SpanContextFromContext(ctx)
	if !sc.IsValid() {
		ctx = SpanContext(ctx, rtxn.Txn.Id)
	}

	step := rtxn.CurrentStep()
	spanName := fmt.Sprintf("Step %s %d %s", rkcy.TxnDirectionName(rtxn.Txn), rtxn.Txn.CurrentStepIdx, step.Command)

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

func RecordProduceError(name string, traceId string, topic string, partition int32, err error) {
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

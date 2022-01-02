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
	otel_codes "go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpgrpc"
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
	gOtelcolEndpoint string
	gExporter        *otlp.Exporter
	gTp              *sdktrace.TracerProvider
	gPusher          *controller.Controller
)

func Initialize(ctx context.Context, otelcolEndpoint string) error {
	var err error
	gOtelcolEndpoint = otelcolEndpoint

	// Set different endpoints for the metrics and traces collectors
	metricsDriver := otlpgrpc.NewDriver(
		otlpgrpc.WithInsecure(),
		otlpgrpc.WithEndpoint(otelcolEndpoint),
	)
	tracesDriver := otlpgrpc.NewDriver(
		otlpgrpc.WithInsecure(),
		otlpgrpc.WithEndpoint(otelcolEndpoint),
	)
	splitCfg := otlp.SplitConfig{
		ForMetrics: metricsDriver,
		ForTraces:  tracesDriver,
	}
	driver := otlp.NewSplitDriver(splitCfg)
	gExporter, err = otlp.NewExporter(ctx, driver)
	if err != nil {
		return err
	}

	gTp = sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithBatcher(
			gExporter,
			// add following two options to ensure flush
			sdktrace.WithBatchTimeout(5*time.Second),
			sdktrace.WithMaxExportBatchSize(10),
		),
	)

	otel.SetTracerProvider(gTp)

	gPusher = controller.New(
		processor.New(
			simple.NewWithExactDistribution(),
			gExporter,
		),
		controller.WithExporter(gExporter),
		controller.WithCollectPeriod(2*time.Second),
	)
	global.SetMeterProvider(gPusher.MeterProvider())

	if err := gPusher.Start(ctx); err != nil {
		gExporter.Shutdown(ctx)
		return err
	}

	return nil
}

func Shutdown(ctx context.Context) {
	if gPusher != nil {
		gPusher.Stop(ctx)
	}
	if gTp != nil {
		gTp.Shutdown(ctx)
	}
	if gExporter != nil {
		gExporter.Shutdown(ctx)
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
	span.SetStatus(otel_codes.Error, err.Error())
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

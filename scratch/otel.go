// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package main

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/exporters/otlp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpgrpc"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/global"
	"go.opentelemetry.io/otel/propagation"
	controller "go.opentelemetry.io/otel/sdk/metric/controller/basic"
	processor "go.opentelemetry.io/otel/sdk/metric/processor/basic"
	"go.opentelemetry.io/otel/sdk/metric/selector/simple"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

type Telemetry struct {
	Ctx    context.Context
	Tp     *sdktrace.TracerProvider
	Pusher *controller.Controller
}

func NewTraceId() string {
	traceId := strings.ReplaceAll(uuid.NewString(), "-", "")
	fmt.Printf("NewTraceId: %s\n", traceId)
	return traceId
}

func NewSpanId() string {
	spanId := strings.ReplaceAll(uuid.NewString(), "-", "")[:16]
	fmt.Printf("NewSpanId: %s\n", spanId)
	return spanId
}

func BlankSpanId() string {
	return "0000000000000000"
}

func TraceIdBytes(traceId string) trace.TraceID {
	b, err := trace.TraceIDFromHex(traceId)
	if err != nil {
		panic(err)
	}
	return b
}

func SpanIdBytes(spanId string) trace.SpanID {
	b, err := trace.SpanIDFromHex(spanId)
	if err != nil {
		panic(err)
	}
	return b
}

type MyCarrier struct{}

func (MyCarrier) Get(key string) string {
	log.Fatalf("MyCarrier.Get %s", key)
	return "ERROR"
}

func (MyCarrier) Set(key string, value string) {
	log.Fatalf("MyCarrier.Set %s, %s", key, value)
}

func (MyCarrier) Keys() []string {
	log.Fatalf("MyCarrier.Keys")
	return nil
}

type MyPropagator struct{}

func (MyPropagator) Extract(ctx context.Context, carrier propagation.TextMapCarrier) context.Context {
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: TraceIdBytes(NewTraceId()),
		//		SpanID:     SpanIdBytes(BlankSpanId()),
		TraceFlags: 0,
	})
	return trace.ContextWithRemoteSpanContext(ctx, sc)
}

func (MyPropagator) Inject(context.Context, propagation.TextMapCarrier) {}

func (MyPropagator) Fields() []string {
	return nil
}

func main() {
	// hack tests

	traceId := NewTraceId()
	traceIdBytes := TraceIdBytes(traceId)
	fmt.Printf("%s %s\n", traceId, traceIdBytes.String())

	var traceParentRe *regexp.Regexp = regexp.MustCompile("([0-9a-f]{2})-([0-9a-f]{32})-([0-9a-f]{16})-([0-9a-f]{2})")
	matches := traceParentRe.FindAllStringSubmatch("00-4bf92f3577b34da6a3ce929d0e0e4736-0000000000000002-01", -1)
	fmt.Printf("matches: %+v\n", matches)

	// test what's in empty SpanContext
	sc := trace.SpanContextFromContext(context.Background())

	fmt.Printf("sc: %+v\n", sc)
	fmt.Printf("HasTraceID %+v\n", sc.HasTraceID())
}

func mainWorks() {

	otel.SetTextMapPropagator(MyPropagator{})

	// Set different endpoints for the metrics and traces collectors
	metricsDriver := otlpgrpc.NewDriver(
		otlpgrpc.WithInsecure(),
		otlpgrpc.WithEndpoint("localhost:4317"),
	)
	tracesDriver := otlpgrpc.NewDriver(
		otlpgrpc.WithInsecure(),
		otlpgrpc.WithEndpoint("localhost:4317"),
	)
	splitCfg := otlp.SplitConfig{
		ForMetrics: metricsDriver,
		ForTraces:  tracesDriver,
	}
	driver := otlp.NewSplitDriver(splitCfg)
	ctx := context.Background()
	exp, err := otlp.NewExporter(ctx, driver)
	if err != nil {
		log.Fatalf("failed to create the collector exporter: %v", err)
	}

	defer func() {
		ctx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()
		if err := exp.Shutdown(ctx); err != nil {
			otel.Handle(err)
		}
	}()

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithBatcher(
			exp,
			// add following two options to ensure flush
			sdktrace.WithBatchTimeout(5*time.Second),
			sdktrace.WithMaxExportBatchSize(10),
		),
	)
	defer func() {
		ctx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()
		if err := tp.Shutdown(ctx); err != nil {
			otel.Handle(err)
		}
	}()
	otel.SetTracerProvider(tp)

	pusher := controller.New(
		processor.New(
			simple.NewWithExactDistribution(),
			exp,
		),
		controller.WithExporter(exp),
		controller.WithCollectPeriod(2*time.Second),
	)
	global.SetMeterProvider(pusher.MeterProvider())

	if err := pusher.Start(ctx); err != nil {
		log.Fatalf("could not start metric controoler: %v", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()
		// pushes any last exports to the receiver
		if err := pusher.Stop(ctx); err != nil {
			otel.Handle(err)
		}
	}()

	tracer := otel.Tracer("test-tracer")
	meter := global.Meter("test-meter")

	// Recorder metric example
	counter := metric.Must(meter).
		NewFloat64Counter(
			"an_important_metric",
			metric.WithDescription("Measures the cumulative epicness of the app"),
		)

	// work begins
	//prop := MyPropagator{}
	prop := otel.GetTextMapPropagator()

	fmt.Printf("ctx before %+v\n", ctx)
	ctx = prop.Extract(ctx, MyCarrier{})
	fmt.Printf("ctx after  %+v\n", ctx)
	ctx, span := tracer.Start(
		ctx,
		"DifferentCollectors-Example")
	defer span.End()
	for i := 0; i < 10; i++ {
		_, iSpan := tracer.Start(ctx, fmt.Sprintf("Sample-%d", i))
		log.Printf("Doing really hard work (%d / 10)\n", i+1)
		counter.Add(ctx, 1.0)

		<-time.After(time.Second)
		iSpan.End()
	}

	log.Printf("Done!")
}

func main0() {
	ctx := context.Background()
	driver := otlpgrpc.NewDriver(otlpgrpc.WithInsecure())
	exporter, err := otlp.NewExporter(ctx, driver)
	if err != nil {
		fmt.Printf("Failed to create the collector exporter: %v", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()
		if err := exporter.Shutdown(ctx); err != nil {
			otel.Handle(err)
		}
	}()

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithBatcher(
			exporter,
			// add following two options to ensure flush
			sdktrace.WithBatchTimeout(5*time.Second),
			sdktrace.WithMaxExportBatchSize(10),
		),
	)
	defer func() {
		ctx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()
		if err := tp.Shutdown(ctx); err != nil {
			otel.Handle(err)
		}
	}()
	otel.SetTracerProvider(tp)

	tracer := otel.Tracer("test-tracer")

	// Then use the OpenTelemetry tracing library, like we normally would.
	ctx, span := tracer.Start(ctx, "CollectorExporter-Example")
	defer span.End()

	for i := 0; i < 10; i++ {
		_, iSpan := tracer.Start(ctx, fmt.Sprintf("Sample-%d", i))
		<-time.After(6 * time.Millisecond)
		iSpan.End()
	}
}

func main1() {
	telem := Telemetry{}
	ctx := context.Background()
	driver := otlpgrpc.NewDriver(otlpgrpc.WithInsecure())
	exporter, err := otlp.NewExporter(ctx, driver)
	if err != nil {
		fmt.Printf("Failed to create the collector exporter: %v", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()
		if err := exporter.Shutdown(ctx); err != nil {
			otel.Handle(err)
		}
	}()

	telem.Tp = sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithBatcher(
			exporter,
			// add following two options to ensure flush
			sdktrace.WithBatchTimeout(5*time.Second),
			sdktrace.WithMaxExportBatchSize(10),
		),
	)
	defer func() {
		ctx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()
		if err := telem.Tp.Shutdown(ctx); err != nil {
			otel.Handle(err)
		}
	}()

	telem.Pusher = controller.New(
		processor.New(
			simple.NewWithExactDistribution(),
			exporter,
		),
		controller.WithExporter(exporter),
		controller.WithCollectPeriod(5*time.Second),
	)

	defer func() {
		ctx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()
		if err := telem.Pusher.Stop(ctx); err != nil {
			otel.Handle(err)
		}
	}()

	otel.SetTracerProvider(telem.Tp)
	global.SetMeterProvider(telem.Pusher.MeterProvider())
	propagator := propagation.NewCompositeTextMapPropagator(propagation.Baggage{}, propagation.TraceContext{})
	otel.SetTextMapPropagator(propagator)

	//------------------------------------------------------------------
	fooKey := attribute.Key("ex.com/foo")
	barKey := attribute.Key("ex.com/bar")
	lemonsKey := attribute.Key("ex.com/lemons")
	anotherKey := attribute.Key("ex.com/another")

	commonAttributes := []attribute.KeyValue{
		lemonsKey.Int(10),
		attribute.String("A", "1"),
		attribute.String("B", "2"),
		attribute.String("C", "3"),
	}

	meter := global.Meter("ex.com/basic")

	observerCallback := func(_ context.Context, result metric.Float64ObserverResult) {
		result.Observe(1, commonAttributes...)
	}
	_ = metric.Must(meter).NewFloat64ValueObserver("ex.com.one", observerCallback,
		metric.WithDescription("A ValueObserver set to 1.0"),
	)

	valueRecorder := metric.Must(meter).NewFloat64ValueRecorder("ex.com.two")

	boundRecorder := valueRecorder.Bind(commonAttributes...)
	defer boundRecorder.Unbind()

	tracer := otel.Tracer("ex.com/basic")
	ctx = baggage.ContextWithValues(ctx,
		fooKey.String("foo1"),
		barKey.String("bar1"),
	)

	func(ctx context.Context) {
		var span trace.Span
		ctx, span = tracer.Start(ctx, "operation")
		defer span.End()

		span.AddEvent("Nice operation!", trace.WithAttributes(attribute.Int("bogons", 100)))
		span.SetAttributes(anotherKey.String("yes"))

		meter.RecordBatch(
			// Note: call-site variables added as context Entries:
			baggage.ContextWithValues(ctx, anotherKey.String("xyz")),
			commonAttributes,

			valueRecorder.Measurement(2.0),
		)

		func(ctx context.Context) {
			var span trace.Span
			ctx, span = tracer.Start(ctx, "Sub operation...")
			defer span.End()

			span.SetAttributes(lemonsKey.String("five"))
			span.AddEvent("Sub span event")
			boundRecorder.Record(ctx, 1.3)
		}(ctx)
	}(ctx)
}

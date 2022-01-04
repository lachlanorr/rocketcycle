// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package telem

import (
	"context"

	metricpb "go.opentelemetry.io/proto/otlp/metrics/v1"
	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"
)

// stubTraceClient
type stubTraceClient struct{}

func (m *stubTraceClient) Start(ctx context.Context) error {
	return nil
}

func (m *stubTraceClient) Stop(ctx context.Context) error {
	return nil
}

func (m *stubTraceClient) UploadTraces(ctx context.Context, protoSpans []*tracepb.ResourceSpans) error {
	return nil
}

// stubTraceClient (END)

// stubMetricClient
type stubMetricClient struct{}

func (m *stubMetricClient) Start(ctx context.Context) error {
	return nil
}

func (m *stubMetricClient) Stop(ctx context.Context) error {
	return nil
}

func (m *stubMetricClient) UploadMetrics(ctx context.Context, protoMetrics []*metricpb.ResourceMetrics) error {
	return nil
}

// stubMetricClient (END)

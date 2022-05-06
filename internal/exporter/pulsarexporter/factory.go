// Copyright  The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package pulsarexporter

import (
	ctx "context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

const (
	typeStr             = "pulsar"
	defaultTracesTopic  = "otlp_spans"
	defaultMetricsTopic = "otlp_metrics"
	defaultLogsTopic    = "otlp_logs"
	defaultEncoding     = "otlp_proto"
	defaultBroker       = "pulsar://localhost:6651"
	//pulsar.ClientOptions
	defaultOperationTimeout  = 30
	defaultConnectionTimeout = 30
	//pulsar.ProducerOptions

)

// FactoryOption applies changes to pulsarExporterFactory.
type FactoryOption func(factory *pulsarExporterFactory)

// NewFactory creates pulsar exporter factory.
func NewFactory(options ...FactoryOption) component.ExporterFactory {
	f := &pulsarExporterFactory{
		metricsMarshalers: metricsMarshalers(),
	}
	for _, o := range options {
		o(f)
	}
	return component.NewExporterFactory(
		typeStr,
		createDefaultConfig,
		component.WithMetricsExporter(f.createMetricsExporter),
	)
}

type pulsarExporterFactory struct {
	metricsMarshalers map[string]MetricsMarshaler
}

func createDefaultConfig() config.Exporter {
	return &Config{
		ExporterSettings: config.NewExporterSettings(config.NewComponentID(typeStr)),
		TimeoutSettings:  exporterhelper.NewDefaultTimeoutSettings(),
		RetrySettings:    exporterhelper.NewDefaultRetrySettings(),
		QueueSettings:    exporterhelper.NewDefaultQueueSettings(),
		Brokers:          defaultBroker,
		// using an empty topic to track when it has not been set by user, default is based on traces or metrics.
		Topic:    "",
		Encoding: defaultEncoding,
		Metadata: Metadata{},
		Producer: Producer{},
	}
}

func (f *pulsarExporterFactory) createMetricsExporter(
	_ ctx.Context,
	set component.ExporterCreateSettings,
	cfg config.Exporter,
) (component.MetricsExporter, error) {
	oCfg := cfg.(*Config)
	if oCfg.Topic == "" {
		oCfg.Topic = defaultMetricsTopic
	}
	if oCfg.Encoding == "otlp_json" {
		set.Logger.Info("otlp_json is considered experimental and should not be used in a production environment")
	}
	exp, err := newMetricsExporter(*oCfg, set, f.metricsMarshalers)
	if err != nil {
		return nil, err
	}
	return exporterhelper.NewMetricsExporter(
		cfg,
		set,
		exp.metricsDataPusher,
		exporterhelper.WithCapabilities(consumer.Capabilities{MutatesData: false}),
		// Disable exporterhelper Timeout, because we cannot pass a Context to the Producer,
		// and will rely on the sarama Producer Timeout logic.
		exporterhelper.WithTimeout(exporterhelper.TimeoutSettings{Timeout: 0}),
		exporterhelper.WithRetry(oCfg.RetrySettings),
		exporterhelper.WithQueue(oCfg.QueueSettings),
		exporterhelper.WithShutdown(exp.Close))
}

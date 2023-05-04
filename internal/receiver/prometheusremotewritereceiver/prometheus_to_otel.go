// Copyright Splunk, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package prometheusremotewritereceiver

import (
	"errors"
	"fmt"
	"math"
	"sync/atomic"
	"time"

	"github.com/prometheus/prometheus/prompb"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/multierr"

	"github.com/signalfx/splunk-otel-collector/internal/receiver/prometheusremotewritereceiver/internal"
)

type MetricData struct {
	MetricName     string
	Labels         []prompb.Label
	Samples        []prompb.Sample
	Exemplars      []prompb.Exemplar
	Histograms     []prompb.Histogram
	MetricMetadata prompb.MetricMetadata
}

type PrometheusRemoteOtelParser struct {
	SfxGatewayCompatability bool
	totalNans               int64
	totalInvalidRequests    int64
	totalBadMetrics         int64
}

func (prwParser *PrometheusRemoteOtelParser) FromPrometheusWriteRequestMetrics(request *prompb.WriteRequest) (pmetric.Metrics, error) {
	var otelMetrics pmetric.Metrics
	metricFamiliesAndData, err := prwParser.partitionWriteRequest(request)
	if nil == err {
		otelMetrics, err = prwParser.TransformPrometheusRemoteWriteToOtel(metricFamiliesAndData)
	}
	if prwParser.SfxGatewayCompatability {
		if otelMetrics == pmetric.NewMetrics() {
			otelMetrics.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty()
		}
		startTime, endTime := getWriteRequestTimestampBounds(request)
		scope := otelMetrics.ResourceMetrics().At(0).ScopeMetrics().At(0)
		prwParser.addBadRequests(scope, startTime, endTime)
		prwParser.addNanDataPoints(scope, startTime, endTime)
		prwParser.addMetricsWithMissingName(scope, startTime, endTime)
	}
	return otelMetrics, err
}

func (prwParser *PrometheusRemoteOtelParser) TransformPrometheusRemoteWriteToOtel(parsedPrwMetrics map[string][]MetricData) (pmetric.Metrics, error) {
	metric := pmetric.NewMetrics()
	rm := metric.ResourceMetrics().AppendEmpty()
	var translationErrors error
	for metricFamily, metrics := range parsedPrwMetrics {
		err := prwParser.addMetrics(rm, metricFamily, metrics)
		if err != nil {
			translationErrors = multierr.Append(translationErrors, err)
		}
	}
	return metric, translationErrors
}

func (prwParser *PrometheusRemoteOtelParser) partitionWriteRequest(writeReq *prompb.WriteRequest) (map[string][]MetricData, error) {
	partitions := make(map[string][]MetricData)
	var translationErrors error
	for index, ts := range writeReq.Timeseries {
		metricName, err := internal.ExtractMetricNameLabel(ts.Labels)
		if err != nil {
			translationErrors = multierr.Append(translationErrors, err)
		}
		metricFamilyName := internal.GetBaseMetricFamilyName(metricName)
		if metricFamilyName == "" {
			translationErrors = multierr.Append(translationErrors, fmt.Errorf("metric family name missing: %s", metricName))
		}

		metricType := internal.GuessMetricTypeByLabels(metricName, ts.Labels)
		metricMetadata := prompb.MetricMetadata{
			MetricFamilyName: metricFamilyName,
			Type:             metricType,
		}
		metricData := MetricData{
			Labels:         ts.Labels,
			Samples:        writeReq.Timeseries[index].Samples,
			Exemplars:      writeReq.Timeseries[index].Exemplars,
			Histograms:     writeReq.Timeseries[index].Histograms,
			MetricName:     metricName,
			MetricMetadata: metricMetadata,
		}
		if len(metricData.Samples) < 1 {
			translationErrors = multierr.Append(translationErrors, fmt.Errorf("no samples found for  %s", metricName))
		}
		partitions[metricData.MetricMetadata.MetricFamilyName] = append(partitions[metricData.MetricMetadata.MetricFamilyName], metricData)
	}

	return partitions, translationErrors
}

// This actually converts from a prometheus prompdb.MetaDataType to the closest equivalent otel type
// See https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/13bcae344506fe2169b59d213361d04094c651f6/receiver/prometheusreceiver/internal/util.go#L106
func (prwParser *PrometheusRemoteOtelParser) addMetrics(rm pmetric.ResourceMetrics, family string, metrics []MetricData) error {
	if family == "" || len(metrics) == 0 {
		return errors.New("missing name or metrics")
	}

	ilm := rm.ScopeMetrics().AppendEmpty()
	ilm.Scope().SetName(typeString)
	ilm.Scope().SetVersion("0.1")

	// When we add native histogram support, this will be a map lookup on metrics family
	metricsMetadata := metrics[0].MetricMetadata

	var err error
	switch metricsMetadata.Type {
	case prompb.MetricMetadata_GAUGE, prompb.MetricMetadata_UNKNOWN:
		err = prwParser.addGaugeMetrics(ilm, family, metrics, metricsMetadata)
	case prompb.MetricMetadata_COUNTER:
		err = prwParser.addCounterMetrics(ilm, family, metrics, metricsMetadata)
	case prompb.MetricMetadata_HISTOGRAM, prompb.MetricMetadata_GAUGEHISTOGRAM:
		if prwParser.SfxGatewayCompatability {
			err = prwParser.addCounterMetrics(ilm, family, metrics, metricsMetadata)
		} else {
			err = fmt.Errorf("this version of the prometheus remote write receiver only supports SfxGatewayCompatability mode")
		}
	case prompb.MetricMetadata_SUMMARY:
		if prwParser.SfxGatewayCompatability {
			err = prwParser.addCounterMetrics(ilm, family, metrics, metricsMetadata)
		} else {
			err = fmt.Errorf("this version of the prometheus remote write receiver only supports SfxGatewayCompatability mode")
		}
	case prompb.MetricMetadata_INFO, prompb.MetricMetadata_STATESET:
		err = prwParser.addInfoStateset(ilm, family, metrics, metricsMetadata)
	default:
		err = fmt.Errorf("unsupported type %s for metric family %s", metricsMetadata.Type, family)
	}
	return err
}

func (prwParser *PrometheusRemoteOtelParser) scaffoldNewMetric(ilm pmetric.ScopeMetrics, name string, metricsMetadata prompb.MetricMetadata) pmetric.Metric {
	nm := ilm.Metrics().AppendEmpty()
	nm.SetUnit(metricsMetadata.Unit)
	nm.SetDescription(metricsMetadata.GetHelp())
	nm.SetName(name)
	return nm
}

// addMetricsWithMissingName is used to report metrics in the remote write request without names
func (prwParser *PrometheusRemoteOtelParser) addBadRequests(ilm pmetric.ScopeMetrics, start time.Time, end time.Time) {
	if !prwParser.SfxGatewayCompatability {
		return
	}
	errMetric := ilm.Metrics().AppendEmpty()
	errMetric.SetName("prometheus.invalid_requests")
	errorSum := errMetric.SetEmptySum()
	errorSum.SetAggregationTemporality(pmetric.AggregationTemporalityCumulative)
	errorSum.SetIsMonotonic(true)
	dp := errorSum.DataPoints().AppendEmpty()
	dp.SetIntValue(atomic.LoadInt64(&prwParser.totalInvalidRequests))
	dp.SetStartTimestamp(pcommon.NewTimestampFromTime(start))
	dp.SetTimestamp(pcommon.NewTimestampFromTime(end))
}

// addMetricsWithMissingName is used to report metrics in the remote write request without names
func (prwParser *PrometheusRemoteOtelParser) addMetricsWithMissingName(ilm pmetric.ScopeMetrics, start time.Time, end time.Time) {
	if !prwParser.SfxGatewayCompatability {
		return
	}
	errMetric := ilm.Metrics().AppendEmpty()
	errMetric.SetName("prometheus.total_bad_datapoints")
	errorSum := errMetric.SetEmptySum()
	errorSum.SetAggregationTemporality(pmetric.AggregationTemporalityCumulative)
	errorSum.SetIsMonotonic(true)
	dp := errorSum.DataPoints().AppendEmpty()
	dp.SetIntValue(atomic.LoadInt64(&prwParser.totalBadMetrics))

	dp.SetStartTimestamp(pcommon.NewTimestampFromTime(start))
	dp.SetTimestamp(pcommon.NewTimestampFromTime(end))
}

func (prwParser *PrometheusRemoteOtelParser) addNanDataPoints(ilm pmetric.ScopeMetrics, start time.Time, end time.Time) {
	if !prwParser.SfxGatewayCompatability {
		return
	}
	errMetric := ilm.Metrics().AppendEmpty()
	errMetric.SetName("prometheus.total_NAN_samples")
	errorSum := errMetric.SetEmptySum()
	errorSum.SetAggregationTemporality(pmetric.AggregationTemporalityCumulative)
	errorSum.SetIsMonotonic(true)
	dp := errorSum.DataPoints().AppendEmpty()
	dp.SetStartTimestamp(pcommon.NewTimestampFromTime(start))
	dp.SetTimestamp(pcommon.NewTimestampFromTime(end))
	dp.SetIntValue(atomic.LoadInt64(&prwParser.totalNans))
}

func (prwParser *PrometheusRemoteOtelParser) addGaugeMetrics(ilm pmetric.ScopeMetrics, family string, metrics []MetricData, metadata prompb.MetricMetadata) error {
	if nil == metrics {
		return fmt.Errorf("Nil metricsdata pointer! %s", family)
	}
	for _, metricsData := range metrics {
		if metricsData.MetricName == "" && prwParser.SfxGatewayCompatability {
			atomic.AddInt64(&prwParser.totalBadMetrics, 1)
			continue
		}
		nm := prwParser.scaffoldNewMetric(ilm, metricsData.MetricName, metadata)
		nm.SetName(metricsData.MetricName)
		gauge := nm.SetEmptyGauge()
		for _, sample := range metricsData.Samples {
			if math.IsNaN(sample.Value) && prwParser.SfxGatewayCompatability {
				atomic.AddInt64(&prwParser.totalNans, 1)
				continue
			}
			dp := gauge.DataPoints().AppendEmpty()
			dp.SetTimestamp(prometheusToOtelTimestamp(sample.GetTimestamp()))
			dp.SetStartTimestamp(prometheusToOtelTimestamp(sample.GetTimestamp()))
			prwParser.setFloatOrInt(dp, sample)
			prwParser.setAttributes(dp, metricsData.Labels)
		}
	}
	return nil
}

func (prwParser *PrometheusRemoteOtelParser) addCounterMetrics(ilm pmetric.ScopeMetrics, family string, metrics []MetricData, metadata prompb.MetricMetadata) error {
	if nil == metrics {
		return fmt.Errorf("Nil metricsdata pointer! %s", family)
	}
	for _, metricsData := range metrics {
		if metricsData.MetricName == "" && prwParser.SfxGatewayCompatability {
			atomic.AddInt64(&prwParser.totalBadMetrics, 1)
			continue
		}
		nm := prwParser.scaffoldNewMetric(ilm, metricsData.MetricName, metadata)
		sumMetric := nm.SetEmptySum()
		sumMetric.SetAggregationTemporality(pmetric.AggregationTemporalityCumulative)
		sumMetric.SetIsMonotonic(true)
		for _, sample := range metricsData.Samples {
			if math.IsNaN(sample.Value) && prwParser.SfxGatewayCompatability {
				atomic.AddInt64(&prwParser.totalNans, 1)
				continue
			}
			dp := nm.Sum().DataPoints().AppendEmpty()
			dp.SetTimestamp(prometheusToOtelTimestamp(sample.GetTimestamp()))
			dp.SetStartTimestamp(prometheusToOtelTimestamp(sample.GetTimestamp()))
			prwParser.setFloatOrInt(dp, sample)
			prwParser.setAttributes(dp, metricsData.Labels)
		}
	}
	return nil
}

func (prwParser *PrometheusRemoteOtelParser) addInfoStateset(ilm pmetric.ScopeMetrics, family string, metrics []MetricData, metadata prompb.MetricMetadata) error {
	var translationErrors []error
	if nil == metrics {
		translationErrors = append(translationErrors, fmt.Errorf("Nil metricsdata pointer! %s", family))
	}
	if translationErrors != nil {
		return multierr.Combine(translationErrors...)
	}
	for _, metricsData := range metrics {
		if metricsData.MetricName == "" && prwParser.SfxGatewayCompatability {
			atomic.AddInt64(&prwParser.totalBadMetrics, 1)
			continue
		}
		nm := prwParser.scaffoldNewMetric(ilm, metricsData.MetricName, metadata)
		// set as SUM but non-monotonic
		sumMetric := nm.SetEmptySum()
		sumMetric.SetIsMonotonic(false)
		sumMetric.SetAggregationTemporality(pmetric.AggregationTemporalityUnspecified)

		for _, sample := range metricsData.Samples {
			if math.IsNaN(sample.Value) && prwParser.SfxGatewayCompatability {
				atomic.AddInt64(&prwParser.totalNans, 1)
				continue
			}
			dp := sumMetric.DataPoints().AppendEmpty()
			dp.SetTimestamp(prometheusToOtelTimestamp(sample.GetTimestamp()))
			dp.SetStartTimestamp(prometheusToOtelTimestamp(sample.GetTimestamp()))
			prwParser.setFloatOrInt(dp, sample)
			prwParser.setAttributes(dp, metricsData.Labels)
		}
	}
	return nil
}

func getSampleTimestampBounds(samples []prompb.Sample) (int64, int64) {
	if len(samples) < 1 {
		return -1, -1
	}
	minTimestamp := int64(math.MaxInt64)
	maxTimestamp := int64(math.MinInt64)
	for _, sample := range samples {
		if minTimestamp > sample.Timestamp {
			minTimestamp = sample.Timestamp
		}
		if maxTimestamp < sample.GetTimestamp() {
			maxTimestamp = sample.GetTimestamp()
		}
	}
	return minTimestamp, maxTimestamp
}

func getWriteRequestTimestampBounds(request *prompb.WriteRequest) (time.Time, time.Time) {
	minTimestamp := int64(math.MaxInt64)
	maxTimestamp := int64(math.MinInt64)
	for _, ts := range request.Timeseries {
		sampleMin, sampleMax := getSampleTimestampBounds(ts.Samples)
		if sampleMin < minTimestamp {
			minTimestamp = sampleMin
		}
		if sampleMax > maxTimestamp {
			maxTimestamp = sampleMax
		}
	}
	return time.UnixMilli(minTimestamp), time.UnixMilli(maxTimestamp)
}

func (prwParser *PrometheusRemoteOtelParser) setFloatOrInt(dp pmetric.NumberDataPoint, sample prompb.Sample) error {
	if math.IsNaN(sample.Value) {
		return fmt.Errorf("NAN value found")
	}
	if float64(int64(sample.Value)) == sample.Value {
		dp.SetIntValue(int64(sample.Value))
	} else {
		dp.SetDoubleValue(sample.Value)
	}
	return nil
}

func prometheusToOtelTimestamp(ts int64) pcommon.Timestamp {
	return pcommon.Timestamp(ts * int64(time.Millisecond))
}

func (prwParser *PrometheusRemoteOtelParser) setAttributes(dp pmetric.NumberDataPoint, labels []prompb.Label) {
	for _, attr := range labels {
		if attr.Name != "__name__" {
			dp.Attributes().PutStr(attr.Name, attr.Value)
		}
	}
}

#!/usr/bin/env python3

"""Aggregator utilities for stripping aggregation function prefixes from metric names."""

import re

# Known aggregation functions used by Datadog / Prometheus.
# Only these are stripped by strip_aggregator so that metric names
# containing colons (e.g. recording rules like "job:http_requests_total:rate5m")
# are preserved.
_AGGREGATORS = {
    'sum', 'avg', 'min', 'max', 'count', 'stddev', 'stdvar',
    'topk', 'bottomk', 'quantile', 'rate', 'irate', 'increase',
    'delta', 'deriv', 'predict_linear', 'resets', 'changes',
    'holt_winters', 'histogram_quantile',
}


def strip_aggregator(metric_name: str) -> str:
    """Strip aggregation function prefix from metric name.

    Handles both Datadog format (e.g. 'sum:metric.name') and
    Prometheus format (e.g. 'sum(container_memory_working_set_bytes)').
    Only strips known aggregation functions to avoid stripping valid
    metric names that contain colons (e.g. recording rules like
    'job:http_requests_total:rate5m').
    """
    # Datadog format: agg:metric.name
    match = re.match(r'^(\w+):(.+)$', metric_name)
    if match and match.group(1) in _AGGREGATORS:
        return match.group(2)
    # Prometheus format: agg(metric)
    match = re.match(r'^(\w+)\((.+)\)$', metric_name)
    if match and match.group(1) in _AGGREGATORS:
        return match.group(2)
    return metric_name

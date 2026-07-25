#!/usr/bin/env python3

import pytest

from fittingjob.aggregator import strip_aggregator


@pytest.mark.parametrize(
    'metric_name, expected', [
        # Datadog format
        ('sum:container.memory.usage', 'container.memory.usage'),
        ('avg:system.cpu.user', 'system.cpu.user'),
        ('max:disk.io.read', 'disk.io.read'),
        # Prometheus format
        ('sum(container_memory_working_set_bytes)', 'container_memory_working_set_bytes'),
        ('avg(container_cpu_usage_seconds_total)', 'container_cpu_usage_seconds_total'),
        ('rate(container_cpu_usage_seconds_total[5m])', 'container_cpu_usage_seconds_total[5m]'),
        # No aggregator - plain metric name
        ('container_memory_working_set_bytes', 'container_memory_working_set_bytes'),
        # Metric names with colons (recording rules) should NOT be stripped
        ('job:http_requests_total:rate5m', 'job:http_requests_total:rate5m'),
        ('instance:process_cpu:rate5m', 'instance:process_cpu:rate5m'),
        # Non-aggregator prefix should NOT be stripped
        ('foo:bar.baz', 'foo:bar.baz'),
        ('foo(bar)', 'foo(bar)'),
    ]
)
def test_strip_aggregator(metric_name, expected):
    assert strip_aggregator(metric_name) == expected

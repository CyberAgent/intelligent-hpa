#!/usr/bin/env python3

import pytest

from fittingjob.prometheus import Prometheus


@pytest.mark.parametrize(
    'metric_name, tags, expected', [
        # Plain metric with tags
        (
            'container_cpu_usage_seconds_total',
            {'namespace': 'ihpa-test'},
            'container_cpu_usage_seconds_total{namespace="ihpa-test"}',
        ),
        # Aggregator with tags
        (
            'sum(container_cpu_usage_seconds_total)',
            {'namespace': 'ihpa-test'},
            'sum(container_cpu_usage_seconds_total{namespace="ihpa-test"})',
        ),
        # Range vector with tags
        (
            'rate(container_cpu_usage_seconds_total[5m])',
            {'namespace': 'ihpa-test'},
            'rate(container_cpu_usage_seconds_total{namespace="ihpa-test"}[5m])',
        ),
        # Nested aggregator with range vector
        (
            'sum(rate(container_cpu_usage_seconds_total[5m]))',
            {'namespace': 'ihpa-test'},
            'sum(rate(container_cpu_usage_seconds_total{namespace="ihpa-test"}[5m]))',
        ),
        # No tags returns metric as-is
        (
            'container_cpu_usage_seconds_total',
            {},
            'container_cpu_usage_seconds_total',
        ),
        # Multiple tags
        (
            'sum(container_memory_working_set_bytes)',
            {'namespace': 'ihpa-test', 'pod': 'nginx-abc'},
            'sum(container_memory_working_set_bytes{namespace="ihpa-test",pod="nginx-abc"})',
        ),
    ]
)
def test_build_query(metric_name, tags, expected):
    p = Prometheus(url='http://localhost:9090')
    result = p._build_query(metric_name, tags)
    assert result == expected

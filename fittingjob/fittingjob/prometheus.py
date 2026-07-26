#!/usr/bin/env python3

from datetime import datetime, timedelta
import json
from typing import List, Dict
from urllib import request, parse

from fittingjob import metrics_provider as mp

QUERY_RANGE_ENDPOINT = '/api/v1/query_range'


class Prometheus(mp.MetricsProvider):
    """
    Prometheus metrics provider for fetching metrics from a Prometheus server.

    Uses the query_range API to fetch historical metrics data over a time range
    at a specified step interval (default: 5 minutes, matching Datadog granularity).
    """

    def __init__(self, url: str, pushgateway_url: str = '', aggregation: str = ''):
        self.url = url
        self.pushgateway_url = pushgateway_url
        self.aggregation = aggregation

    def fetch_metrics(
            self,
            metrics_name: str,
            metrics_tags: Dict[str, str],
            before_days: int = 6,
            before_hours: int = 0,
            before_minutes: int = 0
    ) -> List[mp.Metric]:
        """
        fetch_metrics fetches metrics from Prometheus over the specified date range.
        The step interval is set to 300s (5 minutes) to match Datadog's granularity
        for multi-day queries.
        """
        end = datetime.now()
        start = end - timedelta(days=before_days, hours=before_hours, minutes=before_minutes)

        query = self._build_query(metrics_name, metrics_tags)

        params = {
            'query': query,
            'start': str(int(start.timestamp())),
            'end': str(int(end.timestamp())),
            'step': '300',
        }

        url_parts = list(parse.urlparse(self.url))
        url_parts[2] = url_parts[2].rstrip('/') + QUERY_RANGE_ENDPOINT
        url_parts[4] = parse.urlencode(params)
        req_url = parse.urlunparse(url_parts)

        req = request.Request(url=req_url, method='GET')
        with request.urlopen(req) as resp:
            j = json.loads(resp.read())

        if j.get('status') != 'success':
            print(f'prometheus query failed: {j}')
            return []

        result = j.get('data', {}).get('result', [])
        if len(result) == 0:
            print('no data returned from prometheus')
            return []

        # Aggregate values across multiple series by timestamp (sum).
        # If the query returns multiple series (e.g. missing aggregation or
        # grouping), this prevents mixing unrelated series together.
        timestamp_values: Dict[int, float] = {}
        for series in result:
            for point in series.get('values', []):
                ts = int(point[0])
                val = float(point[1])
                timestamp_values[ts] = timestamp_values.get(ts, 0.0) + val

        ms = []
        for ts in sorted(timestamp_values):
            d = datetime.fromtimestamp(ts)
            ms.append(mp.Metric(d, timestamp_values[ts]))

        return ms

    def _build_query(self, metrics_name: str, metrics_tags: Dict[str, str]) -> str:
        """
        Build a PromQL query from the metric name and tags.
        The metric_name may already contain an aggregation function
        (e.g., "sum(container_cpu_usage_seconds_total)").
        Labels are applied to the innermost metric selector, correctly
        handling nested aggregators (e.g. "sum(rate(metric[5m]))") and
        range-vectors (e.g. "rate(metric[5m])").
        """
        tags_str = tags_string(metrics_tags)
        if tags_str:
            return self._insert_labels(metrics_name, '{' + tags_str + '}')
        return metrics_name

    @staticmethod
    def _insert_labels(expr: str, label_selector: str) -> str:
        """
        Recursively find the innermost metric selector and insert the label
        selector before any range-vector suffix (e.g. [5m]).
        """
        # If the expression is wrapped in an aggregator (e.g. "sum(...)"),
        # recurse into the inner expression.
        if '(' in expr and expr.endswith(')'):
            open_idx = expr.index('(')
            inner = expr[open_idx + 1:-1]
            return expr[:open_idx + 1] + Prometheus._insert_labels(inner, label_selector) + ')'
        # At the innermost metric selector. Insert labels before any range-vector suffix.
        if '[' in expr:
            idx = expr.index('[')
            return expr[:idx] + label_selector + expr[idx:]
        return expr + label_selector


def tags_string(tags: Dict[str, str]) -> str:
    """
    tags_string generates Prometheus format labels from dict.
    e.g., {"namespace": "default"} -> 'namespace="default"'
    """
    return ','.join(f'{k}="{v}"' for k, v in tags.items())

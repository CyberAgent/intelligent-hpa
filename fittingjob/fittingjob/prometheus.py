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

        ms = []
        for series in result:
            for point in series.get('values', []):
                ts = int(point[0])
                val = float(point[1])
                d = datetime.fromtimestamp(ts)
                ms.append(mp.Metric(d, val))

        return ms

    def _build_query(self, metrics_name: str, metrics_tags: Dict[str, str]) -> str:
        """
        Build a PromQL query from the metric name and tags.
        The metric_name may already contain an aggregation function
        (e.g., "sum(container_cpu_usage_seconds_total)").
        Tags are added inside the curly braces of the metric selector.
        """
        tags_str = tags_string(metrics_tags)
        if tags_str:
            # If the metric_name contains an aggregation function like sum(...),
            # we need to insert the tags inside the inner metric selector.
            if '(' in metrics_name and metrics_name.endswith(')'):
                inner = metrics_name[metrics_name.index('(') + 1:-1]
                return f'{metrics_name[:metrics_name.index("(")]}({inner}{{{tags_str}}})'
            else:
                return f'{metrics_name}{{{tags_str}}}'
        return metrics_name


def tags_string(tags: Dict[str, str]) -> str:
    """
    tags_string generates Prometheus format labels from dict.
    e.g., {"namespace": "default"} -> 'namespace="default"'
    """
    return ','.join(f'{k}="{v}"' for k, v in tags.items())

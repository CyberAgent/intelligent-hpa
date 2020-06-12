#!/usr/bin/env python3

from datetime import datetime, timedelta
import http.client
import json
from typing import List, Dict
from urllib import request, parse

from fittingjob import metrics_provider as mp

QUERY_ENDPOINT = 'https://api.datadoghq.com/api/v1/query'


class Datadog(mp.MetricsProvider):
    """
    [Datadog metrics granularity]
     7 days:    per 1 hour
     6 days:    per 30 minutes
     2 days:    per 10 minutes
     1 day:     per 5 minutes
    12 hours:   per 2 minutes
     6 hours:   per 2 minutes
     3 hours:   per 1 minute
     1 hour:    per 20 seconds
    30 minutes: per 10 or 20 seconds
     5 minutes: 6 metrics per minute
     1 minute:  raw

    Non raw metrics is average of section between
    before and after.
    """

    def __init__(self, apikey: str, appkey: str):
        self.apikey = apikey
        self.appkey = appkey

    def fetch_metrics(
            self,
            metrics_name: str,
            metrics_tags: Dict[str, str],
            before_days: int = 6,
            before_hours: int = 0,
            before_minutes: int = 0
    ) -> List[mp.Metric]:
        """
        fetch_metrics fetches metrics every 5 minutes over specified date range.
        """
        ranges = separate_date_range_per_days(
            end=datetime.now(),
            days=before_days,
            hours=before_hours,
            minutes=before_minutes
        )

        ms = []
        for r in ranges:
            query = {
                'query': f'{metrics_name}{{{tags_string(metrics_tags)}}}',
                'from': str(int(r['start'].timestamp())),
                'to': str(int(r['end'].timestamp()))
            }

            with self.__get(QUERY_ENDPOINT, query) as resp:
                j = json.loads(resp.read())

            if len(j['series']) == 0:
                continue

            for p in j['series'][0]['pointlist']:
                unix_sec = p[0] / 1000
                unix_usec = (p[0] % 1000) * 1000
                d = datetime.fromtimestamp(unix_sec) + \
                    timedelta(microseconds=unix_usec)
                ms.append(mp.Metric(d, p[1]))

        return ms

    def __get(self, endpoint: str, query: Dict[str, str]) -> http.client.HTTPResponse:
        headers = {
            'DD-API-KEY': self.apikey,
            'DD-APPLICATION-KEY': self.appkey,
            'Content-Type': 'application/json'
        }

        url_parts = list(parse.urlparse(endpoint))
        url_parts[4] = parse.urlencode(query)

        req = request.Request(
            url=parse.urlunparse(url_parts),
            headers=headers,
            method='GET'
        )

        return request.urlopen(req)


def tags_string(tags: Dict[str, str]) -> str:
    """
    tags_string generates datadog format tags from dict.
    """
    s = ''
    for k in tags:
        s += f'{k}:{tags[k]},'

    return s.rstrip(',')


def separate_date_range_per_days(end: datetime, days: int, hours: int, minutes: int) -> List[Dict[str, datetime]]:
    """
    separate_date_range_per_days splits date by every day for getting fine grained datadog metrics.
    """
    start = end - timedelta(days=days, hours=hours, minutes=minutes)

    ranges = []
    while (end - start) > timedelta(days=1):
        e = start + timedelta(days=1)
        ranges.append({'start': start, 'end': e})
        start = e

    ranges.append({'start': start, 'end': end})
    return ranges

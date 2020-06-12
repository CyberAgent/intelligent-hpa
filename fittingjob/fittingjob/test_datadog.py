#!/usr/bin/env python3

from datetime import datetime
import pytest

from fittingjob import datadog


@pytest.mark.parametrize(
    'tags, expected', [
        (
            {'key': 'value'},
            'key:value'
        ),
        (
            {'key1': 'value1', 'key2': 'value2', 'key3': 'value3'},
            'key1:value1,key2:value2,key3:value3'
        ),
        (
            {},
            ''
        )
    ]
)
def test_tags_string(tags, expected):
    assert datadog.tags_string(tags) == expected


@pytest.mark.parametrize(
    'end, days, hours, minutes, expected', [
        (
            datetime(year=2019, month=12, day=25),
            5,
            0,
            0,
            [
                {
                    'start': datetime(year=2019, month=12, day=20),
                    'end': datetime(year=2019, month=12, day=21)
                },
                {
                    'start': datetime(year=2019, month=12, day=21),
                    'end': datetime(year=2019, month=12, day=22)
                },
                {
                    'start': datetime(year=2019, month=12, day=22),
                    'end': datetime(year=2019, month=12, day=23)
                },
                {
                    'start': datetime(year=2019, month=12, day=23),
                    'end': datetime(year=2019, month=12, day=24)
                },
                {
                    'start': datetime(year=2019, month=12, day=24),
                    'end': datetime(year=2019, month=12, day=25)
                },
            ]
        ),
        (
            datetime(year=2019, month=12, day=25, hour=3),
            5,
            12,
            0,
            [
                {
                    'start': datetime(year=2019, month=12, day=19, hour=15),
                    'end': datetime(year=2019, month=12, day=20, hour=15)
                },
                {
                    'start': datetime(year=2019, month=12, day=20, hour=15),
                    'end': datetime(year=2019, month=12, day=21, hour=15)
                },
                {
                    'start': datetime(year=2019, month=12, day=21, hour=15),
                    'end': datetime(year=2019, month=12, day=22, hour=15)
                },
                {
                    'start': datetime(year=2019, month=12, day=22, hour=15),
                    'end': datetime(year=2019, month=12, day=23, hour=15)
                },
                {
                    'start': datetime(year=2019, month=12, day=23, hour=15),
                    'end': datetime(year=2019, month=12, day=24, hour=15)
                },
                {
                    'start': datetime(year=2019, month=12, day=24, hour=15),
                    'end': datetime(year=2019, month=12, day=25, hour=3)
                },
            ]
        ),
        (
            datetime(year=2019, month=12, day=25),
            5,
            36,
            10,
            [
                {
                    'start': datetime(year=2019, month=12, day=18, hour=11, minute=50),
                    'end': datetime(year=2019, month=12, day=19, hour=11, minute=50)
                },
                {
                    'start': datetime(year=2019, month=12, day=19, hour=11, minute=50),
                    'end': datetime(year=2019, month=12, day=20, hour=11, minute=50)
                },
                {
                    'start': datetime(year=2019, month=12, day=20, hour=11, minute=50),
                    'end': datetime(year=2019, month=12, day=21, hour=11, minute=50)
                },
                {
                    'start': datetime(year=2019, month=12, day=21, hour=11, minute=50),
                    'end': datetime(year=2019, month=12, day=22, hour=11, minute=50)
                },
                {
                    'start': datetime(year=2019, month=12, day=22, hour=11, minute=50),
                    'end': datetime(year=2019, month=12, day=23, hour=11, minute=50)
                },
                {
                    'start': datetime(year=2019, month=12, day=23, hour=11, minute=50),
                    'end': datetime(year=2019, month=12, day=24, hour=11, minute=50)
                },
                {
                    'start': datetime(year=2019, month=12, day=24, hour=11, minute=50),
                    'end': datetime(year=2019, month=12, day=25)
                },
            ]
        )
    ]
)
def test_separate_date_range_per_days(end, days, hours, minutes, expected):
    assert datadog.separate_date_range_per_days(
        end, days, hours, minutes) == expected

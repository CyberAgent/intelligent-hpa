#!/usr/bin/env python3

from abc import ABCMeta, abstractmethod
from datetime import datetime
from typing import Dict, List

import pandas as pd


class Metric:
    def __init__(self, date: datetime, point: float):
        self.date = date
        self.point = point


class MetricsProvider(metaclass=ABCMeta):
    @abstractmethod
    def fetch_metrics(
            self,
            metrics_name: str,
            metrics_tags: Dict[str, str],
            before_days: int = 6,
            before_hours: int = 0,
            before_minutes: int = 0
    ) -> List[Metric]:
        pass


def save_csv(filename: str, metrics: List[Metric]):
    """
    save_csv saves metrics as csv to filename
    """
    with open(filename, mode='w') as f:
        f.write(f'ds,y\n')
        for m in metrics:
            f.write(f'{m.date},{m.point}\n')


def load_csv(filename: str) -> List[Metric]:
    """
    load_csv loads metrics from filename
    """
    metrics = []
    with open(filename, mode='r') as f:
        # skip header
        f.readline()
        while True:
            l = f.readline()
            if not l:
                break
            m = l.rstrip().split(',')
            metrics.append(Metric(m[0], m[1]))
    return metrics


def convert_dataframe(metrics: List[Metric]) -> pd.DataFrame:
    """
    convert_dataframe converts list of metric to pandas DataFrame
    """
    fields = ['date', 'point']
    columns = {'date': 'ds', 'point': 'y'}
    return pd.DataFrame([{f: getattr(m, f) for f in fields} for m in metrics]).rename(columns=columns)

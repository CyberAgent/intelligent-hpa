#!/usr/bin/env python3

from typing import Dict, List

import yaml

from fittingjob import datadog
from fittingjob import metrics_provider as mp


class Config:
    def __init__(
            self,
            provider: Dict[str, List[str]],
            dump_path: str,
            target_metrics_name: str,
            target_tags: Dict[str, str],
            seasonality: str,
            data_configmap_name: str,
            data_configmap_namespace: str,
            change_point_detection: Dict[str, str],
            custom_config: str,
            metrics_period: int = 7):
        self.provider = provider
        self.dump_path = dump_path
        self.target_metrics_name = target_metrics_name
        self.target_tags = target_tags
        self.seasonality = seasonality
        self.data_configmap_name = data_configmap_name
        self.data_configmap_namespace = data_configmap_namespace
        self.change_point_detection = change_point_detection
        self.custom_config = custom_config
        self.metrics_period = metrics_period

    def get_provider(self) -> mp.MetricsProvider:
        if len(self.provider) != 1:
            print(
                f'provider list must be specified only 1 entry ({len(self.provider)} entry exists)')
            return None

        for name in self.provider:
            if name == 'datadog':
                return datadog.Datadog(
                    apikey=self.provider[name]['apikey'],
                    appkey=self.provider[name]['appkey']
                )
        return None


def load(path: str) -> Config:
    with open(path, mode='r') as f:
        d = yaml.safe_load(f)

    print(d)

    return Config(
        provider=d.get('provider'),
        dump_path=d.get('dumpPath', 'model.pickle'),
        target_metrics_name=d.get('targetMetricsName'),
        target_tags=d.get('targetTags'),
        seasonality=d.get('seasonality'),
        data_configmap_name=d.get('dataConfigMapName'),
        data_configmap_namespace=d.get('dataConfigMapNamespace'),
        change_point_detection=d.get('changePointDetection', None),
        custom_config=d.get('customConfig', ""),
        metrics_period=d.get('metricsPeriod', 7)
    )

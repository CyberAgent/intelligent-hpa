#!/usr/bin/env python3

import sys

from fittingjob import model
from fittingjob import config
from fittingjob import configmap
from fittingjob import metrics_provider as mp


def run(config_path: str) -> int:
    cfg = config.load(config_path)

    provider = cfg.get_provider()
    print('fetch metrics...')
    metrics = provider.fetch_metrics(
        metrics_name=cfg.target_metrics_name,
        metrics_tags=cfg.target_tags,
        before_days=cfg.metrics_period
    )

    # Ex. save metrics data
    # mp.save_csv('metrics.csv', metrics)
    # return

    # Ex. load metrics data
    # metrics = mp.convert_dataframe(datadog.load_csv('metrics.csv'))

    df = mp.convert_dataframe(metrics)
    original_length = len(df)
    if original_length < 24*12:  # number of metrics over a day
        print(f'this prediction job is skipped (a few data: {original_length})')

    if cfg.change_point_detection is not None:
        print('change point detection...')
        c = cfg.change_point_detection
        sst = model.SingularSpectrumTransformation(
            window_size=c['windowSize'],
            trajectory_rows=c['trajectoryRows'],
            trajectory_features=c['trajectoryFeatures'],
            test_rows=c['testRows'],
            test_features=c['testFeatures'],
            lag=c['lag'],
        )
        sst.fit(df)
        tmpdf = sst.cutdown(c['percentageThreshold']/100.0)
        if len(tmpdf) < 24*12:
            print(f'cutdowned data is ignored (a few data: {len(tmpdf)})')
        else:
            df = tmpdf

    m = model.IHPAModel()
    transformed_length = len(df)
    print(f'fitting... (cutted: {transformed_length}/{original_length})')
    m.fit(df)
    print(m.cross_validation)

    print('forecasting...')
    forecasted_data = m.predict()
    print(forecasted_data.head())
    print(forecasted_data.tail())

    print(
        f'sending to {cfg.data_configmap_namespace}:{cfg.data_configmap_name} as {cfg.target_metrics_name}...')
    configmap.store_dataframe_to_configmap(
        cfg.data_configmap_name,
        cfg.data_configmap_namespace,
        cfg.target_metrics_name.lstrip('sum:'),
        forecasted_data,
    )

    # m.dump(cfg.dump_path)
    print('all task is succeeded')
    return 0


if __name__ == '__main__':
    if len(sys.argv) != 2:
        print(
            f'train requires only a config path (given {len(sys.argv)-1} args)')
        sys.exit(-1)

    config_path = sys.argv[1]
    sys.exit(run(config_path))

#!/usr/bin/env python3

import io

import pandas as pd
from kubernetes import client, config
from kubernetes.client.rest import ApiException


def store_dataframe_to_configmap(name: str, namespace: str, key: str, data: pd.DataFrame):
    sio = io.StringIO()
    data.to_csv(sio)

    config.load_incluster_config()

    corev1 = client.CoreV1Api()
    try:
        configmap = corev1.read_namespaced_config_map(name, namespace)
    except ApiException as e:
        print('failed to get configmap: {e}')
        raise e

    if configmap.data is None:
        configmap.data = {}

    configmap.data[key] = sio.getvalue()

    try:
        corev1.patch_namespaced_config_map(name, namespace, configmap)
    except ApiException as e:
        print(f'failed to update configmap: {e}')
        raise e

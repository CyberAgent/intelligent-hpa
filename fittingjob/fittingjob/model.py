#!/usr/bin/env python3

import pickle

from fbprophet import Prophet, diagnostics
import numpy as np
import pandas as pd


class IHPAModel:
    def __init__(self):
        # almost time series metrics might be additive on stable system
        self.model = Prophet(
            seasonality_mode='multiplicative',
            growth='linear',
            interval_width=0.8)
        self.train = pd.DataFrame()

    def fit(self, metrics: pd.DataFrame):
        self.train = metrics
        self.model.fit(self.train)

    def predict(self, minute_interval: int = 1, periods: int = 2) -> pd.DataFrame:
        """
        predict future metrics

        Args:
            minute_interval int:
                interval of future metrics.
                this preffered a divisor of 60 (minutes).
            periods int:
                days to predict

        Returns
            pd.DataFrame:
                DataFrame to store configmap
        """
        future = self.model.make_future_dataframe(
            freq=f'{minute_interval}T', periods=int((60/minute_interval)*24*periods))
        forecasted = self.model.predict(future)
        return self.__adapt_estimator_columns(forecasted)

    def __adapt_estimator_columns(self, df: pd.DataFrame) -> pd.DataFrame:
        """
        select columns and convert 'ds' to 'timestamp'
        """
        columns = ['ds', 'yhat', 'yhat_lower', 'yhat_upper']
        selected = df[columns].copy()
        selected['timestamp'] = selected['ds'].map(
            lambda x: int(x.timestamp()))
        selected.drop(columns='ds')
        return selected

    def dump(self, path: str):
        # TODO: return file md5 hash and
        #       check the integrity of the model when loading
        with open(path, mode='wb') as f:
            pickle.dump(self, f)

    def load(self, path: str):
        with open(path, mode='rb') as f:
            m = pickle.load(f)
        self.model = m.model
        self.train = m.train

    def cross_validation(
            self,
            initial: str = '3 days',
            period: str = '1 day',
            horizon: str = '1 day'):
        """
        EX.) initial: '3 days', period: '1 day', horizon: '2 day'
          |<- First Data Point
          |<---initial-->|
                              cutoff #1
          |<----train-data--->|<horizon>|
                                   cutoff #2
          |<----train-data-------->|<horizon>|
                                        cutoff #3
          |<----train-data------------->|<horizon>|
          ...
        --+----+----+----+----+----+----+----+----+----+----+----+----+--
          12   13   14   15   16   17   18   19   20   21   22   23   24th day
        """
        return diagnostics.cross_validation(
            self.model, initial=initial, period=period, horizon=horizon)


class SingularSpectrumTransformation:
    def __init__(
            self,
            window_size: int = 100,
            trajectory_rows: int = 50,
            trajectory_features: int = 5,
            test_rows: int = 50,
            test_features: int = 5,
            lag: int = 288):
        """
        SingularSpectrumTransformation(SST) for change point detection.

        Args:
            window_size int:
                size of sliding window for time series subset vector
            trajectory_rows int:
                number of sliding window of trajectory matrix
            trajectory_features int:
                number of singular value to select trajectory left-singular vevtors
            test_rows int:
                number of sliding window of test matrix
            test_features int:
                number of singular value to select test left-singular vevtors
            lag int:
                lag of trajectory and test matrices
        """
        if trajectory_features > window_size or test_features > window_size:
            print('window_size must be more greater than trajectory/test features')
            return
        self.window_size = window_size
        self.trajectory_rows = trajectory_rows
        self.trajectory_features = trajectory_features
        self.test_rows = test_rows
        self.test_features = test_features
        self.lag = lag

    def fit(self, data: pd.DataFrame):
        """
        calculate change point series of the given time series data

        Args:
            data pd.DataFrame:
                target dataframe
        """
        self.data = data.copy()
        self.change_rates = []
        for t in range(len(self.data)):
            x_idx = t - self.window_size - self.trajectory_features + 1
            z_idx = x_idx + self.lag - 1

            if z_idx+self.window_size+self.test_rows >= len(self.data) or x_idx < 0:
                self.change_rates.append(0)
                continue

            X = self.__window_matrix(
                self.data, self.window_size, x_idx, self.trajectory_rows)
            Z = self.__window_matrix(
                self.data, self.window_size, z_idx, self.test_rows)
            ux, _, _ = np.linalg.svd(X)
            uz, _, _ = np.linalg.svd(Z)
            U = ux[:, :self.trajectory_features]
            Q = uz[:, :self.test_features]

            _, s, _ = np.linalg.svd(np.matmul(U.T, Q))
            self.change_rates.append(1-s[0])

    def __window_matrix(self, data: pd.DataFrame, window_size: int, start_idx: int, rows: int) -> np.ndarray:
        mat = np.zeros((rows, window_size))
        for i in range(rows):
            tmp = data['y'].iloc[start_idx+i:start_idx+i+window_size].values
            mat[i] = tmp
        return mat.T

    def cutdown(self, threshold: float = 0.8) -> pd.DataFrame:
        """
        cutdown data before change point beyond thredhold

        Args:
            threshold float:
                threshold of change point.
                This parameter depends on the time series data,
                so we recommend you to check change point in practice.

        Returns
            pd.DataFrame:
                cutdowned DataFrame
        """
        idx = 0
        for i, cr in enumerate(self.change_rates[::-1]):
            if cr > threshold:
                idx = i
        return self.data[idx:].reset_index(drop=True)

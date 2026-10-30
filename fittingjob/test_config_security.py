import contextlib
import io
import json
import os
import tempfile
import unittest
from unittest.mock import patch

from fittingjob import config


class CredentialTests(unittest.TestCase):
    def test_credentials_from_environment_and_no_config_logging(self):
        payload = {"provider": {"datadog": {"apikey": "inline-api", "appkey": "inline-app"}}}
        with tempfile.NamedTemporaryFile(mode="w+") as source:
            json.dump(payload, source)
            source.flush()
            output = io.StringIO()
            with contextlib.redirect_stdout(output), patch.dict(os.environ, {"APIKey": "secret-api", "APPKey": "secret-app"}):
                provider = config.load(source.name).get_provider()
            self.assertEqual(provider.apikey, "secret-api")
            self.assertEqual(provider.appkey, "secret-app")
            self.assertEqual(output.getvalue(), "")

    def test_inline_compatibility(self):
        with patch.dict(os.environ, {}, clear=True):
            cfg = config.Config({"datadog": {"apikey": "inline-api", "appkey": "inline-app"}}, "", "", {}, "auto", "", "", {}, "")
            self.assertEqual(cfg.get_provider().apikey, "inline-api")
            self.assertEqual(cfg.get_provider().appkey, "inline-app")


if __name__ == "__main__":
    unittest.main()

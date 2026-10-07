import os
import unittest
from unittest.mock import patch
from fittingjob.datadog import query_endpoint, Datadog
import io
class SiteTests(unittest.TestCase):
    def test_default_and_sites(self):
        for site, host in [('', 'datadoghq.com'), ('datadoghq.com', 'datadoghq.com'), ('us5.datadoghq.com', 'us5.datadoghq.com'), ('datadoghq.eu', 'datadoghq.eu')]:
            with self.subTest(site=site), patch.dict(os.environ, {'DD_SITE': site}):
                self.assertEqual(query_endpoint(), 'https://api.' + host + '/api/v1/query')
    def test_fetch_uses_us5(self):
        with patch.dict(os.environ, {'DD_SITE': 'us5.datadoghq.com'}), patch('fittingjob.datadog.request.urlopen', side_effect=lambda req: io.BytesIO(b'{"series":[]}')) as send:
            Datadog('test-api', 'test-app').fetch_metrics('test.metric', {}, before_days=1)
            self.assertTrue(send.called)
            for call in send.call_args_list:
                self.assertTrue(call.args[0].full_url.startswith('https://api.us5.datadoghq.com/api/v1/query?'))
                self.assertNotIn('test-api', call.args[0].full_url)
    def test_unknown_site_rejected(self):
        with patch.dict(os.environ, {'DD_SITE': 'attacker.example'}):
            with self.assertRaises(ValueError):
                query_endpoint()
if __name__ == '__main__':
    unittest.main()

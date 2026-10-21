package datadog

import (
	"fmt"
	"net/http"
	"testing"
)

func TestDatadogBaseURL(t *testing.T) {
	for _, tt := range []struct {
		site, want string
		invalid    bool
	}{
		{"", BaseURL, false}, {"datadoghq.com", BaseURL, false},
		{"us5.datadoghq.com", "https://api.us5.datadoghq.com", false},
		{"datadoghq.eu", "https://api.datadoghq.eu", false},
		{"uk1.datadoghq.com", "https://api.uk1.datadoghq.com", false},
		{"attacker.example", "", true},
		{"ddog-gov.com", "", true},
		{"us2.ddog-gov.com", "", true},
	} {
		t.Run(tt.site, func(t *testing.T) {
			t.Setenv("DD_SITE", tt.site)
			got, err := datadogBaseURL()
			if (err != nil) != tt.invalid || got != tt.want {
				t.Fatalf("got %q, %v; want %q, invalid=%v", got, err, tt.want, tt.invalid)
			}
			if tt.invalid {
				wantErr := fmt.Sprintf("unsupported DD_SITE value: %q", tt.site)
				if err.Error() != wantErr {
					t.Fatalf("got error %q; want %q", err, wantErr)
				}
			}
		})
	}
}

type siteTransport struct{ host string }

func (s *siteTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	s.host = r.URL.Host
	return nil, fmt.Errorf("test transport stops request")
}
func TestDatadogUS5Requests(t *testing.T) {
	t.Setenv("DD_SITE", "us5.datadoghq.com")
	old := http.DefaultClient
	transport := &siteTransport{}
	http.DefaultClient = &http.Client{Transport: transport}
	t.Cleanup(func() { http.DefaultClient = old })
	d := &Datadog{}
	_, _ = d.Fetch("test.metric", 123, nil, nil)
	if transport.host != "api.us5.datadoghq.com" {
		t.Fatalf("Fetch host=%s", transport.host)
	}
	transport.host = ""
	_ = d.Send("test.metric", 123, 1, nil, nil)
	if transport.host != "api.us5.datadoghq.com" {
		t.Fatalf("Send host=%s", transport.host)
	}
}

func TestDatadogUK1Requests(t *testing.T) {
	t.Setenv("DD_SITE", "uk1.datadoghq.com")
	old := http.DefaultClient
	transport := &siteTransport{}
	http.DefaultClient = &http.Client{Transport: transport}
	t.Cleanup(func() { http.DefaultClient = old })
	d := &Datadog{}
	_, _ = d.Fetch("test.metric", 123, nil, nil)
	if transport.host != "api.uk1.datadoghq.com" {
		t.Fatalf("Fetch host=%s", transport.host)
	}
	transport.host = ""
	_ = d.Send("test.metric", 123, 1, nil, nil)
	if transport.host != "api.uk1.datadoghq.com" {
		t.Fatalf("Send host=%s", transport.host)
	}
}

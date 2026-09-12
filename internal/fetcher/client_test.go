package fetcher

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientFetchReleasesUsesInjectedEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "mode=json" {
			t.Fatalf("query = %q, want mode=json", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"version":"go1.23.4","stable":true,"files":[]}]`))
	}))
	defer server.Close()

	client := NewClient(server.Client())
	client.BaseURL = server.URL + "?mode=json"
	got, err := client.FetchReleases(context.Background(), false)
	if err != nil {
		t.Fatalf("FetchReleases() error = %v", err)
	}
	if len(got) != 1 || got[0].Version != "go1.23.4" {
		t.Fatalf("FetchReleases() = %+v", got)
	}
}

func TestClientFetchReleasesRejectsHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "temporary failure", http.StatusBadGateway)
	}))
	defer server.Close()

	client := NewClient(server.Client())
	client.BaseURL = server.URL
	_, err := client.FetchReleases(context.Background(), false)
	if err == nil {
		t.Fatal("FetchReleases() error = nil, want HTTP error")
	}
}

func TestClientFindReleaseByVersion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
			{"version":"go1.23.4","stable":true,"files":[]},
			{"version":"go1.22.9","stable":true,"files":[]}
		]`))
	}))
	defer server.Close()

	client := NewClient(server.Client())
	client.AllURL = server.URL
	got, err := client.FindReleaseByVersion(context.Background(), "1.23")
	if err != nil {
		t.Fatalf("FindReleaseByVersion() error = %v", err)
	}
	if got.Version != "go1.23.4" {
		t.Fatalf("version = %q, want go1.23.4", got.Version)
	}
}

func TestClientFindReleaseByVersionMatchesMinorBoundaries(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
			{"version":"go1.27.1","stable":true,"files":[]},
			{"version":"go1.200.1","stable":true,"files":[]},
			{"version":"go1.20.5","stable":true,"files":[]},
			{"version":"go1.20rc1","stable":false,"files":[]},
			{"version":"go1.2.9","stable":true,"files":[]}
		]`))
	}))
	defer server.Close()

	client := NewClient(server.Client())
	client.AllURL = server.URL

	tests := []struct {
		request string
		want    string
	}{
		{"1.2", "go1.2.9"},
		{"go1.2", "go1.2.9"},
		{"1.20", "go1.20.5"},
		{"1.20.5", "go1.20.5"},
		{"1.20rc1", "go1.20rc1"},
	}
	for _, tt := range tests {
		t.Run(tt.request, func(t *testing.T) {
			got, err := client.FindReleaseByVersion(context.Background(), tt.request)
			if err != nil {
				t.Fatalf("FindReleaseByVersion(%q) error = %v", tt.request, err)
			}
			if got.Version != tt.want {
				t.Fatalf("FindReleaseByVersion(%q) = %q, want %q", tt.request, got.Version, tt.want)
			}
		})
	}
}

func TestClientFindReleaseByVersionRejectsUnknownPartialVersion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"version":"go1.27.1","stable":true,"files":[]}]`))
	}))
	defer server.Close()

	client := NewClient(server.Client())
	client.AllURL = server.URL

	if _, err := client.FindReleaseByVersion(context.Background(), "1.2"); err == nil {
		t.Fatal("FindReleaseByVersion(\"1.2\") matched go1.27.1; want not-found error")
	}
}

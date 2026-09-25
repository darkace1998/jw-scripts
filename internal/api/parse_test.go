package api

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/darkace1998/jw-scripts/internal/config"
	"github.com/darkace1998/jw-scripts/internal/httpx"
)

// fakeMediator serves category JSON from a map of category key to response
// body; unknown keys return 404 and the key "Broken" returns 500.
func fakeMediator(t *testing.T, categories map[string]string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
		if key == "Broken" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		body, ok := categories[key]
		if !ok {
			http.NotFound(w, r)
			return
		}
		_, _ = fmt.Fprint(w, body)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func testClient(srv *httptest.Server, s *config.Settings) *Client {
	httpx.RetryDelay = time.Millisecond
	c := NewClient(s)
	c.baseURL = srv.URL
	c.pubMediaURL = srv.URL + "/pubmedia"
	c.httpClient = httpx.NewClient(5 * time.Second)
	return c
}

const rootCategory = `{"category":{"key":"Root","name":"Root","subcategories":[{"key":"Sub","name":"Sub"},{"key":"Gone","name":"Gone"}],
"media":[{"title":"Video","type":"video","primaryCategory":"Root","firstPublished":"2024-01-02T03:04:05.000Z",
"files":[{"progressiveDownloadURL":"https://x/v_r720P.mp4","label":"720p","filesize":10,"subtitles":{"url":"https://x/v.vtt"}}]}]}}`

const subCategory = `{"category":{"key":"Sub","name":"Sub","media":[{"title":"Video","type":"video","primaryCategory":"Root",
"firstPublished":"2024-01-02T03:04:05.000Z","files":[{"progressiveDownloadURL":"https://x/v_r720P.mp4","label":"720p","filesize":10}]}]}}`

func TestParseBroadcastingToleratesMissingSubcategory(t *testing.T) {
	srv := fakeMediator(t, map[string]string{"Root": rootCategory, "Sub": subCategory})
	c := testClient(srv, &config.Settings{Lang: "E", Quiet: 2, Quality: 720, IncludeCategories: []string{"Root"}})

	data, err := c.ParseBroadcasting()
	if err != nil {
		t.Fatalf("a missing subcategory should not be an error, got %v", err)
	}
	if len(data) != 2 {
		t.Fatalf("expected 2 categories, got %d", len(data))
	}
	var names []string
	for _, cat := range data {
		for _, item := range cat.Contents {
			if m, ok := item.(*Media); ok {
				names = append(names, m.Filename)
			}
		}
	}
	if len(names) != 2 || names[0] != "v_r720P.mp4" || names[1] != "v_r720P.mp4" {
		t.Errorf("the same video in two categories should map to one file, got %v", names)
	}
}

func TestParseBroadcastingReportsFailedCategories(t *testing.T) {
	srv := fakeMediator(t, map[string]string{"Root": rootCategory, "Sub": subCategory})
	c := testClient(srv, &config.Settings{Lang: "E", Quiet: 2, IncludeCategories: []string{"Root", "Missing", "Broken"}})

	data, err := c.ParseBroadcasting()
	if err == nil {
		t.Fatal("expected an error for a missing requested category and a server error")
	}
	if !errors.Is(err, ErrNotFound) || !strings.Contains(err.Error(), "500") {
		t.Errorf("error should mention both failures, got %v", err)
	}
	if len(data) == 0 {
		t.Error("expected partial results for the categories that loaded")
	}
}

func TestGetBroadcastingMP3sSkipsUnpublishedIssues(t *testing.T) {
	now := time.Now()
	current := (now.Year()-jwbStartYear)*12 + int(now.Month()) - jwbStartMonth + 1
	available := fmt.Sprintf("jwb-%d", current-1)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("pub") != available {
			http.NotFound(w, r)
			return
		}
		_, _ = fmt.Fprint(w, `{"files":{"E":{"MP3":[
			{"title":"Program","file":{"url":"https://x/jwb_E_1.mp3","modifiedDatetime":"2024-05-01 10:00:00"},"filesize":5,"track":1},
			{"title":"Program (Audio Description)","file":{"url":"https://x/jwb_E_ad.mp3"},"track":101}]}}}`)
	}))
	t.Cleanup(srv.Close)
	c := testClient(srv, &config.Settings{Lang: "E", Quiet: 2})

	data, err := c.GetBroadcastingMP3s()
	if err != nil {
		t.Fatalf("unpublished issues should not be errors, got %v", err)
	}
	if len(data) != 1 || len(data[0].Contents) != 1 {
		t.Fatalf("expected one category with one program, got %+v", data)
	}
	if m := data[0].Contents[0].(*Media); m.Filename != "jwb_E_1.mp3" || m.Date == 0 {
		t.Errorf("unexpected media %+v", m)
	}
}

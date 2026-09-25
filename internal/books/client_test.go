package books

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/darkace1998/jw-scripts/internal/config"
	"github.com/darkace1998/jw-scripts/internal/httpx"
)

// fakePubMedia serves publications keyed by "pub" or "pub/issue"; every other
// request gets a 404.
func fakePubMedia(t *testing.T, available map[string]bool) *Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		key := q.Get("pub")
		if issue := q.Get("issue"); issue != "" {
			key += "/" + issue
		}
		if !available[key] {
			http.NotFound(w, r)
			return
		}
		_, _ = fmt.Fprintf(w, `{"pubName":%q,"pub":%q,"issue":%q,"files":{"E":{"PDF":[{"title":"t","file":{"url":"https://x/%s.pdf"},"filesize":1}]}}}`,
			key, q.Get("pub"), q.Get("issue"), q.Get("pub"))
	}))
	t.Cleanup(srv.Close)

	httpx.RetryDelay = time.Millisecond
	c := NewClient(&config.Settings{Quiet: 2})
	c.baseURL = srv.URL
	return c
}

func withNow(t *testing.T, ts time.Time) {
	t.Helper()
	old := now
	now = func() time.Time { return ts }
	t.Cleanup(func() { now = old })
}

func TestCategoriesUseCurrentYear(t *testing.T) {
	withNow(t, time.Date(2026, 9, 23, 0, 0, 0, 0, time.UTC))
	cats, _ := NewClient(&config.Settings{}).GetCategories()
	want := map[string]string{
		"daily-text":       "es26",
		"yearbooks":        "dx26",
		"circuit-assembly": "ca-brpgm27",
		"convention":       "co-inv26",
	}
	for _, cat := range cats {
		if code, ok := want[cat.Key]; ok && cat.Publications[0] != code {
			t.Errorf("%s: publication %s, want %s", cat.Key, cat.Publications[0], code)
		}
	}
}

func TestGetCategoryFallsBackToPreviousEdition(t *testing.T) {
	withNow(t, time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC))
	c := fakePubMedia(t, map[string]bool{"dx25": true})

	cat, err := c.GetCategory("E", "yearbooks")
	if err != nil {
		t.Fatalf("GetCategory() returned error: %v", err)
	}
	if len(cat.Books) != 1 || cat.Books[0].ID != "dx25" {
		t.Errorf("expected fallback to dx25, got %+v", cat.Books)
	}
}

func TestGetCategoryReportsMissingPublications(t *testing.T) {
	withNow(t, time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC))
	c := fakePubMedia(t, map[string]bool{})

	cat, err := c.GetCategory("E", "daily-text")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
	if cat == nil || len(cat.Books) != 0 {
		t.Errorf("expected an empty category, got %+v", cat)
	}
}

func TestMagazinesUseLatestOrRequestedIssue(t *testing.T) {
	withNow(t, time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC))
	c := fakePubMedia(t, map[string]bool{"w/202601": true, "w/202512": true, "g/202501": true})

	cat, err := c.GetCategory("E", "magazines")
	if err == nil {
		t.Error("expected an error for Awake! with no issue in the last 12 months")
	}
	if len(cat.Books) != 1 || cat.Books[0].Issue != "202601" {
		t.Errorf("expected latest Watchtower issue 202601, got %+v", cat.Books)
	}

	c.settings.Issue = "202512"
	cat, _ = c.GetCategory("E", "magazines")
	if len(cat.Books) != 1 || cat.Books[0].Issue != "202512" {
		t.Errorf("expected requested issue 202512, got %+v", cat.Books)
	}
}

func TestGetBookSanitizesFilenames(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprint(w, `{"pub":"x","files":{"E":{"PDF":[{"file":{"url":"https://x/a/..%5C..%5Cevil.pdf"}}]}}}`)
	}))
	t.Cleanup(srv.Close)
	c := NewClient(&config.Settings{})
	c.baseURL = srv.URL

	book, err := c.GetBook("E", "x")
	if err != nil {
		t.Fatal(err)
	}
	if name := book.Files[0].Filename; name != "evil.pdf" {
		t.Errorf("Filename = %q, want backslashes removed", name)
	}
}

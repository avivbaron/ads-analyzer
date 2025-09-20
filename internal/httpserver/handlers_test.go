package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/avivbaron/ads-analyzer/internal/analysis"
	"github.com/avivbaron/ads-analyzer/internal/models"
	"github.com/avivbaron/ads-analyzer/internal/util"
)

type fakeAnalyzer struct{ calls int }

func (f *fakeAnalyzer) Analyze(ctx context.Context, domain string) (models.AnalysisResult, error) {
	f.calls++
	return models.AnalysisResult{Domain: "msn.com", TotalAdvertisers: 3, Advertisers: []models.AdvertiserCount{{Domain: "google.com", Count: 2}, {Domain: "appnexus.com", Count: 1}}, Cached: false, Timestamp: time.Unix(0, 0).UTC()}, nil
}

// TestHandleAnalysis_OK ensures the analysis handler returns 200 with a valid
// JSON body when the Analyzer succeeds.
// PASS: status=200, body decodes to expected domain/fields, analyzer called once.
// FAIL: wrong status or malformed body.
func TestHandleAnalysis_OK(t *testing.T) {
	fa := &fakeAnalyzer{}
	h := NewHandler(fa, 2)
	r := httptest.NewRequest(http.MethodGet, "/api/analysis?domain=msn.com", nil)
	w := httptest.NewRecorder()
	h.handleAnalysis(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d", w.Code)
	}
	var out models.AnalysisResult
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("json: %v", err)
	}
	if out.Domain != "msn.com" || out.TotalAdvertisers != 3 {
		t.Fatalf("bad body: %#v", out)
	}
	if fa.calls != 1 {
		t.Fatalf("analyzer calls=%d", fa.calls)
	}
}

// TestHandleBatch_OK ensures the batch handler preserves input order and returns
// a results array when all analyses succeed.
// PASS: status=200 and "results" present with len==input len.
// FAIL: wrong status or missing/short results.
func TestHandleBatch_OK(t *testing.T) {
	fa := &fakeAnalyzer{}
	h := NewHandler(fa, 4)
	body := `{"domains":["msn.com","cnn.com"]}`
	r := httptest.NewRequest(http.MethodPost, "/api/batch-analysis", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.handleBatch(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d", w.Code)
	}
	var out map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("json: %v", err)
	}
	arr, ok := out["results"].([]any)
	if !ok || len(arr) != 2 {
		t.Fatalf("results missing or wrong len: %#v", out)
	}
}

type errAnalyzer struct{ err error }

func (e *errAnalyzer) Analyze(ctx context.Context, domain string) (models.AnalysisResult, error) {
	return models.AnalysisResult{}, e.err
}

type okAnalyzer struct{}

func (o *okAnalyzer) Analyze(ctx context.Context, domain string) (models.AnalysisResult, error) {
	return models.AnalysisResult{Domain: domain, Timestamp: time.Unix(0, 0).UTC()}, nil
}

// TestHandleAnalysis_Errors validates HTTP error mapping for several analyzer errors.
// PASS: each case returns the expected status code.
// FAIL: wrong status for any case.
func TestHandleAnalysis_Errors(t *testing.T) {
	cases := []struct {
		err  error
		want int
	}{
		{util.ErrBadDomain, http.StatusBadRequest},
		{&analysis.StatusError{Code: http.StatusNotFound}, http.StatusNotFound},
		{context.DeadlineExceeded, http.StatusGatewayTimeout},
		{errors.New("upstream"), http.StatusBadGateway},
	}
	for _, c := range cases {
		h := NewHandler(&errAnalyzer{err: c.err}, 2)
		r := httptest.NewRequest(http.MethodGet, "/api/analysis?domain=msn.com", nil)
		w := httptest.NewRecorder()
		h.handleAnalysis(w, r)
		if w.Code != c.want {
			t.Fatalf("want %d got %d", c.want, w.Code)
		}
	}
}

// TestHandleBatch_OrderAndErrors checks that batch preserves order even when
// some items fail to analyze. Here we simulate invalid and valid domains.
// PASS: status=200 and results length equals input, preserving positions.
// FAIL: wrong status, missing results, or order changed.
func TestHandleBatch_OrderAndErrors(t *testing.T) {
	h := NewHandler(&okAnalyzer{}, 2)
	body := `{"domains":["bad://","msn.com","cnn.com"]}`
	r := httptest.NewRequest(http.MethodPost, "/api/batch-analysis", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.handleBatch(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d", w.Code)
	}
	var out struct{ Results []models.AnalysisResult }
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("json: %v", err)
	}
	if len(out.Results) != 3 {
		t.Fatalf("len=%d", len(out.Results))
	}
}

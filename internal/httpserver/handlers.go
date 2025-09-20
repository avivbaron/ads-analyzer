package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/avivbaron/ads-analyzer/internal/analysis"
	"github.com/avivbaron/ads-analyzer/internal/models"
	"github.com/avivbaron/ads-analyzer/internal/util"
)

// Analyzer is the minimal interface our handlers need.
// analysis.Service satisfies this automatically.
type Analyzer interface {
	Analyze(ctx context.Context, domain string) (models.AnalysisResult, error)
}

type Handler struct {
	analyzer     Analyzer
	batchWorkers int
}

func NewHandler(a Analyzer, batchWorkers int) *Handler {
	if batchWorkers <= 0 {
		batchWorkers = 1
	}
	return &Handler{analyzer: a, batchWorkers: batchWorkers}
}

// GET /api/analysis?domain=...
func (h *Handler) handleAnalysis(w http.ResponseWriter, r *http.Request) {
	domain := r.URL.Query().Get("domain")
	if domain == "" {
		writeError(w, http.StatusBadRequest, "missing domain parameter")
		return
	}
	ctx := r.Context()
	res, err := h.analyzer.Analyze(ctx, domain)
	if err != nil {
		writeAnalyzeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// POST /api/batch-analysis
// {"domains":["msn.com","cnn.com"]}
func (h *Handler) handleBatch(w http.ResponseWriter, r *http.Request) {
	var req models.BatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if len(req.Domains) == 0 {
		writeError(w, http.StatusBadRequest, "domains list is empty")
		return
	}

	ctx := r.Context()
	type item struct {
		idx int
		res models.AnalysisResult
		err error
	}

	jobs := make(chan int)
	out := make(chan item)

	wg := sync.WaitGroup{}
	worker := func() {
		defer wg.Done()
		for idx := range jobs {
			res, err := h.analyzer.Analyze(ctx, req.Domains[idx])
			out <- item{idx: idx, res: res, err: err}
		}
	}
	workers := h.batchWorkers
	if workers > len(req.Domains) {
		workers = len(req.Domains)
	}
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go worker()
	}

	go func() {
		for i := range req.Domains {
			jobs <- i
		}
		close(jobs)
		wg.Wait()
		close(out)
	}()

	results := make([]models.AnalysisResult, len(req.Domains))
	for it := range out {
		if it.err != nil {
			// Represent errors as zero-result placeholder with timestamp
			d := req.Domains[it.idx]
			results[it.idx] = models.AnalysisResult{Domain: d, TotalAdvertisers: 0, Advertisers: nil, Cached: false, Timestamp: time.Now().UTC()}
			continue
		}
		results[it.idx] = it.res
	}

	writeJSON(w, http.StatusOK, models.BatchResponse{Results: results})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]any{"error": msg})
}

func writeAnalyzeErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, util.ErrBadDomain):
		writeError(w, http.StatusBadRequest, "invalid domain")
		return
	}
	var se *analysis.StatusError
	if errors.As(err, &se) {
		if se.Code == http.StatusNotFound {
			writeError(w, http.StatusNotFound, "ads.txt not found")
			return
		}
	}
	if errors.Is(err, context.DeadlineExceeded) {
		writeError(w, http.StatusGatewayTimeout, "fetch timeout")
		return
	}
	writeError(w, http.StatusBadGateway, err.Error())
}

package analysis

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/avivbaron/ads-analyzer/internal/metrics"
)

type Fetcher interface {
	GetAdsTxt(ctx context.Context, domain string) ([]byte, error)
}

type httpFetcher struct {
	client       *http.Client
	httpFallback bool
}

func NewHTTPFetcher(timeout time.Duration, httpFallback bool) Fetcher {
	c := &http.Client{
		Timeout: timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return errors.New("stopped after 5 redirects")
			}
			return nil
		},
	}
	return &httpFetcher{client: c, httpFallback: httpFallback}
}

func (f *httpFetcher) GetAdsTxt(ctx context.Context, domain string) ([]byte, error) {
	urls := []string{"https://" + domain + "/ads.txt"}
	if f.httpFallback {
		urls = append(urls, "http://"+domain+"/ads.txt")
	}

	var lastErr error

	for _, u := range urls {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			lastErr = err
			continue
		}

		start := time.Now()
		resp, err := f.client.Do(req)
		metrics.ObserveFetch(req.URL.Scheme, start)
		if err != nil {
			lastErr = err
			continue
		}
		defer resp.Body.Close()

		switch resp.StatusCode {
		case http.StatusOK:
			b, err := io.ReadAll(resp.Body)
			if err != nil {
				lastErr = err
				continue
			}
			return b, nil

		case http.StatusNotFound:
			lastErr = fmt.Errorf("ads.txt not found (%s): %w", u, &StatusError{Code: http.StatusNotFound})

		default:
			lastErr = fmt.Errorf("bad status %d from %s", resp.StatusCode, u)
		}
	}

	return nil, lastErr
}

type StatusError struct {
	Code int
}

func (e *StatusError) Error() string {
	return fmt.Sprintf("http %d", e.Code)
}

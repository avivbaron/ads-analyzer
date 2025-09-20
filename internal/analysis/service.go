package analysis

import (
	"context"
	"sort"
	"time"

	"github.com/avivbaron/ads-analyzer/internal/cache"
	"github.com/avivbaron/ads-analyzer/internal/metrics"
	"github.com/avivbaron/ads-analyzer/internal/models"
	"github.com/avivbaron/ads-analyzer/internal/util"
)

type Service struct {
	cache   cache.Cache
	fetcher Fetcher
	ttl     time.Duration
}

func NewService(c cache.Cache, f Fetcher, ttl time.Duration) *Service {
	return &Service{cache: c, fetcher: f, ttl: ttl}
}

func (s *Service) Analyze(ctx context.Context, rawDomain string) (models.AnalysisResult, error) {
	var res models.AnalysisResult

	domain, err := util.NormalizeDomain(rawDomain)
	if err != nil {
		return res, err
	}

	cacheKey := "analysis:" + domain
	hit, _ := s.cache.Get(ctx, cacheKey, &res)
	if hit {
		metrics.IncHit("analysis")
		res.Cached = true
		return res, nil
	}
	metrics.IncMiss("analysis")

	b, err := s.fetcher.GetAdsTxt(ctx, domain)
	if err != nil {
		return res, err
	}
	counts := ParseAdsTxt(b)
	var list []models.AdvertiserCount
	list = make([]models.AdvertiserCount, 0, len(counts))
	total := 0
	for d, c := range counts {
		list = append(list, models.AdvertiserCount{Domain: d, Count: c})
		total += c
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].Count != list[j].Count {
			return list[i].Count > list[j].Count
		}
		return list[i].Domain < list[j].Domain
	})

	res = models.AnalysisResult{
		Domain:           domain,
		TotalAdvertisers: total,
		Advertisers:      list,
		Cached:           false,
		Timestamp:        time.Now().UTC(),
	}
	_ = s.cache.Set(ctx, cacheKey, res, s.ttl)
	return res, nil
}

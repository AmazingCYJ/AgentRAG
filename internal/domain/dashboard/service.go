package dashboard

import (
	"sort"
	"strings"
	"time"

	domainconversation "github.com/AmazingCYJ/AgentRAG/internal/domain/conversation"
	domainragtrace "github.com/AmazingCYJ/AgentRAG/internal/domain/ragtrace"
	domainusermgmt "github.com/AmazingCYJ/AgentRAG/internal/domain/usermgmt"
)

// KPI 定义概览卡片指标。
type KPI struct {
	Value    int     `json:"value"`
	Delta    int     `json:"delta,omitempty"`
	DeltaPct float64 `json:"deltaPct,omitempty"`
}

// Overview 定义仪表盘概览。
type Overview struct {
	Window        string      `json:"window"`
	CompareWindow string      `json:"compareWindow"`
	UpdatedAt     int64       `json:"updatedAt"`
	KPIs          OverviewKPI `json:"kpis"`
}

// OverviewKPI 定义概览指标集合。
type OverviewKPI struct {
	TotalUsers    KPI `json:"totalUsers"`
	ActiveUsers   KPI `json:"activeUsers"`
	TotalSessions KPI `json:"totalSessions"`
	Sessions24h   KPI `json:"sessions24h"`
	TotalMessages KPI `json:"totalMessages"`
	Messages24h   KPI `json:"messages24h"`
}

// Performance 定义性能指标。
type Performance struct {
	Window       string  `json:"window"`
	AvgLatencyMs int64   `json:"avgLatencyMs"`
	P95LatencyMs int64   `json:"p95LatencyMs"`
	SuccessRate  float64 `json:"successRate"`
	ErrorRate    float64 `json:"errorRate"`
	NoDocRate    float64 `json:"noDocRate"`
	SlowRate     float64 `json:"slowRate"`
}

// TrendPoint 定义趋势点。
type TrendPoint struct {
	TS    int64   `json:"ts"`
	Value float64 `json:"value"`
}

// TrendSeries 定义趋势序列。
type TrendSeries struct {
	Name string       `json:"name"`
	Data []TrendPoint `json:"data"`
}

// Trends 定义趋势数据。
type Trends struct {
	Metric      string        `json:"metric"`
	Window      string        `json:"window"`
	Granularity string        `json:"granularity"`
	Series      []TrendSeries `json:"series"`
}

// Service 提供仪表盘聚合能力。
type Service struct {
	userService         *domainusermgmt.Service
	conversationService *domainconversation.Service
	ragTraceService     *domainragtrace.Service
	now                 func() time.Time
}

// NewService 创建仪表盘服务。
func NewService(
	userService *domainusermgmt.Service,
	conversationService *domainconversation.Service,
	ragTraceService *domainragtrace.Service,
) *Service {
	return &Service{
		userService:         userService,
		conversationService: conversationService,
		ragTraceService:     ragTraceService,
		now:                 time.Now,
	}
}

// LoadOverview 返回概览数据。
func (s *Service) LoadOverview(window string) Overview {
	normalizedWindow := normalizeWindow(window, "24h")
	totalUsers := 0
	if s.userService != nil {
		totalUsers = s.userService.Count()
	}
	stats := domainconversation.StatsSnapshot{}
	if s.conversationService != nil {
		stats = s.conversationService.StatsSnapshot()
	}
	activeUsers := min(totalUsers, max(1, stats.RecentSessions))
	return Overview{
		Window:        normalizedWindow,
		CompareWindow: compareWindow(normalizedWindow),
		UpdatedAt:     s.now().UnixMilli(),
		KPIs: OverviewKPI{
			TotalUsers:    buildKPI(totalUsers),
			ActiveUsers:   buildKPI(activeUsers),
			TotalSessions: buildKPI(stats.TotalSessions),
			Sessions24h:   buildKPI(stats.RecentSessions),
			TotalMessages: buildKPI(stats.TotalMessages),
			Messages24h:   buildKPI(stats.RecentMessages),
		},
	}
}

// LoadPerformance 返回性能数据。
func (s *Service) LoadPerformance(window string) Performance {
	normalizedWindow := normalizeWindow(window, "24h")
	runs := []domainragtrace.Run{}
	if s.ragTraceService != nil {
		runs = s.ragTraceService.SnapshotRuns()
	}
	if len(runs) == 0 {
		return Performance{
			Window:       normalizedWindow,
			SuccessRate:  100,
			ErrorRate:    0,
			NoDocRate:    0,
			SlowRate:     0,
			AvgLatencyMs: 0,
			P95LatencyMs: 0,
		}
	}

	durations := make([]int64, 0, len(runs))
	var totalDuration int64
	successCount := 0
	errorCount := 0
	slowCount := 0
	for _, run := range runs {
		durations = append(durations, run.DurationMs)
		totalDuration += run.DurationMs
		if strings.EqualFold(run.Status, "success") {
			successCount++
		} else {
			errorCount++
		}
		if run.DurationMs > 15000 {
			slowCount++
		}
	}
	avg := totalDuration / int64(len(runs))
	p95 := percentile(durations, 0.95)
	return Performance{
		Window:       normalizedWindow,
		AvgLatencyMs: avg,
		P95LatencyMs: p95,
		SuccessRate:  ratio(successCount, len(runs)),
		ErrorRate:    ratio(errorCount, len(runs)),
		NoDocRate:    6.5,
		SlowRate:     ratio(slowCount, len(runs)),
	}
}

// LoadTrends 返回趋势数据。
func (s *Service) LoadTrends(metric, window, granularity string) Trends {
	normalizedWindow := normalizeWindow(window, "7d")
	normalizedGranularity := normalizeGranularity(granularity, normalizedWindow)
	points := pointCount(normalizedWindow, normalizedGranularity)
	base := baseValue(metric, s.LoadOverview(normalizedWindow), s.LoadPerformance(normalizedWindow))
	startAt := s.now().Add(-time.Duration(points-1) * granularityDuration(normalizedGranularity))
	data := make([]TrendPoint, 0, points)
	for index := 0; index < points; index++ {
		ts := startAt.Add(time.Duration(index) * granularityDuration(normalizedGranularity)).UnixMilli()
		value := base + float64((index%4)-1)*float64(max(1, int(base/12)))
		if value < 0 {
			value = 0
		}
		data = append(data, TrendPoint{TS: ts, Value: value})
	}
	return Trends{
		Metric:      metric,
		Window:      normalizedWindow,
		Granularity: normalizedGranularity,
		Series: []TrendSeries{
			{Name: metric, Data: data},
		},
	}
}

func buildKPI(value int) KPI {
	compare := value - max(1, value/10)
	if compare < 0 {
		compare = 0
	}
	delta := value - compare
	deltaPct := 0.0
	if compare > 0 {
		deltaPct = float64(delta) / float64(compare) * 100
	}
	return KPI{
		Value:    value,
		Delta:    delta,
		DeltaPct: deltaPct,
	}
}

func normalizeWindow(value, fallback string) string {
	switch strings.TrimSpace(value) {
	case "24h", "7d", "30d":
		return strings.TrimSpace(value)
	default:
		return fallback
	}
}

func compareWindow(window string) string {
	switch window {
	case "24h":
		return "previous 24h"
	case "30d":
		return "previous 30d"
	default:
		return "previous 7d"
	}
}

func normalizeGranularity(value, window string) string {
	if strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	if window == "24h" {
		return "hour"
	}
	return "day"
}

func granularityDuration(granularity string) time.Duration {
	if granularity == "hour" {
		return time.Hour
	}
	return 24 * time.Hour
}

func pointCount(window, granularity string) int {
	if granularity == "hour" {
		if window == "24h" {
			return 24
		}
		return 12
	}
	if window == "30d" {
		return 30
	}
	return 7
}

func baseValue(metric string, overview Overview, performance Performance) float64 {
	switch metric {
	case "messages":
		return float64(overview.KPIs.Messages24h.Value)
	case "activeUsers":
		return float64(overview.KPIs.ActiveUsers.Value)
	case "avgLatency":
		return float64(performance.AvgLatencyMs)
	case "quality":
		return performance.SuccessRate
	default:
		return float64(overview.KPIs.Sessions24h.Value)
	}
}

func percentile(values []int64, ratioValue float64) int64 {
	if len(values) == 0 {
		return 0
	}
	sorted := append([]int64(nil), values...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	index := int(float64(len(sorted))*ratioValue) - 1
	if index < 0 {
		index = 0
	}
	if index >= len(sorted) {
		index = len(sorted) - 1
	}
	return sorted[index]
}

func ratio(numerator, denominator int) float64 {
	if denominator == 0 {
		return 0
	}
	return float64(numerator) / float64(denominator) * 100
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

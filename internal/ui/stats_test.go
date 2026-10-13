package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/luvcie/codedrill/internal/drill"
	"github.com/luvcie/codedrill/internal/storage"
)

func TestRenderStatsDashboard(t *testing.T) {
	hist := &storage.HistoryStore{
		Attempts: []storage.Attempt{},
		Stats:    make(map[string]*storage.DrillStats),
	}

	drills := []drill.Drill{
		{ID: "ft_strlen", Title: "ft_strlen", Language: "c"},
	}

	emptyView := RenderStatsDashboard(hist, drills, 90)
	if !strings.Contains(emptyView, "SPEEDRUNNER PROFILE") {
		t.Errorf("expected title in empty stats dashboard")
	}
	if !strings.Contains(emptyView, "LEVEL") {
		t.Errorf("expected LEVEL badge in stats dashboard")
	}
	if !strings.Contains(emptyView, "XP") {
		t.Errorf("expected XP in stats dashboard")
	}
	if !strings.Contains(emptyView, "TOTAL SPRINTS") {
		t.Errorf("expected TOTAL SPRINTS card in empty stats dashboard")
	}
	if !strings.Contains(emptyView, "WIN STREAK") {
		t.Errorf("expected WIN STREAK card in empty stats dashboard")
	}
	if !strings.Contains(emptyView, "TODAY") {
		t.Errorf("expected TODAY card in empty stats dashboard")
	}
	if !strings.Contains(emptyView, "No runs yet") {
		t.Errorf("expected notice in empty stats dashboard")
	}

	hist.Attempts = append(hist.Attempts, storage.Attempt{
		DrillID:   "ft_strlen",
		Duration:  2 * time.Second,
		Passed:    true,
		Timestamp: time.Now(),
		Attempts:  1,
	})
	hist.Stats["ft_strlen"] = &storage.DrillStats{
		DrillID:      "ft_strlen",
		PersonalBest: 2 * time.Second,
		TotalRuns:    1,
		Successful:   1,
		LastAttempt:  time.Now(),
	}

	narrowView := RenderStatsDashboard(hist, drills, 80)
	if !strings.Contains(narrowView, "SPEEDRUNNER PROFILE") {
		t.Errorf("expected title in narrow view")
	}
	if !strings.Contains(narrowView, "PRACTICE TIME") {
		t.Errorf("expected PRACTICE TIME in narrow view")
	}
	if !strings.Contains(narrowView, "TODAY") {
		t.Errorf("expected TODAY in narrow view")
	}

	wideView := RenderStatsDashboard(hist, drills, 140)
	if !strings.Contains(wideView, "SPEEDRUNNER PROFILE") {
		t.Errorf("expected title in wide view")
	}
	if !strings.Contains(wideView, "CLEARED") {
		t.Errorf("expected CLEARED in wide view")
	}
	if !strings.Contains(wideView, "TODAY") {
		t.Errorf("expected TODAY in wide view")
	}

	ultraNarrowView := RenderStatsDashboard(hist, drills, 60)
	if !strings.Contains(ultraNarrowView, "SPEEDRUNNER PROFILE") {
		t.Errorf("expected title in ultra-narrow view")
	}
	if !strings.Contains(ultraNarrowView, "DRILL") {
		t.Errorf("expected DRILL in ultra-narrow view")
	}

}

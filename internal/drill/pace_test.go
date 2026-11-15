package drill

import (
	"slices"
	"testing"
	"time"
)

func TestVect2RankBoundaries(t *testing.T) {
	d := Drill{ID: "exam05-vect2", Difficulty: "hard"}
	boundaries := []struct {
		rank string
		time time.Duration
	}{
		{"D", 0},
		{"D+", 30 * time.Minute},
		{"C-", 25 * time.Minute},
		{"C", 22 * time.Minute},
		{"C+", 19 * time.Minute},
		{"B-", 16 * time.Minute},
		{"B", 14 * time.Minute},
		{"B+", 12 * time.Minute},
		{"A-", 10 * time.Minute},
		{"A", 9 * time.Minute},
		{"A+", 8 * time.Minute},
		{"S-", 7 * time.Minute},
		{"S", 6 * time.Minute},
		{"S+", 5 * time.Minute},
		{"SS", 4 * time.Minute},
		{"U", 3 * time.Minute},
		{"X", 150 * time.Second},
		{"X+", 2 * time.Minute},
	}
	for i, boundary := range boundaries[1:] {
		t.Run(boundary.rank, func(t *testing.T) {
			for _, offset := range []time.Duration{-time.Nanosecond, 0, time.Nanosecond} {
				want := boundaries[i].rank
				if offset < 0 {
					want = boundary.rank
				}
				if got := CalculateRank(boundary.time+offset, d); got != want {
					t.Errorf("rank at %v %+v = %s, want %s", boundary.time, offset, got, want)
				}
			}
		})
	}
	for duration, want := range map[time.Duration]string{
		-time.Second: "", 0: "", time.Nanosecond: "X+", time.Hour: "D",
	} {
		if got := CalculateRank(duration, d); got != want {
			t.Errorf("rank at %v = %q, want %q", duration, got, want)
		}
	}
}

func TestBuiltinRankScales(t *testing.T) {
	want := []string{"D", "D+", "C-", "C", "C+", "B-", "B", "B+", "A-", "A", "A+", "S-", "S", "S+", "SS", "U", "X"}
	drills, err := LoadBuiltins()
	if err != nil {
		t.Fatal(err)
	}
	if len(rankSeconds) != len(drills) {
		t.Fatalf("%d schedules for %d built-in exercises", len(rankSeconds), len(drills))
	}
	for _, d := range drills {
		t.Run(d.ID, func(t *testing.T) {
			seconds, ok := rankSeconds[d.ID]
			if !ok {
				t.Fatal("built-in exercise has no individual calibration")
			}
			for i, cutoff := range seconds {
				if cutoff <= 0 || i > 0 && cutoff >= seconds[i-1] {
					t.Fatalf("cutoffs must be positive and strictly decreasing: %v", seconds)
				}
			}
			targets := RankTargets(d)
			var got []string
			for _, target := range targets {
				got = append(got, target.Rank)
				if target.Inclusive {
					t.Errorf("%s cutoff is inclusive", target.Rank)
				}
			}
			if !slices.Equal(got, want) {
				t.Fatalf("public ranks = %v, want %v", got, want)
			}
			if targets[0].Time != 0 {
				t.Fatal("D must allow any successful completion")
			}
			for i, target := range targets[1:] {
				for _, offset := range []time.Duration{-time.Nanosecond, 0, time.Nanosecond} {
					want := targets[i].Rank
					if offset < 0 {
						want = target.Rank
					}
					if got := CalculateRank(target.Time+offset, d); got != want {
						t.Errorf("rank at %v %+v = %s, want %s", target.Time, offset, got, want)
					}
				}
			}
			secret := time.Duration(seconds[len(seconds)-1]) * time.Second
			for _, offset := range []time.Duration{-time.Nanosecond, 0, time.Nanosecond} {
				want := "X"
				if offset < 0 {
					want = "X+"
				}
				if got := CalculateRank(secret+offset, d); got != want {
					t.Errorf("rank at secret boundary %+v = %s, want %s", offset, got, want)
				}
			}
			for duration, want := range map[time.Duration]string{
				-time.Second: "", 0: "", time.Nanosecond: "X+", 24 * time.Hour: "D",
			} {
				if got := CalculateRank(duration, d); got != want {
					t.Errorf("rank at %v = %q, want %q", duration, got, want)
				}
			}
		})
	}
}

func TestCustomDrillsKeepDifficultyRanks(t *testing.T) {
	for _, tc := range []struct {
		difficulty string
		s, a, b    time.Duration
	}{
		{"easy", 4 * time.Second, 8 * time.Second, 15 * time.Second},
		{"medium", 12 * time.Second, 25 * time.Second, 50 * time.Second},
		{"hard", 25 * time.Second, 50 * time.Second, 90 * time.Second},
		{"MEDIUM", 12 * time.Second, 25 * time.Second, 50 * time.Second},
		{"custom", 4 * time.Second, 8 * time.Second, 15 * time.Second},
	} {
		for _, id := range []string{"custom-strlen", "custom-life", "custom-vect2"} {
			d := Drill{ID: id, Title: "vect2", Difficulty: tc.difficulty}
			for _, boundary := range []struct {
				time        time.Duration
				rank, after string
			}{
				{tc.s, "S", "A"}, {tc.a, "A", "B"}, {tc.b, "B", "C"},
			} {
				for _, offset := range []time.Duration{-time.Nanosecond, 0, time.Nanosecond} {
					want := boundary.rank
					if offset > 0 {
						want = boundary.after
					}
					if got := CalculateRank(boundary.time+offset, d); got != want {
						t.Errorf("%s/%s at %v %+v = %s, want %s", id, tc.difficulty, boundary.time, offset, got, want)
					}
				}
			}
		}
	}
}

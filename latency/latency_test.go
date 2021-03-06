package latency

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/openconfig/gnmi/errdiff"
	"github.com/openconfig/gnmi/metadata"
)

func TestLatencyWithoutWindows(t *testing.T) {
	defer func() {
		now = time.Now
	}()
	var windows []time.Duration
	RegisterMetadata(windows)
	lat := New(windows)
	m := metadata.New()
	compute := func(ts, nts time.Time) {
		now = func() time.Time { return nts }
		lat.Compute(ts)
	}
	updateReset := func(nts time.Time) {
		now = func() time.Time { return nts }
		lat.UpdateReset(m)
	}
	// Make sure it is still ok to call Compute and UpdateReset functions
	// even if no latency windows are set.
	compute(time.Unix(97, 0), time.Unix(98, 0)) // 1 second
	compute(time.Unix(96, 0), time.Unix(99, 0)) // 3 second
	updateReset(time.Unix(100, 0))
	compute(time.Unix(96, 0), time.Unix(101, 0)) // 5 second
	compute(time.Unix(94, 0), time.Unix(101, 0)) // 7 second
	updateReset(time.Unix(102, 0))
}

func TestLatency(t *testing.T) {
	defer func() {
		now = time.Now
	}()
	smWin, mdWin, lgWin := 2*time.Second, 4*time.Second, 8*time.Second
	windows := []time.Duration{smWin, mdWin, lgWin}
	RegisterMetadata(windows)
	meta := func(w time.Duration, typ StatType) string {
		return stat{window: w, typ: typ}.metaName()
	}
	var latStats []string
	for _, w := range windows {
		for _, typ := range []StatType{Avg, Max, Min} {
			latStats = append(latStats, meta(w, typ))
		}
	}
	lat := New(windows)
	m := metadata.New()
	checkLatency := func(desc string, lm map[string]time.Duration) {
		for name, want := range lm {
			val, err := m.GetInt(name)
			if err != nil {
				t.Fatalf("%s: metadata %q: got unexpected error %v", desc, name, err)
			}
			if val != want.Nanoseconds() {
				t.Fatalf("%s: metadata %q: got %d, want %d", desc, name, val, want.Nanoseconds())
			}
		}
		for _, name := range latStats {
			if _, ok := lm[name]; ok {
				continue
			}
			if _, err := m.GetInt(name); err == nil {
				t.Fatalf("%s: metadata %q: didn't get expected error", desc, name)
			}
		}
	}
	checkLatency("initial state", nil)

	compute := func(ts, nts time.Time) {
		now = func() time.Time { return nts }
		lat.Compute(ts)
	}
	updateReset := func(nts time.Time) {
		now = func() time.Time { return nts }
		lat.UpdateReset(m)
	}

	compute(time.Unix(97, 0), time.Unix(98, 0)) // 1 second
	compute(time.Unix(96, 0), time.Unix(99, 0)) // 3 second
	updateReset(time.Unix(100, 0))
	checkLatency("after interval 1", map[string]time.Duration{
		meta(smWin, Avg): 2 * time.Second,
		meta(smWin, Max): 3 * time.Second,
		meta(smWin, Min): 1 * time.Second})

	compute(time.Unix(96, 0), time.Unix(101, 0)) // 5 second
	compute(time.Unix(94, 0), time.Unix(101, 0)) // 7 second
	updateReset(time.Unix(102, 0))
	checkLatency("after interval 2", map[string]time.Duration{
		meta(smWin, Avg): 6 * time.Second,
		meta(smWin, Max): 7 * time.Second,
		meta(smWin, Min): 5 * time.Second,
		meta(mdWin, Avg): 4 * time.Second,
		meta(mdWin, Max): 7 * time.Second,
		meta(mdWin, Min): 1 * time.Second})

	compute(time.Unix(98, 1000), time.Unix(103, 1000))  // 5 second
	compute(time.Unix(100, 2000), time.Unix(103, 2000)) // 3 second
	updateReset(time.Unix(104, 0))
	checkLatency("after interval 3", map[string]time.Duration{
		meta(smWin, Avg): 4 * time.Second,
		meta(smWin, Max): 5 * time.Second,
		meta(smWin, Min): 3 * time.Second,
		meta(mdWin, Avg): 5 * time.Second,
		meta(mdWin, Max): 7 * time.Second,
		meta(mdWin, Min): 3 * time.Second})

	compute(time.Unix(101, 0), time.Unix(105, 0)) // 4 second
	updateReset(time.Unix(106, 0))
	checkLatency("after interval 4", map[string]time.Duration{
		meta(smWin, Avg): 4 * time.Second,
		meta(smWin, Max): 4 * time.Second,
		meta(smWin, Min): 4 * time.Second,
		meta(mdWin, Avg): 4 * time.Second,
		meta(mdWin, Max): 5 * time.Second,
		meta(mdWin, Min): 3 * time.Second,
		meta(lgWin, Avg): 4 * time.Second,
		meta(lgWin, Max): 7 * time.Second,
		meta(lgWin, Min): 1 * time.Second})

	compute(time.Unix(104, 1000), time.Unix(107, 1000)) // 3 second
	compute(time.Unix(105, 2000), time.Unix(107, 2000)) // 2 second
	compute(time.Unix(106, 3000), time.Unix(107, 3000)) // 1 second
	updateReset(time.Unix(108, 0))
	checkLatency("after interval 5", map[string]time.Duration{
		meta(smWin, Avg): 2 * time.Second,
		meta(smWin, Max): 3 * time.Second,
		meta(smWin, Min): 1 * time.Second,
		meta(mdWin, Avg): 2500 * time.Millisecond,
		meta(mdWin, Max): 4 * time.Second,
		meta(mdWin, Min): 1 * time.Second,
		meta(lgWin, Avg): 3750 * time.Millisecond,
		meta(lgWin, Max): 7 * time.Second,
		meta(lgWin, Min): 1 * time.Second})

	updateReset(time.Unix(110, 0))
	checkLatency("after interval 6", map[string]time.Duration{
		meta(smWin, Avg): 2 * time.Second,
		meta(smWin, Max): 3 * time.Second,
		meta(smWin, Min): 1 * time.Second,
		meta(mdWin, Avg): 2 * time.Second,
		meta(mdWin, Max): 3 * time.Second,
		meta(mdWin, Min): 1 * time.Second,
		meta(lgWin, Avg): 3 * time.Second,
		meta(lgWin, Max): 5 * time.Second,
		meta(lgWin, Min): 1 * time.Second})

	updateReset(time.Unix(112, 0))
	checkLatency("after interval 7", map[string]time.Duration{
		meta(smWin, Avg): 2 * time.Second,
		meta(smWin, Max): 3 * time.Second,
		meta(smWin, Min): 1 * time.Second,
		meta(mdWin, Avg): 2 * time.Second,
		meta(mdWin, Max): 3 * time.Second,
		meta(mdWin, Min): 1 * time.Second,
		meta(lgWin, Avg): 2500 * time.Millisecond,
		meta(lgWin, Max): 4 * time.Second,
		meta(lgWin, Min): 1 * time.Second})

	compute(time.Unix(110, 0), time.Unix(113, 0)) // 3 second
	compute(time.Unix(108, 0), time.Unix(113, 0)) // 5 second
	updateReset(time.Unix(114, 0))
	checkLatency("after interval 8", map[string]time.Duration{
		meta(smWin, Avg): 4 * time.Second,
		meta(smWin, Max): 5 * time.Second,
		meta(smWin, Min): 3 * time.Second,
		meta(mdWin, Avg): 4 * time.Second,
		meta(mdWin, Max): 5 * time.Second,
		meta(mdWin, Min): 3 * time.Second,
		meta(lgWin, Avg): 2800 * time.Millisecond,
		meta(lgWin, Max): 5 * time.Second,
		meta(lgWin, Min): 1 * time.Second})
}

func TestParseWindows(t *testing.T) {
	tests := []struct {
		desc    string
		windows []string
		period  time.Duration
		want    []time.Duration
		err     interface{}
	}{{
		desc:    "wrong time Duration",
		windows: []string{"abc"},
		period:  2 * time.Second,
		err:     true,
	}, {
		desc:    "window is not a multiple of update period",
		windows: []string{"2s", "5s"},
		period:  2 * time.Second,
		err:     "not a multiple of metadata update period",
	}, {
		desc:    "success",
		windows: []string{"2s", "30s", "5m", "3h"},
		period:  2 * time.Second,
		want:    []time.Duration{2 * time.Second, 30 * time.Second, 5 * time.Minute, 3 * time.Hour},
	}}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			got, err := ParseWindows(tt.windows, tt.period)
			if diff := errdiff.Check(err, tt.err); diff != "" {
				t.Fatalf("ParseWindows(%v) got error diff: %v", tt.windows, diff)
			}
			if err != nil {
				return
			}
			if diff := cmp.Diff(got, tt.want, nil); diff != "" {
				t.Errorf("ParseWindows(%v): got %v, want %v\ndiff: %s", tt.windows, got, tt.want, diff)
			}
		})
	}
}

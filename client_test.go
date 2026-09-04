package weheat

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// newTestClient serves fixture files and records the last request. No rate limiting.
func newTestClient(t *testing.T, fixture string) (*Client, *http.Request) {
	t.Helper()
	var last http.Request
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		last = *r
		if r.Header.Get("Authorization") != "Bearer test-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		body, err := os.ReadFile(filepath.Join("testdata", fixture+".json"))
		if err != nil {
			t.Fatal(err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)
	c := NewClient(staticToken("test-token"), WithBaseURL(srv.URL), WithRateLimiter(nil))
	return c, &last
}

func TestMe(t *testing.T) {
	c, last := newTestClient(t, "users_me")
	me, err := c.Me(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if last.URL.Path != "/users/me" || me.ID != "x" || me.CreatedOn.IsZero() {
		t.Fatalf("path=%s me=%+v", last.URL.Path, me)
	}
}

func TestHeatPumps(t *testing.T) {
	c, last := newTestClient(t, "heat_pumps")
	state := DeviceState(3)
	res, err := c.HeatPumps(context.Background(), &HeatPumpsQuery{Page: 1, PageSize: 1000, State: &state})
	if err != nil {
		t.Fatal(err)
	}
	q := last.URL.Query()
	if q.Get("page") != "1" || q.Get("pageSize") != "1000" || q.Get("State") != "3" {
		t.Fatalf("query=%v", q)
	}
	if len(res.Data) != 1 || res.Data[0].SerialNumber != "x" || *res.Metadata.TotalCount != 1 {
		t.Fatalf("res=%+v", res)
	}
}

func TestHeatPump(t *testing.T) {
	c, last := newTestClient(t, "heat_pump")
	hp, err := c.HeatPump(context.Background(), "abc")
	if err != nil || last.URL.Path != "/heat-pumps/abc" || hp.ID != "x" {
		t.Fatalf("err=%v path=%s hp=%+v", err, last.URL.Path, hp)
	}
}

func TestLatestLog(t *testing.T) {
	c, last := newTestClient(t, "logs_latest")
	log, err := c.LatestLog(context.Background(), "abc")
	if err != nil || last.URL.Path != "/heat-pumps/abc/logs/latest" || *log.IsOnline != true || *log.TWaterIn != 1.5 {
		t.Fatalf("err=%v path=%s log=%+v", err, last.URL.Path, log)
	}
}

func TestLogsQuery(t *testing.T) {
	c, last := newTestClient(t, "logs")
	start := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)
	logs, err := c.Logs(context.Background(), "abc", start, end, IntervalHour)
	if err != nil {
		t.Fatal(err)
	}
	q := last.URL.Query()
	if last.URL.Path != "/heat-pumps/abc/logs" || q.Get("interval") != "Hour" ||
		q.Get("startTime") != "2026-09-01T00:00:00Z" || q.Get("endTime") != "2026-09-02T00:00:00Z" {
		t.Fatalf("path=%s query=%v", last.URL.Path, q)
	}
	if len(logs) != 1 || *logs[0].TWaterInAverage != 1.5 {
		t.Fatalf("logs=%+v", logs)
	}
}

func TestRawLogs(t *testing.T) {
	c, last := newTestClient(t, "logs_raw")
	logs, err := c.RawLogs(context.Background(), "abc", time.Time{}, time.Time{})
	if err != nil || last.URL.Path != "/heat-pumps/abc/logs/raw" || last.URL.RawQuery != "" || len(logs) != 1 {
		t.Fatalf("err=%v url=%s n=%d", err, last.URL, len(logs))
	}
}

func TestEnergyLogs(t *testing.T) {
	c, last := newTestClient(t, "energy_logs")
	logs, err := c.EnergyLogs(context.Background(), "abc", time.Time{}, time.Time{}, IntervalDay)
	if err != nil || last.URL.Path != "/energy-logs/abc" || last.URL.Query().Get("interval") != "Day" || logs[0].TotalEinHeating != 1.5 {
		t.Fatalf("err=%v url=%s logs=%+v", err, last.URL, logs)
	}
}

func TestEnergyTotal(t *testing.T) {
	c, last := newTestClient(t, "energy_total")
	tot, err := c.EnergyTotal(context.Background(), "abc")
	if err != nil || last.URL.Path != "/energy-logs/abc/total" || *tot.TotalEinHeating != 1.5 {
		t.Fatalf("err=%v path=%s tot=%+v", err, last.URL.Path, tot)
	}
}

func TestAPIError(t *testing.T) {
	c, _ := newTestClient(t, "users_me")
	bad := NewClient(staticToken("wrong"), WithBaseURL(c.baseURL), WithRateLimiter(nil))
	_, err := bad.Me(context.Background())
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusUnauthorized {
		t.Fatalf("err=%v", err)
	}
}

func TestTimeUnmarshal(t *testing.T) {
	cases := map[string]time.Time{
		`"2026-09-03T12:00:00Z"`:          time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC),
		`"2026-09-03T12:00:00.123+02:00"`: time.Date(2026, 9, 3, 10, 0, 0, 123e6, time.UTC),
		`"2026-09-03T12:00:00.1234567"`:   time.Date(2026, 9, 3, 12, 0, 0, 123456700, time.UTC),
	}
	for in, want := range cases {
		var got Time
		if err := got.UnmarshalJSON([]byte(in)); err != nil || !got.Equal(want) {
			t.Errorf("%s: got %v err %v", in, got, err)
		}
	}
}

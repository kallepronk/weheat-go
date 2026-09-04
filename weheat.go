// Package weheat is a thin client for the WeHeat third-party API.
//
// One method per endpoint, DTOs mirror the Python `weheat` SDK, no behaviour layer.
package weheat

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"golang.org/x/oauth2"
)

// DefaultBaseURL is the production API root.
const DefaultBaseURL = "https://api.weheat.nl/third_party/api/v1"

// Interval is the aggregation bucket for log and energy queries.
type Interval string

// Intervals accepted by the API. Per-request windows differ per interval (see README).
const (
	IntervalMinute        Interval = "Minute"
	IntervalFiveMinute    Interval = "FiveMinute"
	IntervalFifteenMinute Interval = "FifteenMinute"
	IntervalHour          Interval = "Hour"
	IntervalDay           Interval = "Day"
	IntervalWeek          Interval = "Week"
	IntervalMonth         Interval = "Month"
	IntervalYear          Interval = "Year"
)

// Client talks to the WeHeat API.
type Client struct {
	baseURL string
	http    *http.Client
	limiter *RateLimiter
}

// Option configures a Client.
type Option func(*Client)

// WithBaseURL overrides the API root (tests, future environments).
func WithBaseURL(u string) Option { return func(c *Client) { c.baseURL = strings.TrimRight(u, "/") } }

// WithHTTPClient replaces the HTTP client. It must add the bearer token itself.
func WithHTTPClient(h *http.Client) Option { return func(c *Client) { c.http = h } }

// WithRateLimiter replaces the default limiter. Pass nil to disable rate limiting.
func WithRateLimiter(r *RateLimiter) Option { return func(c *Client) { c.limiter = r } }

// NewClient returns a client that authenticates with ts and rate-limits at DefaultRateLimit per hour.
func NewClient(ts oauth2.TokenSource, opts ...Option) *Client {
	c := &Client{
		baseURL: DefaultBaseURL,
		http:    oauth2.NewClient(context.Background(), ts),
		limiter: NewRateLimiter(DefaultRateLimit, DefaultRateWindow),
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

// APIError is returned for non-2xx responses.
type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("weheat: HTTP %d: %s", e.StatusCode, e.Body)
}

// Me returns the authenticated user. GET /users/me
func (c *Client) Me(ctx context.Context) (*ReadUserMeDto, error) {
	var out ReadUserMeDto
	return &out, c.get(ctx, "/users/me", nil, &out)
}

// HeatPumpsQuery filters the heat pump listing. Zero values are omitted.
type HeatPumpsQuery struct {
	Page           int
	PageSize       int
	Models         []HeatPumpModel
	OrganisationID string
	Search         string
	State          *DeviceState
}

// HeatPumps lists heat pumps visible to the user. GET /heat-pumps
func (c *Client) HeatPumps(ctx context.Context, q *HeatPumpsQuery) (*ReadAllHeatPumpDtoPagedResponse, error) {
	v := url.Values{}
	if q != nil {
		if q.Page > 0 {
			v.Set("page", strconv.Itoa(q.Page))
		}
		if q.PageSize > 0 {
			v.Set("pageSize", strconv.Itoa(q.PageSize))
		}
		for _, m := range q.Models {
			v.Add("Model", strconv.Itoa(int(m)))
		}
		if q.OrganisationID != "" {
			v.Set("OrganisationId", q.OrganisationID)
		}
		if q.Search != "" {
			v.Set("Search", q.Search)
		}
		if q.State != nil {
			v.Set("State", strconv.Itoa(int(*q.State)))
		}
	}
	var out ReadAllHeatPumpDtoPagedResponse
	return &out, c.get(ctx, "/heat-pumps", v, &out)
}

// HeatPump returns one heat pump. GET /heat-pumps/{id}
func (c *Client) HeatPump(ctx context.Context, id string) (*ReadHeatPumpDto, error) {
	var out ReadHeatPumpDto
	return &out, c.get(ctx, "/heat-pumps/"+url.PathEscape(id), nil, &out)
}

// LatestLog returns the most recent raw log plus online flag. GET /heat-pumps/{id}/logs/latest
func (c *Client) LatestLog(ctx context.Context, id string) (*RawHeatpumpLogAndIsOnlineDto, error) {
	var out RawHeatpumpLogAndIsOnlineDto
	return &out, c.get(ctx, "/heat-pumps/"+url.PathEscape(id)+"/logs/latest", nil, &out)
}

// Logs returns aggregated logs between start and end. GET /heat-pumps/{id}/logs
func (c *Client) Logs(ctx context.Context, id string, start, end time.Time, interval Interval) ([]HeatPumpLogViewDto, error) {
	v := timeRange(start, end)
	v.Set("interval", string(interval))
	var out []HeatPumpLogViewDto
	return out, c.get(ctx, "/heat-pumps/"+url.PathEscape(id)+"/logs", v, &out)
}

// RawLogs returns unaggregated logs between start and end (max 4 hours). GET /heat-pumps/{id}/logs/raw
func (c *Client) RawLogs(ctx context.Context, id string, start, end time.Time) ([]RawHeatPumpLogDto, error) {
	var out []RawHeatPumpLogDto
	return out, c.get(ctx, "/heat-pumps/"+url.PathEscape(id)+"/logs/raw", timeRange(start, end), &out)
}

// EnergyLogs returns energy buckets between start and end. GET /energy-logs/{id}
func (c *Client) EnergyLogs(ctx context.Context, id string, start, end time.Time, interval Interval) ([]EnergyViewDto, error) {
	v := timeRange(start, end)
	v.Set("interval", string(interval))
	var out []EnergyViewDto
	return out, c.get(ctx, "/energy-logs/"+url.PathEscape(id), v, &out)
}

// EnergyTotal returns lifetime energy totals. GET /energy-logs/{id}/total
func (c *Client) EnergyTotal(ctx context.Context, id string) (*TotalEnergyAggregate, error) {
	var out TotalEnergyAggregate
	return &out, c.get(ctx, "/energy-logs/"+url.PathEscape(id)+"/total", nil, &out)
}

func timeRange(start, end time.Time) url.Values {
	v := url.Values{}
	if !start.IsZero() {
		v.Set("startTime", start.UTC().Format(time.RFC3339))
	}
	if !end.IsZero() {
		v.Set("endTime", end.UTC().Format(time.RFC3339))
	}
	return v
}

func (c *Client) get(ctx context.Context, path string, query url.Values, out any) error {
	if c.limiter != nil {
		if err := c.limiter.Wait(ctx); err != nil {
			return err
		}
	}
	u := c.baseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return &APIError{StatusCode: resp.StatusCode, Body: strings.TrimSpace(string(body))}
	}
	return json.Unmarshal(body, out)
}

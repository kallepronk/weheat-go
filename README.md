# weheat-go

Go client for the [WeHeat](https://www.weheat.nl) third-party API.

Thin by design: one method per endpoint, DTOs mirror the Python
[`weheat`](https://pypi.org/project/weheat/) SDK 1:1 so its docs and the Home Assistant
integration map directly. No caching, no behaviour layer.

## Install

```bash
go get github.com/kallepronk/weheat-go
```

## Auth setup

WeHeat uses Keycloak (`auth.weheat.nl`, realm `Weheat`). No registration needed: log in with
your normal WeHeat account against the public `WeheatCommunityAPI` client
(`weheat.CommunityClientID`, no secret). The confidential `HomeAssistantAPI` client, whose ID
and secret WeHeat publishes in its [knowledge base](https://support.weheat.nl/s/article/Is-er-een-offici%C3%ABle-Home-Assistant-integratie),
works too.

```go
cfg := weheat.OAuthConfig(weheat.CommunityClientID, "")

// Option A: password grant with your WeHeat account.
ts, err := weheat.PasswordTokenSource(ctx, cfg, email, password)

// Option B: refresh token you stored earlier.
ts := weheat.RefreshTokenSource(ctx, cfg, refreshToken)

// Keycloak rotates refresh tokens. Persist the new one or you lock yourself out after a restart.
ts = weheat.NotifyTokenSource(ts, func(tok *oauth2.Token) { save(tok.RefreshToken) })

client := weheat.NewClient(ts)
```

## Quickstart

```go
me, err := client.Me(ctx)
pumps, err := client.HeatPumps(ctx, nil)
latest, err := client.LatestLog(ctx, pumps.Data[0].ID)
logs, err := client.Logs(ctx, id, start, end, weheat.IntervalHour)
```

Endpoints: `Me`, `HeatPumps`, `HeatPump`, `LatestLog`, `Logs`, `RawLogs`, `EnergyLogs`, `EnergyTotal`.

## Rate limit

WeHeat allows 34 requests per hour per user, shared across every client using that account.
The client blocks at 30/hour by default. Tune or disable:

```go
weheat.NewClient(ts, weheat.WithRateLimiter(weheat.NewRateLimiter(10, time.Hour)))
weheat.NewClient(ts, weheat.WithRateLimiter(nil)) // off
```

Per-request windows per interval: Minute 2 days, FiveMinute 1 week, FifteenMinute and Hour 1 month,
Day 1 year, Week 2 years, Month 5 years. `RawLogs` covers at most 4 hours.

## CLI

See [weheat-cli](https://github.com/kallepronk/weheat-cli).

## Development

```bash
mise install
go test ./...
WEHEAT_LIVE=1 go test -run TestLive ./...   # hits the real API, uses 2 requests
```

Fixtures in `testdata/` are synthetic until real responses are recorded.

## Credits

DTO shapes follow the Python `weheat` SDK by WeHeat. `joshp123/weheat-golang` served as a
reference for typed models but is not a dependency.

MIT.

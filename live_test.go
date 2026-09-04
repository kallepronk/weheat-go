package weheat

import (
	"context"
	"os"
	"testing"
)

// TestLive hits the real API. Needs WEHEAT_LIVE=1 and WEHEAT_REFRESH_TOKEN; WEHEAT_CLIENT_ID
// defaults to CommunityClientID, WEHEAT_CLIENT_SECRET may be empty. Uses 2 of the 34 hourly requests; stop any collector first.
func TestLive(t *testing.T) {
	if os.Getenv("WEHEAT_LIVE") != "1" {
		t.Skip("set WEHEAT_LIVE=1 to run against the real API")
	}
	ctx := context.Background()
	clientID := os.Getenv("WEHEAT_CLIENT_ID")
	if clientID == "" {
		clientID = CommunityClientID
	}
	cfg := OAuthConfig(clientID, os.Getenv("WEHEAT_CLIENT_SECRET"))
	c := NewClient(RefreshTokenSource(ctx, cfg, os.Getenv("WEHEAT_REFRESH_TOKEN")))

	me, err := c.Me(ctx)
	if err != nil {
		t.Fatal(err)
	}
	pumps, err := c.HeatPumps(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("user %s has %d heat pumps", me.ID, len(pumps.Data))
}

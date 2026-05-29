package spotify

import (
	"strings"
	"testing"
)

func TestAuthArgs(t *testing.T) {
	// Client credentials (tracks/albums): official API + creds + no-cache, no user-auth.
	joined := strings.Join(authArgs(AuthOptions{ClientID: "id123", ClientSecret: "secret456"}), " ")
	for _, want := range []string{"--use-official-api", "--client-id id123", "--client-secret secret456", "--no-cache"} {
		if !strings.Contains(joined, want) {
			t.Errorf("client-cred authArgs missing %q, got: %s", want, joined)
		}
	}
	if strings.Contains(joined, "--user-auth") {
		t.Errorf("client-cred authArgs must not request user-auth, got: %s", joined)
	}

	// User auth (playlists/liked): official API + creds + user-auth + cache-path, NO --no-cache.
	joined = strings.Join(authArgs(AuthOptions{ClientID: "id123", ClientSecret: "secret456", UserAuth: true, CachePath: "C:/cache/oauth"}), " ")
	for _, want := range []string{"--use-official-api", "--client-id id123", "--user-auth", "--cache-path C:/cache/oauth"} {
		if !strings.Contains(joined, want) {
			t.Errorf("user-auth authArgs missing %q, got: %s", want, joined)
		}
	}
	if strings.Contains(joined, "--no-cache") {
		t.Errorf("user-auth authArgs must NOT disable cache (token must persist), got: %s", joined)
	}

	// Missing creds -> only --no-cache, never official API or user-auth.
	joined = strings.Join(authArgs(AuthOptions{}), " ")
	if !strings.Contains(joined, "--no-cache") || strings.Contains(joined, "--use-official-api") || strings.Contains(joined, "--user-auth") {
		t.Errorf("no-creds authArgs should be just --no-cache, got: %s", joined)
	}
}

func TestRedactCmd(t *testing.T) {
	args := []string{"-m", "spotdl", "url", "https://x", "--client-id", "id123", "--client-secret", "secret456", "--no-cache"}
	out := redactCmd(args)
	if strings.Contains(out, "id123") || strings.Contains(out, "secret456") {
		t.Errorf("redactCmd leaked credentials: %s", out)
	}
	if !strings.Contains(out, "--client-id") || !strings.Contains(out, "***") {
		t.Errorf("redactCmd should keep flag names and mask values: %s", out)
	}
}

func TestParseResolvedURL(t *testing.T) {
	out := "You might be blocked by YouTube Music. If downloads fail, use a VPN.\n" +
		"Processing query: https://open.spotify.com/track/abc\n" +
		"https://music.youtube.com/watch?v=lYBUbBu4W08\n"
	got := parseResolvedURL(out)
	if got != "https://music.youtube.com/watch?v=lYBUbBu4W08" {
		t.Errorf("parseResolvedURL got %q", got)
	}
	// "Processing query:" line contains a spotify URL but is not a bare URL line.
	if parseResolvedURL("Processing query: https://open.spotify.com/track/abc\n") != "" {
		t.Errorf("should not return the Processing-query spotify URL")
	}
	if parseResolvedURL("no url here\n") != "" {
		t.Errorf("should return empty when no URL present")
	}
}

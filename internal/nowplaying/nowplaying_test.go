package nowplaying

import (
	"errors"
	"testing"
)

func TestParseTrack(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    Track
		wantErr bool
	}{
		{
			name: "empty output is nothing playing",
			raw:  "",
			want: Track{Present: false},
		},
		{
			name: "NO_SESSION sentinel is nothing playing",
			raw:  "NO_SESSION (API reachable, nothing playing right now)",
			want: Track{Present: false},
		},
		{
			name:    "ERROR prefix returns an error",
			raw:     "ERROR: the WinRT call blew up",
			wantErr: true,
		},
		{
			name: "valid spotify json maps to a playing track",
			raw:  `{"Title":"Bohemian Rhapsody","Artist":"Queen","App":"Spotify.exe","Status":"Playing"}`,
			want: Track{
				Title:   "Bohemian Rhapsody",
				Artist:  "Queen",
				Source:  "Spotify",
				Playing: true,
				Present: true,
			},
		},
		{
			name: "paused browser track keeps title but Playing is false",
			raw:  `{"Title":"Lo-fi beats to code to","Artist":"","App":"Chrome","Status":"Paused"}`,
			want: Track{
				Title:   "Lo-fi beats to code to",
				Artist:  "",
				Source:  "Chrome",
				Playing: false,
				Present: true,
			},
		},
		{
			name: "json with empty title is nothing playing",
			raw:  `{"Title":"","Artist":"","App":"Spotify.exe","Status":"Stopped"}`,
			want: Track{Present: false},
		},
		{
			name: "whitespace is trimmed from fields",
			raw:  `{"Title":"  Clair de Lune  ","Artist":"  Debussy ","App":"firefox","Status":"  Playing "}`,
			want: Track{
				Title:   "Clair de Lune",
				Artist:  "Debussy",
				Source:  "Firefox",
				Playing: true,
				Present: true,
			},
		},
		{
			name: "garbage output degrades quietly to nothing playing",
			raw:  "some unexpected stderr line",
			want: Track{Present: false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseTrack(tt.raw)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseTrack() err = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if got != tt.want {
				t.Errorf("parseTrack() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestFriendlySource(t *testing.T) {
	tests := []struct {
		appID string
		want  string
	}{
		{"Spotify.exe", "Spotify"},
		{"spotifyab.SpotifyMusic", "Spotify"},
		{"msedge", "Edge"},
		{"Microsoft.Edge", "Edge"},
		{"Chrome", "Chrome"},
		{"firefox.exe", "Firefox"},
		{"brave.exe", "Brave"},
		{"vlc.exe", "VLC"},
		{"", ""},
		{"SomeUnknownApp.exe", "SomeUnknownApp"},
		{"weird-player", "weird-player"},
	}

	for _, tt := range tests {
		t.Run(tt.appID, func(t *testing.T) {
			if got := friendlySource(tt.appID); got != tt.want {
				t.Errorf("friendlySource(%q) = %q, want %q", tt.appID, got, tt.want)
			}
		})
	}
}

func TestExecProviderCurrent(t *testing.T) {
	t.Run("success path parses helper output", func(t *testing.T) {
		p := &execProvider{runPS: func() (string, error) {
			return `{"Title":"Strobe","Artist":"deadmau5","App":"Spotify.exe","Status":"Playing"}`, nil
		}}
		got, err := p.Current()
		if err != nil {
			t.Fatalf("Current() unexpected error: %v", err)
		}
		want := Track{Title: "Strobe", Artist: "deadmau5", Source: "Spotify", Playing: true, Present: true}
		if got != want {
			t.Errorf("Current() = %+v, want %+v", got, want)
		}
	})

	t.Run("runPS error is wrapped as HelperError", func(t *testing.T) {
		p := &execProvider{runPS: func() (string, error) {
			return "", errors.New("powershell not found")
		}}
		_, err := p.Current()
		if err == nil {
			t.Fatal("Current() expected error, got nil")
		}
		var he *HelperError
		if !errors.As(err, &he) {
			t.Errorf("Current() error = %T, want *HelperError", err)
		}
	})
}

func TestNoopProvider(t *testing.T) {
	got, err := noopProvider{}.Current()
	if err != nil {
		t.Fatalf("noop Current() error: %v", err)
	}
	if got.Present {
		t.Errorf("noop Current() = %+v, want Present=false", got)
	}
}

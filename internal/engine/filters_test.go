package engine

import (
	"reflect"
	"testing"

	"github.com/swamp2k/copyarr/internal/config"
)

func TestPathAllowedIncludeOverridesExclude(t *testing.T) {
	if !pathAllowed("movies/keep.mkv", []string{"*.mkv"}, []string{"**"}) {
		t.Fatal("include must override a broad exclude")
	}
	if pathAllowed("movies/drop.txt", []string{"*.mkv"}, []string{"**"}) {
		t.Fatal("non-included file should still be excluded")
	}
}

func TestPathAllowedIncludesDoNotBecomeAllowlist(t *testing.T) {
	if !pathAllowed("notes/readme.txt", []string{"*.mkv"}, nil) {
		t.Fatal("include rules are exceptions, not an implicit exclude-all")
	}
}

func TestFilterPatterns(t *testing.T) {
	tests := []struct{ pattern, path string; want bool }{
		{"*.mkv", "a/b/movie.mkv", true},
		{"season/**/sample.*", "season/x/y/sample.mkv", true},
		{"temp/**", "temp/a/b.txt", true},
		{"temp/**", "other/temp/a.txt", false},
	}
	for _, tc := range tests {
		if got := filterMatch(tc.pattern, tc.path); got != tc.want {
			t.Errorf("filterMatch(%q,%q)=%v want %v", tc.pattern, tc.path, got, tc.want)
		}
	}
}

func TestRcloneFilterOrderKeepsIncludesFirst(t *testing.T) {
	r := config.Rule{Includes: []string{"*.mkv"}, Excludes: []string{"**"}}
	want := []string{"--filter", "+ *.mkv", "--filter", "+ */", "--filter", "- **"}
	if got := rcloneFilterArgs(r); !reflect.DeepEqual(got, want) {
		t.Fatalf("rcloneFilterArgs=%v want %v", got, want)
	}
}

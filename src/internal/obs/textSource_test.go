package obs

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPickTextKind(t *testing.T) {
	cases := []struct {
		name      string
		available []string
		want      string
	}{
		{
			name:      "freetype v2 is preferred when present",
			available: []string{"image_source", "text_ft2_source_v2", "text_gdiplus_v3", "ffmpeg_source"},
			want:      "text_ft2_source_v2",
		},
		{
			name:      "falls back to gdiplus on a build without freetype",
			available: []string{"ffmpeg_source", "text_gdiplus_v3", "image_source"},
			want:      "text_gdiplus_v3",
		},
		{
			name:      "older gdiplus",
			available: []string{"text_gdiplus_v2"},
			want:      "text_gdiplus_v2",
		},
		{
			name:      "unversioned freetype",
			available: []string{"text_ft2_source"},
			want:      "text_ft2_source",
		},
		{
			name:      "no text source at all",
			available: []string{"ffmpeg_source", "image_source", "scene"},
			want:      "",
		},
		{
			name:      "empty list",
			available: nil,
			want:      "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := pickTextKind(tc.available); got != tc.want {
				t.Errorf("pickTextKind(%v) = %q, want %q", tc.available, got, tc.want)
			}
		})
	}
}

func TestGetTextInputKind_FromObs(t *testing.T) {
	f := &fakeOBS{inputKinds: []string{"ffmpeg_source", "text_ft2_source_v2", "text_gdiplus_v3"}}
	srv := httptest.NewServer(http.HandlerFunc(f.serve))
	defer srv.Close()

	client := newTestClient(t, strings.TrimPrefix(srv.URL, "http://"))
	kind, err := client.getTextInputKind()
	if err != nil {
		t.Fatalf("getTextInputKind: %v", err)
	}
	if kind != "text_ft2_source_v2" {
		t.Errorf("expected text_ft2_source_v2, got %q", kind)
	}
	if client.textKind != kind {
		t.Errorf("expected the kind to be cached, got %q", client.textKind)
	}
}

func TestGetTextInputKind_GdiplusOnlyBuild(t *testing.T) {
	f := &fakeOBS{inputKinds: []string{"ffmpeg_source", "text_gdiplus_v3"}}
	srv := httptest.NewServer(http.HandlerFunc(f.serve))
	defer srv.Close()

	client := newTestClient(t, strings.TrimPrefix(srv.URL, "http://"))
	kind, err := client.getTextInputKind()
	if err != nil {
		t.Fatalf("getTextInputKind: %v", err)
	}
	if kind != "text_gdiplus_v3" {
		t.Errorf("expected text_gdiplus_v3, got %q", kind)
	}
}

// An OBS with no text source must fail loudly rather than create an input with
// an empty kind, which would leave the stream permanently invisible.
func TestGetTextInputKind_NoTextSourceIsAnError(t *testing.T) {
	f := &fakeOBS{inputKinds: []string{"ffmpeg_source", "image_source"}}
	srv := httptest.NewServer(http.HandlerFunc(f.serve))
	defer srv.Close()

	client := newTestClient(t, strings.TrimPrefix(srv.URL, "http://"))
	kind, err := client.getTextInputKind()
	if err == nil {
		t.Fatalf("expected an error, got kind %q", kind)
	}
	if kind != "" {
		t.Errorf("expected empty kind on error, got %q", kind)
	}
}

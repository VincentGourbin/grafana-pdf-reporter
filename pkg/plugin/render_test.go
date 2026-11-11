package plugin

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend/httpclient"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
)

func testPNG(t *testing.T) []byte {
	t.Helper()
	var out bytes.Buffer
	if err := png.Encode(&out, image.NewRGBA(image.Rect(0, 0, 4, 4))); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

func TestHTTPClientOutlastsARender(t *testing.T) {
	s := DefaultSettings()
	client, err := newHTTPClient(context.Background(), s, log.DefaultLogger)
	if err != nil {
		t.Fatal(err)
	}
	if want := s.RenderTimeout + 15*time.Second; client.Timeout != want {
		t.Fatalf("timeout %s, want %s", client.Timeout, want)
	}
}

func TestHTTPClientRejectsInvalidCA(t *testing.T) {
	s := DefaultSettings()
	s.TLSCACert = "not a certificate"
	if _, err := newHTTPClient(context.Background(), s, log.DefaultLogger); err == nil {
		t.Fatal("an unusable CA must not fall back to the system roots")
	}
}

func TestRenderHonorsResponseLimit(t *testing.T) {
	image := testPNG(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(image)
	}))
	defer srv.Close()
	s := DefaultSettings()
	s.GrafanaURL = srv.URL
	client, err := newHTTPClient(context.Background(), s, log.DefaultLogger)
	if err != nil {
		t.Fatal(err)
	}

	got, err := renderDashboardPNG(context.Background(), client, s, "d1", "now-1h", "now", "dark", "", 1920)
	if err != nil || !bytes.Equal(got, image) {
		t.Fatalf("unlimited render: %d bytes, %v", len(got), err)
	}

	limited := httpclient.WithResponseLimit(context.Background(), 8)
	_, err = renderDashboardPNG(limited, client, s, "d1", "now-1h", "now", "dark", "", 1920)
	if !errors.Is(err, httpclient.ErrResponseBodyTooLarge) {
		t.Fatalf("want the response limit error, got %v", err)
	}
}

func TestRenderErrorBodyIsBounded(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(strings.Repeat("x", 10*maxErrorBody)))
	}))
	defer srv.Close()
	s := DefaultSettings()
	s.GrafanaURL = srv.URL
	_, err := renderDashboardPNG(context.Background(), http.DefaultClient, s, "d1", "now-1h", "now", "dark", "", 1920)
	var status *grafanaStatusError
	if !errors.As(err, &status) || len(status.Body) != maxErrorBody {
		t.Fatalf("got %v", err)
	}
}

// pngHeader is a PNG reduced to its IHDR chunk: enough for DecodeConfig to read
// huge dimensions without the test allocating them.
func pngHeader(width, height uint32) []byte {
	data := make([]byte, 0, 17)
	data = append(data, "IHDR"...)
	data = binary.BigEndian.AppendUint32(data, width)
	data = binary.BigEndian.AppendUint32(data, height)
	data = append(data, 8, 6, 0, 0, 0) // 8-bit RGBA
	out := []byte("\x89PNG\r\n\x1a\n")
	out = binary.BigEndian.AppendUint32(out, 13)
	out = append(out, data...)
	return binary.BigEndian.AppendUint32(out, crc32.ChecksumIEEE(data))
}

func TestCropRefusesOversizedImages(t *testing.T) {
	_, _, _, err := cropToContent(pngHeader(30000, 30000), "dark")
	if err == nil || !strings.Contains(err.Error(), "too large") {
		t.Fatalf("got %v", err)
	}
}

func TestCropKeepsNormalImages(t *testing.T) {
	if _, w, h, err := cropToContent(testPNG(t), "dark"); err != nil || w != 4 || h != 4 {
		t.Fatalf("got %dx%d, %v", w, h, err)
	}
}

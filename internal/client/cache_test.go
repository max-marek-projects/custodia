package client

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
)

// --- helpers ---------------------------------------------------------------

// testLogger returns a logger that discards all output, so tests don't
// clutter stderr with WARN/DEBUG lines.
func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// setupConfigDir redirects os.UserConfigDir() to a fresh temporary directory
// and returns the full path to the "custodia" folder inside it.
//
// The environment variable used depends on the platform: XDG_CONFIG_HOME on
// Linux, HOME on macOS, AppData on Windows. t.Setenv restores the previous
// value at the end of the test.
func setupConfigDir(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	switch runtime.GOOS {
	case "windows":
		t.Setenv("AppData", tmp)
	case "darwin":
		t.Setenv("HOME", tmp)
	default:
		t.Setenv("XDG_CONFIG_HOME", tmp)
	}
	base, err := os.UserConfigDir()
	if err != nil {
		t.Fatalf("UserConfigDir: %v", err)
	}
	return filepath.Join(base, "custodia")
}

// mustNewCache fails the test if newSecretCache returns an error.
func mustNewCache(t *testing.T, folder string) *secretCache {
	t.Helper()
	c, err := newSecretCache(folder, testLogger())
	if err != nil {
		t.Fatalf("newSecretCache: %v", err)
	}
	return c
}

// sampleEntry returns a fully-populated cache entry for use in tests.
func sampleEntry() cachedSecret {
	return cachedSecret{
		UserID:   1,
		DataType: 1,
		Name:     "MySecret",
		Version:  0,
		Salt:     []byte("0123456789abcdef"),
		IV:       []byte("0123456789ab"),
		Data:     []byte("encrypted-payload"),
		Metadata: map[string]string{"env": "prod"},
	}
}

// writeRawCacheFile creates the custodia config directory and writes raw
// content to cache.json. Used to simulate corrupt or legacy files.
func writeRawCacheFile(t *testing.T, folder, content string) string {
	t.Helper()
	if err := os.MkdirAll(folder, 0700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	path := filepath.Join(folder, cacheFileName)
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return path
}

// --- cacheKey --------------------------------------------------------------

func Test_cacheKey_Format(t *testing.T) {
	tests := []struct {
		name    string
		userID  int64
		secret  string
		version uint64
		want    string
	}{
		{"basic", 1, "MySecret", 0, "1:MySecret:0"},
		{"non-zero version", 42, "foo", 7, "42:foo:7"},
		{"empty name", 0, "", 0, "0::0"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cacheKey(tt.userID, tt.secret, tt.version)
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

// --- newSecretCache / Load -------------------------------------------------

func Test_newSecretCache_MissingFile(t *testing.T) {
	setupConfigDir(t)
	c := mustNewCache(t, "custodia")
	if c == nil {
		t.Fatal("cache is nil")
	}
	if got := len(c.entries); got != 0 {
		t.Fatalf("expected empty cache, got %d entries", got)
	}
}

func Test_newSecretCache_LoadsExistingFile(t *testing.T) {
	folder := setupConfigDir(t)

	// Seed a valid cache file.
	seed := mustNewCache(t, "custodia")
	if err := seed.Put(sampleEntry()); err != nil {
		t.Fatalf("seed Put: %v", err)
	}
	// Sanity: file exists.
	if _, err := os.Stat(filepath.Join(folder, cacheFileName)); err != nil {
		t.Fatalf("cache file not written: %v", err)
	}

	// New instance must see the entry.
	fresh := mustNewCache(t, "custodia")
	got, ok := fresh.Get(1, "MySecret", 0)
	if !ok {
		t.Fatal("entry not loaded from disk")
	}
	if got.Name != "MySecret" {
		t.Fatalf("wrong entry loaded: %+v", got)
	}
}

func Test_newSecretCache_CorruptFile(t *testing.T) {
	folder := setupConfigDir(t)
	writeRawCacheFile(t, folder, "{this is not valid JSON")

	// A corrupt file must not abort startup: newSecretCache should return
	// an empty cache without an error.
	c := mustNewCache(t, "custodia")
	if got := len(c.entries); got != 0 {
		t.Fatalf("expected empty cache, got %d entries", got)
	}
}

func Test_Load_CalledTwiceIsIdempotent(t *testing.T) {
	setupConfigDir(t)
	c := mustNewCache(t, "custodia")
	if err := c.Put(sampleEntry()); err != nil {
		t.Fatalf("Put: %v", err)
	}
	if err := c.Load(); err != nil {
		t.Fatalf("second Load: %v", err)
	}
	// Entry survives.
	if _, ok := c.Get(1, "MySecret", 0); !ok {
		t.Fatal("entry lost after reload")
	}
}

// --- cachePath -------------------------------------------------------------

func Test_cachePath_UsesConfigDirAndFolder(t *testing.T) {
	folder := setupConfigDir(t)
	c := mustNewCache(t, "custodia")

	path, err := c.cachePath()
	if err != nil {
		t.Fatalf("cachePath: %v", err)
	}
	want := filepath.Join(folder, cacheFileName)
	if path != want {
		t.Fatalf("got %q, want %q", path, want)
	}
	// Directory must have been created.
	if _, err := os.Stat(folder); err != nil {
		t.Fatalf("cache directory not created: %v", err)
	}
}

func Test_cachePath_EmptyFolderFallsBackToDefault(t *testing.T) {
	setupConfigDir(t)
	c := mustNewCache(t, "") // empty folder
	path, err := c.cachePath()
	if err != nil {
		t.Fatalf("cachePath: %v", err)
	}
	if filepath.Base(filepath.Dir(path)) != "custodia" {
		t.Fatalf("expected fallback folder 'custodia', got %q", path)
	}
}

// --- Put / Get -------------------------------------------------------------

func Test_PutAndGet_RoundTrip(t *testing.T) {
	setupConfigDir(t)
	c := mustNewCache(t, "custodia")

	in := sampleEntry()
	if err := c.Put(in); err != nil {
		t.Fatalf("Put: %v", err)
	}

	out, ok := c.Get(in.UserID, in.Name, in.Version)
	if !ok {
		t.Fatal("Get: entry not found")
	}
	if out.UserID != in.UserID {
		t.Errorf("UserID: got %d, want %d", out.UserID, in.UserID)
	}
	if out.DataType != in.DataType {
		t.Errorf("DataType: got %d, want %d", out.DataType, in.DataType)
	}
	if out.Name != in.Name {
		t.Errorf("Name: got %q, want %q", out.Name, in.Name)
	}
	if out.Version != in.Version {
		t.Errorf("Version: got %d, want %d", out.Version, in.Version)
	}
	if string(out.Salt) != string(in.Salt) {
		t.Errorf("Salt mismatch")
	}
	if string(out.IV) != string(in.IV) {
		t.Errorf("IV mismatch")
	}
	if string(out.Data) != string(in.Data) {
		t.Errorf("Data mismatch")
	}
	if out.Metadata["env"] != "prod" {
		t.Errorf("Metadata mismatch: %+v", out.Metadata)
	}
	if out.CachedAt.IsZero() {
		t.Error("CachedAt was not populated by Put")
	}
}

func Test_Get_Miss(t *testing.T) {
	setupConfigDir(t)
	c := mustNewCache(t, "custodia")

	if _, ok := c.Get(1, "does-not-exist", 0); ok {
		t.Fatal("expected miss for unknown name")
	}
}

func Test_Get_IsolatedByKeyFields(t *testing.T) {
	setupConfigDir(t)
	c := mustNewCache(t, "custodia")
	if err := c.Put(sampleEntry()); err != nil {
		t.Fatalf("Put: %v", err)
	}

	cases := []struct {
		name    string
		userID  int64
		secret  string
		version uint64
	}{
		{"different user", 2, "MySecret", 0},
		{"different name", 1, "Other", 0},
		{"different version", 1, "MySecret", 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, ok := c.Get(tc.userID, tc.secret, tc.version); ok {
				t.Fatalf("unexpected hit for %d:%s:%d", tc.userID, tc.secret, tc.version)
			}
		})
	}
}

func Test_Put_OverwritesExistingEntry(t *testing.T) {
	setupConfigDir(t)
	c := mustNewCache(t, "custodia")

	first := sampleEntry()
	first.Data = []byte("first")
	if err := c.Put(first); err != nil {
		t.Fatalf("Put first: %v", err)
	}

	second := sampleEntry()
	second.Data = []byte("second")
	if err := c.Put(second); err != nil {
		t.Fatalf("Put second: %v", err)
	}

	got, ok := c.Get(1, "MySecret", 0)
	if !ok {
		t.Fatal("entry missing after overwrite")
	}
	if string(got.Data) != "second" {
		t.Fatalf("expected overwrite, got data=%q", got.Data)
	}
}

func Test_Put_PersistsAcrossInstances(t *testing.T) {
	setupConfigDir(t)

	c1 := mustNewCache(t, "custodia")
	if err := c1.Put(sampleEntry()); err != nil {
		t.Fatalf("Put: %v", err)
	}

	c2 := mustNewCache(t, "custodia")
	got, ok := c2.Get(1, "MySecret", 0)
	if !ok {
		t.Fatal("entry not visible in second instance")
	}
	if string(got.Data) != "encrypted-payload" {
		t.Fatalf("wrong payload: %q", got.Data)
	}
}

// --- file permissions ------------------------------------------------------

func Test_SaveFilePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file permissions are not enforced on Windows")
	}
	folder := setupConfigDir(t)
	c := mustNewCache(t, "custodia")
	if err := c.Put(sampleEntry()); err != nil {
		t.Fatalf("Put: %v", err)
	}

	info, err := os.Stat(filepath.Join(folder, cacheFileName))
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if mode := info.Mode().Perm(); mode != 0600 {
		t.Fatalf("expected 0600 permissions, got %o", mode)
	}
}

// --- concurrency -----------------------------------------------------------

func Test_ConcurrentPutAndGet(t *testing.T) {
	setupConfigDir(t)
	c := mustNewCache(t, "custodia")

	const (
		workers    = 8
		iterations = 20
	)

	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				entry := cachedSecret{
					UserID:  int64(worker),
					Name:    fmt.Sprintf("secret-%d-%d", worker, i),
					Version: uint64(i),
					Data:    []byte("payload"),
				}
				_ = c.Put(entry)
				_, _ = c.Get(entry.UserID, entry.Name, entry.Version)
			}
		}(w)
	}
	wg.Wait()

	// Spot-check one of the written entries.
	if _, ok := c.Get(0, "secret-0-0", 0); !ok {
		t.Fatal("expected at least one written entry to be retrievable")
	}
}

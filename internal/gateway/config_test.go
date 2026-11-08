package gateway

import (
	"os"
	"path/filepath"
	"testing"
)

const testMetadata = `{
  "name": "shellight-test",
  "gateway_endpoint": "https://127.0.0.1:17690",
  "is_remote": false,
  "gateway_port": 0,
  "auth_mode": "mtls"
}`

// fixture writes a CLI config dir like the one of the test gateway and
// returns the dir.
func fixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	gwDir := filepath.Join(dir, "gateways", "shellight-test")
	if err := os.MkdirAll(gwDir, 0o700); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "active_gateway"), "shellight-test")
	writeFile(t, filepath.Join(gwDir, "metadata.json"), testMetadata)
	return dir
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestLoad(t *testing.T) {
	dir := fixture(t)
	writeFile(t, filepath.Join(dir, "active_gateway"), "shellight-test\n") // newline is trimmed
	g, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	want := Gateway{
		Name:     "shellight-test",
		Endpoint: "https://127.0.0.1:17690",
		Addr:     "127.0.0.1:17690",
	}
	if g != want {
		t.Errorf("Load = %+v, want %+v", g, want)
	}
}

func TestLoadDefaultPort(t *testing.T) {
	dir := fixture(t)
	writeFile(t, filepath.Join(dir, "gateways", "shellight-test", "metadata.json"),
		`{"gateway_endpoint":"https://gateway.example","auth_mode":"mtls"}`)
	g, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if g.Addr != "gateway.example:443" {
		t.Errorf("Addr = %q", g.Addr)
	}
}

func TestLoadErrors(t *testing.T) {
	tests := []struct {
		name, active, metadata string
	}{
		{"active gateway unknown", "other", testMetadata},
		{"active gateway escapes", "../x", testMetadata},
		{"metadata not json", "shellight-test", "{"},
		{"endpoint not https", "shellight-test", `{"gateway_endpoint":"http://127.0.0.1:17690","auth_mode":"mtls"}`},
		{"auth mode not mtls", "shellight-test", `{"gateway_endpoint":"https://127.0.0.1:17690","auth_mode":"token"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := fixture(t)
			writeFile(t, filepath.Join(dir, "active_gateway"), tt.active)
			writeFile(t, filepath.Join(dir, "gateways", "shellight-test", "metadata.json"), tt.metadata)
			if _, err := Load(dir); err == nil {
				t.Fatal("Load: no error")
			}
		})
	}

	if _, err := Load(t.TempDir()); err == nil {
		t.Error("Load without active_gateway: no error")
	}
}

func TestDir(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/xdg")
	if d, _ := Dir(); d != "/xdg/openshell" {
		t.Errorf("Dir with XDG_CONFIG_HOME = %q", d)
	}

	t.Setenv("XDG_CONFIG_HOME", "relative") // not absolute: ignored per XDG spec
	t.Setenv("HOME", "/home/someone")
	if d, _ := Dir(); d != "/home/someone/.config/openshell" {
		t.Errorf("Dir with relative XDG_CONFIG_HOME = %q", d)
	}
}

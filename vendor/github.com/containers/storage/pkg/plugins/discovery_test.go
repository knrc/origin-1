package plugins

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func Setup(t *testing.T) (string, func()) {
	tmpdir, err := ioutil.TempDir("", "docker-test")
	if err != nil {
		t.Fatal(err)
	}
	backup := socketsPath
	socketsPath = tmpdir
	specsPaths = []string{tmpdir}

	return tmpdir, func() {
		socketsPath = backup
		os.RemoveAll(tmpdir)
	}
}

func TestFileSpecPlugin(t *testing.T) {
	tmpdir, unregister := Setup(t)
	defer unregister()

	cases := []struct {
		path string
		name string
		addr string
		fail bool
	}{
		// TODO Windows: Factor out the unix:// variants.
		{filepath.Join(tmpdir, "echo.spec"), "echo", "unix://var/lib/containers/storage/plugins/echo.sock", false},
		{filepath.Join(tmpdir, "echo", "echo.spec"), "echo", "unix://var/lib/containers/storage/plugins/echo.sock", false},
		{filepath.Join(tmpdir, "foo.spec"), "foo", "tcp://localhost:8080", false},
		{filepath.Join(tmpdir, "foo", "foo.spec"), "foo", "tcp://localhost:8080", false},
		{filepath.Join(tmpdir, "bar.spec"), "bar", "localhost:8080", true}, // unknown transport
	}

	for _, c := range cases {
		if err := os.MkdirAll(filepath.Dir(c.path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := ioutil.WriteFile(c.path, []byte(c.addr), 0644); err != nil {
			t.Fatal(err)
		}

		r := newLocalRegistry()
		p, err := r.Plugin(c.name)
		if c.fail && err == nil {
			continue
		}

		if err != nil {
			t.Fatal(err)
		}

		if p.name != c.name {
			t.Fatalf("Expected plugin `%s`, got %s\n", c.name, p.Name)
		}

		if p.Addr != c.addr {
			t.Fatalf("Expected plugin addr `%s`, got %s\n", c.addr, p.Addr)
		}

		if p.TLSConfig.InsecureSkipVerify != true {
			t.Fatalf("Expected TLS verification to be skipped")
		}
	}
}

func TestFileJSONSpecPlugin(t *testing.T) {
	tmpdir, unregister := Setup(t)
	defer unregister()

	p := filepath.Join(tmpdir, "example.json")
	spec := `{
  "Name": "plugin-example",
  "Addr": "https://example.com/containers-storage/plugin",
  "TLSConfig": {
    "CAFile": "/usr/shared/containers-storage/certs/example-ca.pem",
    "CertFile": "/usr/shared/containers-storage/certs/example-cert.pem",
    "KeyFile": "/usr/shared/containers-storage/certs/example-key.pem"
	}
}`

	if err := ioutil.WriteFile(p, []byte(spec), 0644); err != nil {
		t.Fatal(err)
	}

	r := newLocalRegistry()
	plugin, err := r.Plugin("example")
	if err != nil {
		t.Fatal(err)
	}

	if plugin.name != "example" {
		t.Fatalf("Expected plugin `plugin-example`, got %s\n", plugin.Name)
	}

	if plugin.Addr != "https://example.com/containers-storage/plugin" {
		t.Fatalf("Expected plugin addr `https://example.com/containers-storage/plugin`, got %s\n", plugin.Addr)
	}

	if plugin.TLSConfig.CAFile != "/usr/shared/containers-storage/certs/example-ca.pem" {
		t.Fatalf("Expected plugin CA `/usr/shared/containers-storage/certs/example-ca.pem`, got %s\n", plugin.TLSConfig.CAFile)
	}

	if plugin.TLSConfig.CertFile != "/usr/shared/containers-storage/certs/example-cert.pem" {
		t.Fatalf("Expected plugin Certificate `/usr/shared/containers-storage/certs/example-cert.pem`, got %s\n", plugin.TLSConfig.CertFile)
	}

	if plugin.TLSConfig.KeyFile != "/usr/shared/containers-storage/certs/example-key.pem" {
		t.Fatalf("Expected plugin Key `/usr/shared/containers-storage/certs/example-key.pem`, got %s\n", plugin.TLSConfig.KeyFile)
	}
}

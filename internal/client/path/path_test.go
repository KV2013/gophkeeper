package path

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestDataDir(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("тест актуален только для linux")
	}

	home := filepath.Join(os.TempDir(), "gophkeeper-home")

	tests := map[string]struct {
		xdg  string
		want string
	}{
		"XDG_DATA_HOME задан": {
			xdg:  filepath.Join(os.TempDir(), "xdg"),
			want: filepath.Join(os.TempDir(), "xdg", "gophkeeper"),
		},
		"XDG_DATA_HOME не задан": {
			xdg:  "",
			want: filepath.Join(home, ".local", "share", "gophkeeper"),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Setenv("XDG_DATA_HOME", tc.xdg)
			t.Setenv("HOME", home)

			dir, err := DataDir()
			if err != nil {
				t.Fatalf("DataDir: %v", err)
			}
			if dir != tc.want {
				t.Fatalf("DataDir: got %q, want %q", dir, tc.want)
			}
		})
	}
}

func TestDBPath(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("тест актуален только для linux")
	}

	tests := map[string]struct {
		xdg     string
		wantDir string
	}{
		"XDG_DATA_HOME задан": {
			xdg:     filepath.Join(os.TempDir(), "xdg"),
			wantDir: filepath.Join(os.TempDir(), "xdg", "gophkeeper"),
		},
		"XDG_DATA_HOME не задан": {
			xdg:     "",
			wantDir: filepath.Join(os.TempDir(), "gophkeeper-home", ".local", "share", "gophkeeper"),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Setenv("XDG_DATA_HOME", tc.xdg)
			t.Setenv("HOME", filepath.Join(os.TempDir(), "gophkeeper-home"))

			p, err := DBPath()
			if err != nil {
				t.Fatalf("DBPath: %v", err)
			}
			if want := filepath.Join(tc.wantDir, "gophkeeper.db"); p != want {
				t.Fatalf("DBPath: got %q, want %q", p, want)
			}
		})
	}
}

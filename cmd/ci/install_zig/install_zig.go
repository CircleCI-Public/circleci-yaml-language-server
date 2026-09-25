// Command install_zig installs the pinned Zig, the C compiler cgo uses for
// every release target, and prints the path to its binary.
//
// It is cached under bin/zig: a second run finds it there and downloads
// nothing. To bump it, take the new archive names and checksums from
// https://ziglang.org/download/index.json.
package main

import (
	"archive/tar"
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/ulikunitz/xz"
)

const version = "0.16.0"

// checksums are the SHA-256 of each host's archive, keyed by Zig's
// <arch>-<os> name for it.
var checksums = map[string]string{
	"x86_64-linux":   "70e49664a74374b48b51e6f3fdfbf437f6395d42509050588bd49abe52ba3d00",
	"aarch64-linux":  "ea4b09bfb22ec6f6c6ceac57ab63efb6b46e17ab08d21f69f3a48b38e1534f17",
	"x86_64-macos":   "0387557ed1877bc6a2e1802c8391953baddba76081876301c522f52977b52ba7",
	"aarch64-macos":  "b23d70deaa879b5c2d486ed3316f7eaa53e84acf6fc9cc747de152450d401489",
	"x86_64-windows": "68659eb5f1e4eb1437a722f1dd889c5a322c9954607f5edcf337bc3684a75a7e",
}

func main() {
	dir := flag.String("dir", filepath.Join("bin", "zig"), "directory to cache Zig in")
	flag.Parse()

	zig, err := install(*dir, runtime.GOOS, runtime.GOARCH)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "install_zig:", err)
		os.Exit(1)
	}

	// Forward slashes, so the path can go into CC on Windows unquoted.
	fmt.Println(filepath.ToSlash(zig))
}

func install(dir, goos, goarch string) (string, error) {
	host, err := zigHost(goos, goarch)
	if err != nil {
		return "", err
	}

	name := "zig-" + host + "-" + version
	exe := "zig"
	ext := ".tar.xz"
	if goos == "windows" {
		exe = "zig.exe"
		ext = ".zip"
	}

	dir, err = filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	zig := filepath.Join(dir, name, exe)
	if _, err := os.Stat(zig); err == nil {
		return zig, nil
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	archive, err := download(dir, "https://ziglang.org/download/"+version+"/"+name+ext, checksums[host])
	if err != nil {
		return "", err
	}
	defer func() { _ = os.Remove(archive) }()

	// Unpack beside the cache and rename it into place, so an interrupted run
	// doesn't leave a half-extracted Zig that the next run would trust.
	tmp, err := os.MkdirTemp(dir, ".extract-")
	if err != nil {
		return "", err
	}
	defer func() { _ = os.RemoveAll(tmp) }()

	if ext == ".zip" {
		err = unzip(archive, tmp)
	} else {
		err = untarXZ(archive, tmp)
	}
	if err != nil {
		return "", fmt.Errorf("unpacking %s: %w", filepath.Base(archive), err)
	}

	if err := os.RemoveAll(filepath.Join(dir, name)); err != nil {
		return "", err
	}
	if err := os.Rename(filepath.Join(tmp, name), filepath.Join(dir, name)); err != nil {
		return "", err
	}
	return zig, nil
}

// zigHost is Zig's name for the machine Go is running on.
func zigHost(goos, goarch string) (string, error) {
	arch := map[string]string{"amd64": "x86_64", "arm64": "aarch64"}[goarch]
	osName := map[string]string{"linux": "linux", "darwin": "macos", "windows": "windows"}[goos]
	host := arch + "-" + osName
	if _, ok := checksums[host]; !ok {
		return "", fmt.Errorf("no Zig %s pinned for %s/%s", version, goos, goarch)
	}
	return host, nil
}

// download fetches url into dir and checks it against sha256.
func download(dir, url, want string) (_ string, err error) {
	f, err := os.CreateTemp(dir, ".download-")
	if err != nil {
		return "", err
	}
	defer func() {
		_ = f.Close()
		if err != nil {
			_ = os.Remove(f.Name())
		}
	}()

	resp, err := http.Get(url) //nolint:gosec,noctx // a pinned URL, and a one-shot command
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("downloading %s: %s", url, resp.Status)
	}

	sum := sha256.New()
	if _, err := io.Copy(io.MultiWriter(f, sum), resp.Body); err != nil {
		return "", fmt.Errorf("downloading %s: %w", url, err)
	}
	if got := hex.EncodeToString(sum.Sum(nil)); got != want {
		return "", fmt.Errorf("%s has checksum %s, want %s", url, got, want)
	}
	return f.Name(), f.Close()
}

func untarXZ(archive, dest string) error {
	f, err := os.Open(archive)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	xr, err := xz.NewReader(f)
	if err != nil {
		return err
	}

	tr := tar.NewReader(xr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		path, err := within(dest, hdr.Name)
		if err != nil {
			return err
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			err = os.MkdirAll(path, 0o755)
		case tar.TypeReg:
			err = writeFile(path, hdr.FileInfo().Mode(), tr)
		case tar.TypeSymlink:
			rel, relErr := filepath.Rel(dest, filepath.Dir(path))
			if relErr != nil {
				return relErr
			}
			if _, err = within(dest, filepath.Join(rel, hdr.Linkname)); err == nil {
				err = os.Symlink(hdr.Linkname, path)
			}
		}
		if err != nil {
			return err
		}
	}
}

func unzip(archive, dest string) error {
	zr, err := zip.OpenReader(archive)
	if err != nil {
		return err
	}
	defer func() { _ = zr.Close() }()

	for _, zf := range zr.File {
		path, err := within(dest, zf.Name)
		if err != nil {
			return err
		}
		if zf.FileInfo().IsDir() {
			if err := os.MkdirAll(path, 0o755); err != nil {
				return err
			}
			continue
		}

		r, err := zf.Open()
		if err != nil {
			return err
		}
		err = writeFile(path, zf.Mode(), r)
		_ = r.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func writeFile(path string, mode os.FileMode, r io.Reader) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode.Perm()|0o600)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, r); err != nil { //nolint:gosec // a checksummed archive
		_ = f.Close()
		return err
	}
	return f.Close()
}

// within joins name onto dir, refusing a name that would land outside it.
// A rooted name is refused everywhere, though on Windows "/etc" without a
// drive letter isn't absolute and would only land inside dir.
func within(dir, name string) (string, error) {
	path := filepath.Join(dir, name)
	rooted := filepath.IsAbs(name) || strings.HasPrefix(name, "/") || strings.HasPrefix(name, `\`)
	if rooted || !strings.HasPrefix(path, filepath.Clean(dir)+string(filepath.Separator)) {
		return "", fmt.Errorf("archive entry %q escapes %s", name, dir)
	}
	return path, nil
}

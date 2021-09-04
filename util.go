package github

import (
	"errors"
	"io"
	"os"
	"path/filepath"

	"github.com/google/go-github/v35/github"
	"github.com/nhatthm/plugin-registry/plugin"
	"github.com/spf13/afero"
	"gopkg.in/yaml.v3"
)

// ErrArtifactNotFound indicates that the artifact is not found.
var ErrArtifactNotFound = errors.New("artifact not found")

func writeFile(fs afero.Fs, path string, r io.Reader) error {
	f, err := fs.OpenFile(path, os.O_CREATE|os.O_RDWR, os.FileMode(0o644))
	if err != nil {
		return err
	}

	defer f.Close() // nolint: errcheck

	if _, err := io.Copy(f, r); err != nil {
		return err
	}

	if r, ok := r.(io.ReadCloser); ok {
		_ = r.Close() // nolint: errcheck
	}

	return nil
}

func loadMetadata(r io.Reader) (*plugin.Plugin, error) {
	var p plugin.Plugin

	dec := yaml.NewDecoder(r)

	if err := dec.Decode(&p); err != nil {
		return nil, err
	}

	if r, ok := r.(io.ReadCloser); ok {
		_ = r.Close() // nolint: errcheck
	}

	return &p, nil
}

func writeMetadata(fs afero.Fs, path string, p *plugin.Plugin) error {
	return writeFile(fs, filepath.Join(path, plugin.MetadataFile), newYamlReader(p))
}

func findAsset(r *github.RepositoryRelease, file string) (*github.ReleaseAsset, error) {
	if len(r.Assets) == 0 {
		return nil, ErrArtifactNotFound
	}

	for _, a := range r.Assets {
		if *a.Name == file {
			return a, nil
		}
	}

	return nil, ErrArtifactNotFound
}

func chmod(fs afero.Fs, contentType *string, path string, fileMode os.FileMode) error {
	if contentType == nil {
		return nil
	}

	switch *contentType {
	case "application/octet-stream",
		"application/gzip":
		return fs.Chmod(path, fileMode)

	default:
		return nil
	}
}

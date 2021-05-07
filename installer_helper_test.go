package github_test

import (
	"fmt"
	"io"
	"path/filepath"

	goGitHub "github.com/google/go-github/v35/github"
	"github.com/nhatthm/plugin-registry/plugin"
	"github.com/spf13/afero"
	"github.com/spf13/afero/mem"
	"github.com/stretchr/testify/mock"
)

func newEmptyFile(name string) afero.File {
	return mem.NewFileHandle(mem.CreateFile(name))
}

func newShadowedFile(name string, source string) afero.File {
	return newFileFromSource(newEmptyFile(name), source)
}

func newFileFromSource(f afero.File, source string) afero.File {
	data, err := afero.ReadFile(afero.NewOsFs(), source)
	if err != nil {
		panic(err)
	}

	return newFileWithData(f, data)
}

func newFileWithData(f afero.File, data []byte) afero.File {
	_, _ = f.Write(data)           // nolint: errcheck
	_, _ = f.Seek(0, io.SeekStart) // nolint: errcheck

	return f
}

func newEmptyMetadataFile() afero.File {
	return newEmptyFile(plugin.MetadataFile)
}

// nolint: unparam
func newMetadataFile(source string) afero.File {
	return newShadowedFile(plugin.MetadataFile, source)
}

func newMetadataFileFromString(data string) afero.File {
	return newFileWithData(newEmptyMetadataFile(), []byte(data))
}

func newMetadataFileFromStringf(format string, args ...interface{}) afero.File {
	return newMetadataFileFromString(fmt.Sprintf(format, args...))
}

func newRelease(tagName string) *goGitHub.RepositoryRelease {
	return &goGitHub.RepositoryRelease{
		TagName: &tagName,
	}
}

func newReleaseWithArtifact(tagName, fileName string) *goGitHub.RepositoryRelease {
	r := newRelease(tagName)
	r.Assets = []*goGitHub.ReleaseAsset{
		{
			ID:   int64Ptr(42),
			Name: &fileName,
		},
	}

	return r
}

func newReleaseWithArtifactAndContentType(tagName, fileName, contentType string) *goGitHub.RepositoryRelease {
	r := newRelease(tagName)
	r.Assets = []*goGitHub.ReleaseAsset{
		{
			ID:          int64Ptr(42),
			Name:        &fileName,
			ContentType: &contentType,
		},
	}

	return r
}

// nolint: unparam
func newReleaseWithArtifactf(tagName string, format string, args ...interface{}) *goGitHub.RepositoryRelease {
	return newReleaseWithArtifact(tagName, fmt.Sprintf(format, args...))
}

func expectFileName(expect string) interface{} {
	return mock.MatchedBy(func(actual string) bool {
		return filepath.Base(filepath.Clean(actual)) == expect
	})
}

func expectFileNamef(format string, args ...interface{}) interface{} { // nolint: unparam
	return expectFileName(fmt.Sprintf(format, args...))
}

func int64Ptr(i int64) *int64 {
	return &i
}

func stringPtr(i string) *string {
	return &i
}

func stringPtrf(format string, args ...interface{}) *string {
	return stringPtr(fmt.Sprintf(format, args...))
}

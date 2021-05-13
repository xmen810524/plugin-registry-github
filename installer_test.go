package github_test

import (
	"context"
	"errors"
	"net/http"
	"os"
	"runtime"
	"strings"
	"testing"

	goGitHub "github.com/google/go-github/v35/github"
	"github.com/nhatthm/aferomock"
	github "github.com/nhatthm/plugin-registry-github"
	"github.com/nhatthm/plugin-registry-github/mock/service"
	"github.com/nhatthm/plugin-registry/installer"
	"github.com/nhatthm/plugin-registry/plugin"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestInstaller_Install(t *testing.T) {
	t.Parallel()

	installer.Register("fail", func(_ context.Context, source string) bool {
		return strings.HasSuffix(source, ".fail")
	}, func(afero.Fs) installer.Installer {
		return installer.CallbackInstaller(func(context.Context, string, string) (*plugin.Plugin, error) {
			return nil, errors.New("could not install")
		})
	})

	installer.Register("success", func(_ context.Context, source string) bool {
		return strings.HasSuffix(source, ".success")
	}, func(afero.Fs) installer.Installer {
		return installer.CallbackInstaller(func(context.Context, string, string) (*plugin.Plugin, error) {
			return &plugin.Plugin{Name: "my-plugin"}, nil
		})
	})

	testCases := []struct {
		scenario       string
		mockFs         aferomock.FsMocker
		mockService    service.RepositoryServiceMocker
		source         string
		expectedResult *plugin.Plugin
		expectedError  string
	}{
		{
			scenario:      "could not parse url",
			source:        "/tmp/plugin.zip",
			expectedError: "could not parse url: not a github url",
		},
		{
			scenario: "could not get latest release",
			source:   "github.com/owner/my-plugin",
			mockService: service.MockRepositoryService(func(s *service.RepositoryService) {
				s.On("GetLatestRelease", mock.Anything, "owner", "my-plugin").
					Return(nil, nil, errors.New("get error"))
			}),
			expectedError: "could not find latest release: get error",
		},
		{
			scenario: "latest release tag is empty",
			source:   "github.com/owner/my-plugin",
			mockService: service.MockRepositoryService(func(s *service.RepositoryService) {
				s.On("GetLatestRelease", mock.Anything, "owner", "my-plugin").
					Return(&goGitHub.RepositoryRelease{}, nil, nil)
			}),
			expectedError: "latest release has no tag name",
		},
		{
			scenario: "could not get release by tag",
			source:   "github.com/owner/my-plugin@v1.4.2",
			mockService: service.MockRepositoryService(func(s *service.RepositoryService) {
				s.On("GetReleaseByTag", mock.Anything, "owner", "my-plugin", "v1.4.2").
					Return(nil, nil, errors.New("get error"))
			}),
			expectedError: "could not get release: get error",
		},
		{
			scenario: "could not download metadata",
			source:   "github.com/owner/my-plugin@v1.4.2",
			mockService: service.MockRepositoryService(func(s *service.RepositoryService) {
				s.On("GetReleaseByTag", mock.Anything, "owner", "my-plugin", "v1.4.2").
					Return(newRelease("v1.4.2"), nil, nil)

				s.On("DownloadContents", mock.Anything, "owner", "my-plugin", ".plugin.registry.yaml",
					&goGitHub.RepositoryContentGetOptions{Ref: "v1.4.2"}).
					Return(nil, nil, errors.New("download error"))
			}),
			expectedError: "could not get plugin metadata: download error",
		},
		{
			scenario: "could not load metadata",
			source:   "github.com/owner/my-plugin@v1.4.2",
			mockService: service.MockRepositoryService(func(s *service.RepositoryService) {
				s.On("GetReleaseByTag", mock.Anything, "owner", "my-plugin", "v1.4.2").
					Return(newRelease("v1.4.2"), nil, nil)

				s.On("DownloadContents", mock.Anything, "owner", "my-plugin", ".plugin.registry.yaml",
					&goGitHub.RepositoryContentGetOptions{Ref: "v1.4.2"}).
					Return(newEmptyMetadataFile(), nil, nil)
			}),
			expectedError: "could not load plugin metadata: EOF",
		},
		{
			scenario: "could not find artifact (no artifact)",
			source:   "github.com/owner/my-plugin@v1.4.2",
			mockService: service.MockRepositoryService(func(s *service.RepositoryService) {
				s.On("GetReleaseByTag", mock.Anything, "owner", "my-plugin", "v1.4.2").
					Return(newRelease("v1.4.2"), nil, nil)

				s.On("DownloadContents", mock.Anything, "owner", "my-plugin", ".plugin.registry.yaml",
					&goGitHub.RepositoryContentGetOptions{Ref: "v1.4.2"}).
					Return(newMetadataFile("resources/fixtures/.plugin.registry.yaml"), nil, nil)
			}),
			expectedError: "could not find artifact: artifact not found",
		},
		{
			scenario: "could not find artifact",
			source:   "github.com/owner/my-plugin@v1.4.2",
			mockService: service.MockRepositoryService(func(s *service.RepositoryService) {
				s.On("GetReleaseByTag", mock.Anything, "owner", "my-plugin", "v1.4.2").
					Return(newReleaseWithArtifact("v1.4.2", "unknown.zip"), nil, nil)

				s.On("DownloadContents", mock.Anything, "owner", "my-plugin", ".plugin.registry.yaml",
					&goGitHub.RepositoryContentGetOptions{Ref: "v1.4.2"}).
					Return(newMetadataFile("resources/fixtures/.plugin.registry.yaml"), nil, nil)
			}),
			expectedError: "could not find artifact: artifact not found",
		},
		{
			scenario: "could not download artifact",
			source:   "github.com/owner/my-plugin@v1.4.2",
			mockService: service.MockRepositoryService(func(s *service.RepositoryService) {
				artifact := newReleaseWithArtifactf("v1.4.2", "my-plugin-1.4.2-%s-%s.tar.gz", runtime.GOOS, runtime.GOARCH)

				s.On("GetReleaseByTag", mock.Anything, "owner", "my-plugin", "v1.4.2").
					Return(artifact, nil, nil)

				s.On("DownloadContents", mock.Anything, "owner", "my-plugin", ".plugin.registry.yaml",
					&goGitHub.RepositoryContentGetOptions{Ref: "v1.4.2"}).
					Return(newMetadataFile("resources/fixtures/.plugin.registry.yaml"), nil, nil)

				s.On("DownloadReleaseAsset", mock.Anything, "owner", "my-plugin", int64(42), http.DefaultClient).
					Return(nil, "", errors.New("download error"))
			}),
			expectedError: "could not download artifact: download error",
		},
		{
			scenario: "could not create temp dir",
			source:   "github.com/owner/my-plugin@v1.4.2",
			mockFs: aferomock.MockFs(func(fs *aferomock.Fs) {
				fs.On("Mkdir", mock.Anything, os.FileMode(0700)).
					Return(errors.New("mkdir error"))
			}),
			mockService: service.MockRepositoryService(func(s *service.RepositoryService) {
				artifact := newReleaseWithArtifactf("v1.4.2", "my-plugin-1.4.2-%s-%s.tar.gz", runtime.GOOS, runtime.GOARCH)

				s.On("GetReleaseByTag", mock.Anything, "owner", "my-plugin", "v1.4.2").
					Return(artifact, nil, nil)

				s.On("DownloadContents", mock.Anything, "owner", "my-plugin", ".plugin.registry.yaml",
					&goGitHub.RepositoryContentGetOptions{Ref: "v1.4.2"}).
					Return(newMetadataFile("resources/fixtures/.plugin.registry.yaml"), nil, nil)

				s.On("DownloadReleaseAsset", mock.Anything, "owner", "my-plugin", int64(42), http.DefaultClient).
					Return(newEmptyFile("my-plugin.tar.gz"), "", nil)
			}),
			expectedError: "could not create temp dir: mkdir error",
		},
		{
			scenario: "could not write artifact (open error)",
			source:   "github.com/owner/my-plugin@v1.4.2",
			mockFs: aferomock.MockFs(func(fs *aferomock.Fs) {
				fs.On("Mkdir", mock.Anything, os.FileMode(0700)).
					Return(nil)

				fs.On("OpenFile",
					expectFileNamef("my-plugin-1.4.2-%s-%s.tar.gz", runtime.GOOS, runtime.GOARCH),
					os.O_CREATE|os.O_RDWR, os.FileMode(0644)).
					Return(nil, errors.New("open error"))

				fs.On("RemoveAll", mock.Anything).Return(nil)
			}),
			mockService: service.MockRepositoryService(func(s *service.RepositoryService) {
				artifact := newReleaseWithArtifactf("v1.4.2", "my-plugin-1.4.2-%s-%s.tar.gz", runtime.GOOS, runtime.GOARCH)

				s.On("GetReleaseByTag", mock.Anything, "owner", "my-plugin", "v1.4.2").
					Return(artifact, nil, nil)

				s.On("DownloadContents", mock.Anything, "owner", "my-plugin", ".plugin.registry.yaml",
					&goGitHub.RepositoryContentGetOptions{Ref: "v1.4.2"}).
					Return(newMetadataFile("resources/fixtures/.plugin.registry.yaml"), nil, nil)

				s.On("DownloadReleaseAsset", mock.Anything, "owner", "my-plugin", int64(42), http.DefaultClient).
					Return(newEmptyFile("my-plugin.tar.gz"), "", nil)
			}),
			expectedError: "could not write artifact: open error",
		},
		{
			scenario: "could not write artifact (copy error)",
			source:   "github.com/owner/my-plugin@v1.4.2",
			mockFs: aferomock.MockFs(func(fs *aferomock.Fs) {
				fs.On("Mkdir", mock.Anything, os.FileMode(0700)).
					Return(nil)

				fs.On("OpenFile",
					expectFileNamef("my-plugin-1.4.2-%s-%s.tar.gz", runtime.GOOS, runtime.GOARCH),
					os.O_CREATE|os.O_RDWR, os.FileMode(0644)).
					Return(newEmptyFile("my-plugin.tar.gz"), nil)

				fs.On("RemoveAll", mock.Anything).Return(nil)
			}),
			mockService: service.MockRepositoryService(func(s *service.RepositoryService) {
				artifact := newReleaseWithArtifactf("v1.4.2", "my-plugin-1.4.2-%s-%s.tar.gz", runtime.GOOS, runtime.GOARCH)

				s.On("GetReleaseByTag", mock.Anything, "owner", "my-plugin", "v1.4.2").
					Return(artifact, nil, nil)

				s.On("DownloadContents", mock.Anything, "owner", "my-plugin", ".plugin.registry.yaml",
					&goGitHub.RepositoryContentGetOptions{Ref: "v1.4.2"}).
					Return(newMetadataFile("resources/fixtures/.plugin.registry.yaml"), nil, nil)

				asset := newEmptyFile("my-plugin.tar.gz")
				_ = asset.Close() // nolint: errcheck

				s.On("DownloadReleaseAsset", mock.Anything, "owner", "my-plugin", int64(42), http.DefaultClient).
					Return(asset, "", nil)
			}),
			expectedError: "could not write artifact: File is closed",
		},
		{
			scenario: "could not chmod artifact",
			source:   "github.com/owner/my-plugin@v1.4.2",
			mockFs: aferomock.MockFs(func(fs *aferomock.Fs) {
				fs.On("Mkdir", mock.Anything, os.FileMode(0700)).
					Return(nil)

				fs.On("OpenFile",
					expectFileNamef("my-plugin-1.4.2-%s-%s.tar.gz", runtime.GOOS, runtime.GOARCH),
					os.O_CREATE|os.O_RDWR, os.FileMode(0644)).
					Return(newEmptyFile("my-plugin.tar.gz"), nil)

				fs.On("Chmod",
					expectFileNamef("my-plugin-1.4.2-%s-%s.tar.gz", runtime.GOOS, runtime.GOARCH),
					os.FileMode(0755)).
					Return(errors.New("chmod error"))

				fs.On("RemoveAll", mock.Anything).Return(nil)
			}),
			mockService: service.MockRepositoryService(func(s *service.RepositoryService) {
				artifact := newReleaseWithArtifactf("v1.4.2", "my-plugin-1.4.2-%s-%s.tar.gz", runtime.GOOS, runtime.GOARCH)
				artifact.Assets[0].ContentType = stringPtr("application/octet-stream")

				s.On("GetReleaseByTag", mock.Anything, "owner", "my-plugin", "v1.4.2").
					Return(artifact, nil, nil)

				s.On("DownloadContents", mock.Anything, "owner", "my-plugin", ".plugin.registry.yaml",
					&goGitHub.RepositoryContentGetOptions{Ref: "v1.4.2"}).
					Return(newMetadataFile("resources/fixtures/.plugin.registry.yaml"), nil, nil)

				s.On("DownloadReleaseAsset", mock.Anything, "owner", "my-plugin", int64(42), http.DefaultClient).
					Return(newEmptyFile("my-plugin.tar.gz"), "", nil)
			}),
			expectedError: "could not chmod artifact: chmod error",
		},
		{
			scenario: "could not write plugin metadata",
			source:   "github.com/owner/my-plugin@v1.4.2",
			mockFs: aferomock.MockFs(func(fs *aferomock.Fs) {
				fs.On("Mkdir", mock.Anything, os.FileMode(0700)).
					Return(nil)

				fs.On("OpenFile",
					expectFileNamef("my-plugin-1.4.2-%s-%s.tar.gz", runtime.GOOS, runtime.GOARCH),
					os.O_CREATE|os.O_RDWR, os.FileMode(0644)).
					Return(newEmptyFile("my-plugin.tar.gz"), nil)

				metadataFile := newEmptyFile(".plugin.registry.yaml")
				_ = metadataFile.Close() // nolint: errcheck

				fs.On("OpenFile",
					expectFileName(".plugin.registry.yaml"), os.O_CREATE|os.O_RDWR, os.FileMode(0644)).
					Return(metadataFile, nil)

				fs.On("RemoveAll", mock.Anything).Return(nil)
			}),
			mockService: service.MockRepositoryService(func(s *service.RepositoryService) {
				artifact := newReleaseWithArtifactf("v1.4.2", "my-plugin-1.4.2-%s-%s.tar.gz", runtime.GOOS, runtime.GOARCH)

				s.On("GetReleaseByTag", mock.Anything, "owner", "my-plugin", "v1.4.2").
					Return(artifact, nil, nil)

				s.On("DownloadContents", mock.Anything, "owner", "my-plugin", ".plugin.registry.yaml",
					&goGitHub.RepositoryContentGetOptions{Ref: "v1.4.2"}).
					Return(newMetadataFile("resources/fixtures/.plugin.registry.yaml"), nil, nil)

				s.On("DownloadReleaseAsset", mock.Anything, "owner", "my-plugin", int64(42), http.DefaultClient).
					Return(newEmptyFile("my-plugin.tar.gz"), "", nil)
			}),
			expectedError: "could not write plugin metadata: File is closed",
		},
		{
			scenario: "could not find installer because no installer supports 7z",
			source:   "github.com/owner/my-plugin@v1.4.2",
			mockFs: aferomock.MockFs(func(fs *aferomock.Fs) {
				fs.On("Mkdir", mock.Anything, os.FileMode(0700)).
					Return(nil)

				fs.On("OpenFile",
					expectFileName("my-plugin.7z"), os.O_CREATE|os.O_RDWR, os.FileMode(0644)).
					Return(newEmptyFile("my-plugin.7z"), nil)

				fs.On("OpenFile",
					expectFileName(".plugin.registry.yaml"), os.O_CREATE|os.O_RDWR, os.FileMode(0644)).
					Return(newEmptyFile(".plugin.registry.yaml"), nil)

				fs.On("Stat", expectFileName("my-plugin.7z")).
					Return(aferomock.NewFileInfo(func(i *aferomock.FileInfo) {
						i.On("IsDir").Return(false)
						i.On("Name").Return("my-plugin.7z")
					}), nil)

				fs.On("RemoveAll", mock.Anything).Return(nil)
			}),
			mockService: service.MockRepositoryService(func(s *service.RepositoryService) {
				artifact := newReleaseWithArtifact("v1.4.2", "my-plugin.7z")

				s.On("GetReleaseByTag", mock.Anything, "owner", "my-plugin", "v1.4.2").
					Return(artifact, nil, nil)

				metadataFile := newMetadataFileFromStringf(`
name: my-plugin
artifacts:
    %s/%s:
        file: my-plugin.7z
`, runtime.GOOS, runtime.GOARCH)

				s.On("DownloadContents", mock.Anything, "owner", "my-plugin", ".plugin.registry.yaml",
					&goGitHub.RepositoryContentGetOptions{Ref: "v1.4.2"}).
					Return(metadataFile, nil, nil)

				s.On("DownloadReleaseAsset", mock.Anything, "owner", "my-plugin", int64(42), http.DefaultClient).
					Return(newEmptyFile("my-plugin.7z"), "", nil)
			}),
			expectedError: "no supported installer",
		},
		{
			scenario: "could not install",
			source:   "github.com/owner/my-plugin@v1.4.2",
			mockFs: aferomock.MockFs(func(fs *aferomock.Fs) {
				fs.On("Mkdir", mock.Anything, os.FileMode(0700)).
					Return(nil)

				fs.On("OpenFile",
					expectFileName("my-plugin.fail"), os.O_CREATE|os.O_RDWR, os.FileMode(0644)).
					Return(newEmptyFile("my-plugin.fail"), nil)

				fs.On("OpenFile",
					expectFileName(".plugin.registry.yaml"), os.O_CREATE|os.O_RDWR, os.FileMode(0644)).
					Return(newEmptyFile(".plugin.registry.yaml"), nil)

				fs.On("Stat", expectFileName("my-plugin.fail")).Maybe().
					Return(aferomock.NewFileInfo(func(i *aferomock.FileInfo) {
						i.On("IsDir").Return(false)
						i.On("Name").Return("my-plugin.fail")
					}), nil)

				fs.On("RemoveAll", mock.Anything).Return(nil)
			}),
			mockService: service.MockRepositoryService(func(s *service.RepositoryService) {
				artifact := newReleaseWithArtifact("v1.4.2", "my-plugin.fail")

				s.On("GetReleaseByTag", mock.Anything, "owner", "my-plugin", "v1.4.2").
					Return(artifact, nil, nil)

				metadataFile := newMetadataFileFromStringf(`
name: my-plugin
artifacts:
    %s/%s:
        file: my-plugin.fail
`, runtime.GOOS, runtime.GOARCH)

				s.On("DownloadContents", mock.Anything, "owner", "my-plugin", ".plugin.registry.yaml",
					&goGitHub.RepositoryContentGetOptions{Ref: "v1.4.2"}).
					Return(metadataFile, nil, nil)

				s.On("DownloadReleaseAsset", mock.Anything, "owner", "my-plugin", int64(42), http.DefaultClient).
					Return(newEmptyFile("my-plugin.fail"), "", nil)
			}),
			expectedError: "could not install",
		},
		{
			scenario: "success",
			source:   "github.com/owner/my-plugin@v1.4.2",
			mockFs: aferomock.MockFs(func(fs *aferomock.Fs) {
				fs.On("Mkdir", mock.Anything, os.FileMode(0700)).
					Return(nil)

				fs.On("OpenFile",
					expectFileName("my-plugin.success"), os.O_CREATE|os.O_RDWR, os.FileMode(0644)).
					Return(newEmptyFile("my-plugin.success"), nil)

				fs.On("OpenFile",
					expectFileName(".plugin.registry.yaml"), os.O_CREATE|os.O_RDWR, os.FileMode(0644)).
					Return(newEmptyFile(".plugin.registry.yaml"), nil)

				fs.On("Stat", expectFileName("my-plugin.success")).Maybe().
					Return(aferomock.NewFileInfo(func(i *aferomock.FileInfo) {
						i.On("IsDir").Return(false)
						i.On("Name").Return("my-plugin.success")
					}), nil)

				fs.On("RemoveAll", mock.Anything).Return(nil)
			}),
			mockService: service.MockRepositoryService(func(s *service.RepositoryService) {
				artifact := newReleaseWithArtifact("v1.4.2", "my-plugin.success")

				s.On("GetReleaseByTag", mock.Anything, "owner", "my-plugin", "v1.4.2").
					Return(artifact, nil, nil)

				metadataFile := newMetadataFileFromStringf(`
name: my-plugin
artifacts:
    %s/%s:
        file: my-plugin.success
`, runtime.GOOS, runtime.GOARCH)

				s.On("DownloadContents", mock.Anything, "owner", "my-plugin", ".plugin.registry.yaml",
					&goGitHub.RepositoryContentGetOptions{Ref: "v1.4.2"}).
					Return(metadataFile, nil, nil)

				s.On("DownloadReleaseAsset", mock.Anything, "owner", "my-plugin", int64(42), http.DefaultClient).
					Return(newEmptyFile("my-plugin.success"), "", nil)
			}),
			expectedResult: &plugin.Plugin{Name: "my-plugin"},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.scenario, func(t *testing.T) {
			t.Parallel()

			if tc.mockFs == nil {
				tc.mockFs = aferomock.NoMockFs
			}

			if tc.mockService == nil {
				tc.mockService = service.NoMockRepositoryService
			}

			c := github.NewInstaller(
				github.WithFs(tc.mockFs(t)),
				github.WithService(tc.mockService(t)),
			)
			result, err := c.Install(context.Background(), "/tmp", tc.source)

			assert.Equal(t, tc.expectedResult, result)

			if tc.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tc.expectedError)
			}
		})
	}
}

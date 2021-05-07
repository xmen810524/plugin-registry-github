package github_test

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	goGitHub "github.com/google/go-github/v35/github"
	"github.com/nhatthm/httpmock"
	github "github.com/nhatthm/plugin-registry-github"
	fsCtx "github.com/nhatthm/plugin-registry/context"
	"github.com/nhatthm/plugin-registry/installer"
	"github.com/nhatthm/plugin-registry/plugin"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func newRepositoryService(baseURL string) github.RepositoryService {
	u, err := url.Parse(strings.TrimSuffix(baseURL, "/") + "/")
	if err != nil {
		panic(err)
	}

	c := goGitHub.NewClient(nil)
	c.BaseURL = u

	return c.Repositories
}

func mockServerRelease(version, file, contentType string) func(s *httpmock.Server) {
	fileName := filepath.Base(file)

	return func(s *httpmock.Server) {
		s.ExpectGet(fmt.Sprintf("/repos/owner/my-plugin/releases/tags/%s", version)).
			ReturnJSON(newReleaseWithArtifactAndContentType(version, fileName, contentType))

		mockServerAssets(file)(s)
	}
}

func mockServerLatestRelease(file string, contentType string) func(s *httpmock.Server) {
	fileName := filepath.Base(file)

	return func(s *httpmock.Server) {
		s.ExpectGet("/repos/owner/my-plugin/releases/latest").
			ReturnJSON(newReleaseWithArtifactAndContentType("v1.4.2", fileName, contentType))

		mockServerAssets(file)(s)
	}
}

func mockServerAssets(file string) func(s *httpmock.Server) {
	fileName := filepath.Base(file)

	return func(s *httpmock.Server) {
		s.ExpectGet("/repos/owner/my-plugin/contents/?ref=v1.4.2").
			ReturnJSON([]*goGitHub.RepositoryContent{
				{
					Name:        stringPtr(".plugin.registry.yaml"),
					DownloadURL: stringPtrf("%s/owner/my-plugin/v1.4.2/.plugin.registry.yaml", s.URL()),
				},
			})

		s.ExpectGet("/owner/my-plugin/v1.4.2/.plugin.registry.yaml").
			WithHandler(func(*http.Request) ([]byte, error) {
				return yaml.Marshal(plugin.Plugin{
					Name:   "my-plugin",
					Hidden: true,
					Artifacts: plugin.Artifacts{
						plugin.RuntimeArtifactIdentifier(): {
							File: fileName,
						},
					},
				})
			})

		s.ExpectGet("/repos/owner/my-plugin/releases/assets/42").
			WithHeader("Accept", "application/octet-stream").
			ReturnFile(file)
	}
}

func TestIntegrationInstaller_Install(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		scenario       string
		mockServer     httpmock.Mocker
		version        string
		expectedResult *plugin.Plugin
	}{
		{
			scenario:   "zip",
			mockServer: httpmock.New(mockServerLatestRelease("resources/fixtures/zip/my-plugin.zip", "application/zip")),
			expectedResult: &plugin.Plugin{
				Name:    "my-plugin",
				URL:     "https://github.com/owner/my-plugin",
				Version: "1.4.2",
				Enabled: false,
				Hidden:  true,
				Artifacts: plugin.Artifacts{
					plugin.RuntimeArtifactIdentifier(): {
						File: "my-plugin.zip",
					},
				},
			},
		},
		{
			scenario:   "zip no parent",
			mockServer: httpmock.New(mockServerLatestRelease("resources/fixtures/zip/my-plugin-no-parent.zip", "application/zip")),
			expectedResult: &plugin.Plugin{
				Name:    "my-plugin",
				URL:     "https://github.com/owner/my-plugin",
				Version: "1.4.2",
				Enabled: false,
				Hidden:  true,
				Artifacts: plugin.Artifacts{
					plugin.RuntimeArtifactIdentifier(): {
						File: "my-plugin-no-parent.zip",
					},
				},
			},
		},
		{
			scenario:   "tarball",
			mockServer: httpmock.New(mockServerLatestRelease("resources/fixtures/gzip/my-plugin.tar.gz", "application/gzip")),
			expectedResult: &plugin.Plugin{
				Name:    "my-plugin",
				URL:     "https://github.com/owner/my-plugin",
				Version: "1.4.2",
				Enabled: false,
				Hidden:  true,
				Artifacts: plugin.Artifacts{
					plugin.RuntimeArtifactIdentifier(): {
						File: "my-plugin.tar.gz",
					},
				},
			},
		},
		{
			scenario:   "gunzip",
			mockServer: httpmock.New(mockServerLatestRelease("resources/fixtures/gzip/my-plugin-no-parent.gz", "application/gzip")),
			expectedResult: &plugin.Plugin{
				Name:    "my-plugin",
				URL:     "https://github.com/owner/my-plugin",
				Version: "1.4.2",
				Enabled: false,
				Hidden:  true,
				Artifacts: plugin.Artifacts{
					plugin.RuntimeArtifactIdentifier(): {
						File: "my-plugin-no-parent.gz",
					},
				},
			},
		},
		{
			scenario:   "binary",
			mockServer: httpmock.New(mockServerLatestRelease("resources/fixtures/binary/my-plugin", "application/octet-stream")),
			expectedResult: &plugin.Plugin{
				Name:    "my-plugin",
				URL:     "https://github.com/owner/my-plugin",
				Version: "1.4.2",
				Enabled: false,
				Hidden:  true,
				Artifacts: plugin.Artifacts{
					plugin.RuntimeArtifactIdentifier(): {
						File: "my-plugin",
					},
				},
			},
		},
		{
			scenario:   "empty version",
			mockServer: httpmock.New(mockServerLatestRelease("resources/fixtures/binary/my-plugin", "application/octet-stream")),
			version:    "@",
			expectedResult: &plugin.Plugin{
				Name:    "my-plugin",
				URL:     "https://github.com/owner/my-plugin",
				Version: "1.4.2",
				Enabled: false,
				Hidden:  true,
				Artifacts: plugin.Artifacts{
					plugin.RuntimeArtifactIdentifier(): {
						File: "my-plugin",
					},
				},
			},
		},
		{
			scenario:   "latest version",
			mockServer: httpmock.New(mockServerLatestRelease("resources/fixtures/binary/my-plugin", "application/octet-stream")),
			version:    "@latest",
			expectedResult: &plugin.Plugin{
				Name:    "my-plugin",
				URL:     "https://github.com/owner/my-plugin",
				Version: "1.4.2",
				Enabled: false,
				Hidden:  true,
				Artifacts: plugin.Artifacts{
					plugin.RuntimeArtifactIdentifier(): {
						File: "my-plugin",
					},
				},
			},
		},
		{
			scenario:   "specified version",
			mockServer: httpmock.New(mockServerRelease("v1.4.2", "resources/fixtures/binary/my-plugin", "application/octet-stream")),
			version:    "@v1.4.2",
			expectedResult: &plugin.Plugin{
				Name:    "my-plugin",
				URL:     "https://github.com/owner/my-plugin",
				Version: "1.4.2",
				Enabled: false,
				Hidden:  true,
				Artifacts: plugin.Artifacts{
					plugin.RuntimeArtifactIdentifier(): {
						File: "my-plugin",
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.scenario, func(t *testing.T) {
			t.Parallel()

			svr := tc.mockServer(t)

			source := fmt.Sprintf("github.com/owner/my-plugin%s", tc.version)
			dest := t.TempDir()

			osFs := afero.NewOsFs()
			ctx := fsCtx.WithFs(context.Background(), osFs)

			i, err := installer.Find(ctx, source)
			require.NoError(t, err)
			assert.IsType(t, &github.Installer{}, i)

			i.(*github.Installer).WithService(newRepositoryService(svr.URL()))

			result, err := i.Install(context.Background(), dest, source)
			require.NoError(t, err)

			assert.Equal(t, tc.expectedResult, result)

			file := filepath.Join(dest, result.Name, result.Name)

			info, err := osFs.Stat(file)
			require.NoError(t, err)
			assert.Equal(t, os.FileMode(0755), info.Mode())

			data, err := afero.ReadFile(osFs, file)
			require.NoError(t, err)

			expected := "#!/bin/bash\n"

			assert.Equal(t, expected, string(data))
		})
	}
}

package service_test

import (
	"context"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/google/go-github/v35/github"
	"github.com/nhatthm/plugin-registry-github/mock/service"
	"github.com/spf13/afero"
	"github.com/spf13/afero/mem"
	"github.com/stretchr/testify/assert"
)

func newMetadataFile() afero.File {
	return mem.NewFileHandle(mem.CreateFile(".plugin.registry.yaml"))
}

func TestDownloadContents(t *testing.T) {
	t.Parallel()

	opt := &github.RepositoryContentGetOptions{Ref: "v1.0.1"}
	file := newMetadataFile()

	testCases := []struct {
		scenario         string
		mockService      service.RepositoryServiceMocker
		expectedReader   io.ReadCloser
		expectedResponse *github.Response
		expectedError    string
	}{
		{
			scenario: "reader is io.ReadCloser",
			mockService: service.MockRepositoryService(func(s *service.RepositoryService) {
				s.On("DownloadContents", context.Background(), "owner", "repo", ".plugin.registry.yaml", opt).
					Return(file, nil, nil)
			}),
			expectedReader: file,
		},
		{
			scenario: "reader is io.Reader",
			mockService: service.MockRepositoryService(func(s *service.RepositoryService) {
				s.On("DownloadContents", context.Background(), "owner", "repo", ".plugin.registry.yaml", opt).
					Return(strings.NewReader(`hello`), nil, nil)
			}),
			expectedReader: ioutil.NopCloser(strings.NewReader(`hello`)),
		},
		{
			scenario: "response is not nil",
			mockService: service.MockRepositoryService(func(s *service.RepositoryService) {
				s.On("DownloadContents", context.Background(), "owner", "repo", ".plugin.registry.yaml", opt).
					Return(nil, &github.Response{FirstPage: 1}, nil)
			}),
			expectedResponse: &github.Response{FirstPage: 1},
		},
		{
			scenario: "error is not nil",
			mockService: service.MockRepositoryService(func(s *service.RepositoryService) {
				s.On("DownloadContents", context.Background(), "owner", "repo", ".plugin.registry.yaml", opt).
					Return(nil, nil, errors.New("error"))
			}),
			expectedError: "error",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.scenario, func(t *testing.T) {
			t.Parallel()

			s := tc.mockService(t)

			rdr, resp, err := s.DownloadContents(context.Background(), "owner", "repo", ".plugin.registry.yaml", opt)

			assert.Equal(t, tc.expectedReader, rdr)
			assert.Equal(t, tc.expectedResponse, resp)

			if tc.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tc.expectedError)
			}
		})
	}
}

func TestGetLatestRelease(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		scenario         string
		mockService      service.RepositoryServiceMocker
		expectedRelease  *github.RepositoryRelease
		expectedResponse *github.Response
		expectedError    string
	}{
		{
			scenario: "release is not nil",
			mockService: service.MockRepositoryService(func(s *service.RepositoryService) {
				s.On("GetLatestRelease", context.Background(), "owner", "repo").
					Return(&github.RepositoryRelease{}, nil, nil)
			}),
			expectedRelease: &github.RepositoryRelease{},
		},
		{
			scenario: "response is not nil",
			mockService: service.MockRepositoryService(func(s *service.RepositoryService) {
				s.On("GetLatestRelease", context.Background(), "owner", "repo").
					Return(nil, &github.Response{FirstPage: 1}, nil)
			}),
			expectedResponse: &github.Response{FirstPage: 1},
		},
		{
			scenario: "error is not nil",
			mockService: service.MockRepositoryService(func(s *service.RepositoryService) {
				s.On("GetLatestRelease", context.Background(), "owner", "repo").
					Return(nil, nil, errors.New("error"))
			}),
			expectedError: "error",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.scenario, func(t *testing.T) {
			t.Parallel()

			s := tc.mockService(t)

			release, resp, err := s.GetLatestRelease(context.Background(), "owner", "repo")

			assert.Equal(t, tc.expectedRelease, release)
			assert.Equal(t, tc.expectedResponse, resp)

			if tc.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tc.expectedError)
			}
		})
	}
}

func TestGetReleaseByTag(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		scenario         string
		mockService      service.RepositoryServiceMocker
		expectedRelease  *github.RepositoryRelease
		expectedResponse *github.Response
		expectedError    string
	}{
		{
			scenario: "release is not nil",
			mockService: service.MockRepositoryService(func(s *service.RepositoryService) {
				s.On("GetReleaseByTag", context.Background(), "owner", "repo", "v1.0.1").
					Return(&github.RepositoryRelease{}, nil, nil)
			}),
			expectedRelease: &github.RepositoryRelease{},
		},
		{
			scenario: "response is not nil",
			mockService: service.MockRepositoryService(func(s *service.RepositoryService) {
				s.On("GetReleaseByTag", context.Background(), "owner", "repo", "v1.0.1").
					Return(nil, &github.Response{FirstPage: 1}, nil)
			}),
			expectedResponse: &github.Response{FirstPage: 1},
		},
		{
			scenario: "error is not nil",
			mockService: service.MockRepositoryService(func(s *service.RepositoryService) {
				s.On("GetReleaseByTag", context.Background(), "owner", "repo", "v1.0.1").
					Return(nil, nil, errors.New("error"))
			}),
			expectedError: "error",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.scenario, func(t *testing.T) {
			t.Parallel()

			s := tc.mockService(t)

			release, resp, err := s.GetReleaseByTag(context.Background(), "owner", "repo", "v1.0.1")

			assert.Equal(t, tc.expectedRelease, release)
			assert.Equal(t, tc.expectedResponse, resp)

			if tc.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tc.expectedError)
			}
		})
	}
}

func TestDownloadReleaseAsset(t *testing.T) {
	t.Parallel()

	file := newMetadataFile()

	testCases := []struct {
		scenario       string
		mockService    service.RepositoryServiceMocker
		expectedReader io.ReadCloser
		expectedURL    string
		expectedError  string
	}{
		{
			scenario: "reader is io.ReadCloser",
			mockService: service.MockRepositoryService(func(s *service.RepositoryService) {
				s.On("DownloadReleaseAsset", context.Background(), "owner", "repo", int64(42), http.DefaultClient).
					Return(file, "", nil)
			}),
			expectedReader: file,
		},
		{
			scenario: "reader is io.Reader",
			mockService: service.MockRepositoryService(func(s *service.RepositoryService) {
				s.On("DownloadReleaseAsset", context.Background(), "owner", "repo", int64(42), http.DefaultClient).
					Return(strings.NewReader(`hello`), "", nil)
			}),
			expectedReader: ioutil.NopCloser(strings.NewReader(`hello`)),
		},
		{
			scenario: "redirect url is not empty",
			mockService: service.MockRepositoryService(func(s *service.RepositoryService) {
				s.On("DownloadReleaseAsset", context.Background(), "owner", "repo", int64(42), http.DefaultClient).
					Return(nil, "https://github.com/owner/new-repo", nil)
			}),
			expectedURL: "https://github.com/owner/new-repo",
		},
		{
			scenario: "error is not nil",
			mockService: service.MockRepositoryService(func(s *service.RepositoryService) {
				s.On("DownloadReleaseAsset", context.Background(), "owner", "repo", int64(42), http.DefaultClient).
					Return(nil, "", errors.New("error"))
			}),
			expectedError: "error",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.scenario, func(t *testing.T) {
			t.Parallel()

			s := tc.mockService(t)

			rdr, resp, err := s.DownloadReleaseAsset(context.Background(), "owner", "repo", 42, http.DefaultClient)

			assert.Equal(t, tc.expectedReader, rdr)
			assert.Equal(t, tc.expectedURL, resp)

			if tc.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tc.expectedError)
			}
		})
	}
}

package service

import (
	"context"
	"io"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/google/go-github/v35/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// RepositoryServiceMocker is RepositoryService mocker.
type RepositoryServiceMocker func(tb testing.TB) *RepositoryService

// NoMockRepositoryService is no mock RepositoryService.
var NoMockRepositoryService = MockRepositoryService()

// RepositoryService is a github.RepositoryService.
type RepositoryService struct {
	mock.Mock
}

// DownloadContents satisfies github.RepositoryService.
func (r *RepositoryService) DownloadContents(
	ctx context.Context,
	owner, repo, filepath string,
	opts *github.RepositoryContentGetOptions,
) (rdr io.ReadCloser, resp *github.Response, err error) {
	ret := r.Called(ctx, owner, repo, filepath, opts)

	ret1 := ret.Get(0)
	ret2 := ret.Get(1)
	err = ret.Error(2)

	switch ret1 := ret1.(type) {
	case nil:
		rdr = nil

	case io.ReadCloser:
		rdr = ret1

	default:
		rdr = ioutil.NopCloser(ret1.(io.Reader))
	}

	if ret2 != nil {
		resp = ret2.(*github.Response) // nolint: errcheck
	}

	return
}

// GetLatestRelease satisfies github.RepositoryService.
func (r *RepositoryService) GetLatestRelease(
	ctx context.Context,
	owner, repo string,
) (release *github.RepositoryRelease, resp *github.Response, err error) {
	ret := r.Called(ctx, owner, repo)

	ret1 := ret.Get(0)
	ret2 := ret.Get(1)
	err = ret.Error(2)

	if ret1 != nil {
		release = ret1.(*github.RepositoryRelease) // nolint: errcheck
	}

	if ret2 != nil {
		resp = ret2.(*github.Response) // nolint: errcheck
	}

	return
}

// GetReleaseByTag satisfies github.RepositoryService.
func (r *RepositoryService) GetReleaseByTag(
	ctx context.Context,
	owner, repo, tag string,
) (release *github.RepositoryRelease, resp *github.Response, err error) {
	ret := r.Called(ctx, owner, repo, tag)

	ret1 := ret.Get(0)
	ret2 := ret.Get(1)
	err = ret.Error(2)

	if ret1 != nil {
		release = ret1.(*github.RepositoryRelease) // nolint: errcheck
	}

	if ret2 != nil {
		resp = ret2.(*github.Response) // nolint: errcheck
	}

	return
}

// DownloadReleaseAsset satisfies github.RepositoryService.
func (r *RepositoryService) DownloadReleaseAsset(
	ctx context.Context,
	owner, repo string,
	id int64,
	followRedirectsClient *http.Client,
) (rdr io.ReadCloser, redirectURL string, err error) {
	ret := r.Called(ctx, owner, repo, id, followRedirectsClient)

	ret1 := ret.Get(0)
	redirectURL = ret.String(1)
	err = ret.Error(2)

	switch ret1 := ret1.(type) {
	case nil:
		rdr = nil

	case io.ReadCloser:
		rdr = ret1

	default:
		rdr = ioutil.NopCloser(ret1.(io.Reader))
	}

	return
}

// mockRepositoryService mocks github.RepositoryService interface.
func mockRepositoryService(mocks ...func(s *RepositoryService)) *RepositoryService {
	s := &RepositoryService{}

	for _, m := range mocks {
		m(s)
	}

	return s
}

// MockRepositoryService creates RepositoryService mock with cleanup to ensure all the expectations are met.
//goland:noinspection GoNameStartsWithPackageName
func MockRepositoryService(mocks ...func(s *RepositoryService)) RepositoryServiceMocker {
	return func(tb testing.TB) *RepositoryService {
		tb.Helper()

		s := mockRepositoryService(mocks...)

		tb.Cleanup(func() {
			assert.True(tb, s.Mock.AssertExpectations(tb))
		})

		return s
	}
}

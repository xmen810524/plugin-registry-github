package github

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

//goland:noinspection HttpUrlsUsage
func TestIsPlugin(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		scenario string
		url      string
		expected bool
	}{
		{
			scenario: "has http",
			url:      "http://github.com/owner/my-plugin",
			expected: true,
		},
		{
			scenario: "has https",
			url:      "https://github.com/owner/my-plugin",
			expected: true,
		},
		{
			scenario: "has no protocol",
			url:      "github.com/owner/my-plugin",
			expected: true,
		},
		{
			scenario: "not a github plugin",
			url:      "gitlab.com/owner/my-plugin",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.scenario, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.expected, isPlugin(tc.url))
		})
	}
}

//goland:noinspection HttpUrlsUsage
func TestParseURL(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		scenario           string
		url                string
		expectedOwner      string
		expectedRepository string
		expectedVersion    string
		expectedError      string
	}{
		// Failure cases.
		{
			scenario:      "not github without scheme",
			url:           "gitlab.com/owner/my-plugin",
			expectedError: "not a github url",
		},
		{
			scenario:      "not github with scheme",
			url:           "http://gitlab.com/owner/my-plugin",
			expectedError: "not a github url",
		},
		{
			scenario:      "missing hostname",
			url:           "/owner/my-plugin",
			expectedError: "not a github url",
		},
		{
			scenario:      "missing owner",
			url:           "github.com//my-plugin",
			expectedError: "missing github owner",
		},
		{
			scenario:      "missing repository",
			url:           "github.com/owner",
			expectedError: "missing github repository",
		},
		{
			scenario:      "missing repository with slash",
			url:           "github.com/owner/",
			expectedError: "missing github repository",
		},
		{
			scenario:      "missing repository with double slash",
			url:           "github.com/owner//",
			expectedError: "missing github repository",
		},
		// Success cases.
		{
			scenario:           "without scheme and without version",
			url:                "github.com/owner/my-plugin",
			expectedOwner:      "owner",
			expectedRepository: "my-plugin",
		},
		{
			scenario:           "without scheme and with version",
			url:                "github.com/owner/my-plugin@v1.1.0",
			expectedOwner:      "owner",
			expectedRepository: "my-plugin",
			expectedVersion:    "v1.1.0",
		},
		{
			scenario:           "with http:// and without version",
			url:                "http://github.com/owner/my-plugin",
			expectedOwner:      "owner",
			expectedRepository: "my-plugin",
		},
		{
			scenario:           "with https:// and without version",
			url:                "https://github.com/owner/my-plugin",
			expectedOwner:      "owner",
			expectedRepository: "my-plugin",
		},
		{
			scenario:           "with scheme and version",
			url:                "https://github.com/owner/my-plugin@v1.1.0",
			expectedOwner:      "owner",
			expectedRepository: "my-plugin",
			expectedVersion:    "v1.1.0",
		},
		{
			scenario:           "with www and without scheme and without version",
			url:                "www.github.com/owner/my-plugin",
			expectedOwner:      "owner",
			expectedRepository: "my-plugin",
		},
		{
			scenario:           "with www and without scheme and with version",
			url:                "www.github.com/owner/my-plugin@v1.1.0",
			expectedOwner:      "owner",
			expectedRepository: "my-plugin",
			expectedVersion:    "v1.1.0",
		},
		{
			scenario:           "with www and with http:// and without version",
			url:                "http://www.github.com/owner/my-plugin",
			expectedOwner:      "owner",
			expectedRepository: "my-plugin",
		},
		{
			scenario:           "with www and with https:// and without version",
			url:                "https://www.github.com/owner/my-plugin",
			expectedOwner:      "owner",
			expectedRepository: "my-plugin",
		},
		{
			scenario:           "with www and with scheme and version",
			url:                "https://www.github.com/owner/my-plugin@v1.1.0",
			expectedOwner:      "owner",
			expectedRepository: "my-plugin",
			expectedVersion:    "v1.1.0",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.scenario, func(t *testing.T) {
			t.Parallel()

			owner, repository, version, err := parseURL(tc.url)

			assert.Equal(t, tc.expectedOwner, owner)
			assert.Equal(t, tc.expectedRepository, repository)
			assert.Equal(t, tc.expectedVersion, version)

			if tc.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tc.expectedError)
			}
		})
	}
}

func TestTrimVersion(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		scenario string
		version  string
		expected string
	}{
		{
			scenario: "master",
			version:  "master",
			expected: "master",
		},
		{
			scenario: "only v",
			version:  "v",
			expected: "v",
		},
		{
			scenario: "volley",
			version:  "volley",
			expected: "volley",
		},
		{
			scenario: "no v prefix",
			version:  "1.3.2",
			expected: "1.3.2",
		},
		{
			scenario: "with v prefix",
			version:  "v1.3.2",
			expected: "1.3.2",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.scenario, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.expected, trimVersion(tc.version))
		})
	}
}

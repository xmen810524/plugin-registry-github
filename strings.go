package github

import (
	"context"
	"regexp"
	"strings"

	"github.com/bool64/ctxd"
)

var trimVersionPattern = regexp.MustCompile(`^[vV][0-9]+\.[0-9]+(\.[0-9]+)?`)

func splitURL(url string) (string, string) {
	parts := strings.SplitN(url, "@", 2)

	if len(parts) == 1 {
		parts = append(parts, "")
	}

	return parts[0], parts[1]
}

func getOwnerRepository(pluginURL string) (string, string) {
	pluginURL = stripHostname(pluginURL, githubHostname)
	pluginURL = strings.TrimPrefix(pluginURL, "/")
	pluginURL = strings.TrimSuffix(pluginURL, "/")

	parts := strings.SplitN(pluginURL, "/", 2)

	if len(parts) == 1 {
		parts = append(parts, "")
	}

	return parts[0], parts[1]
}

//goland:noinspection HttpUrlsUsage
func stripScheme(url string) string {
	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimPrefix(url, "http://")
	url = strings.TrimPrefix(url, "www.")

	return url
}

func stripHostname(url, hostname string) string {
	url = stripScheme(url)
	url = strings.TrimPrefix(url, hostname)

	return url
}

// isPlugin checks whether the given plugin URL is from github or not.
func isPlugin(pluginURL string) bool {
	_, _, _, err := parseURL(pluginURL) // nolint: dogsled

	return err == nil
}

// parseURL parses the url to owner, repository and version.
func parseURL(pluginURL string) (owner, repository, version string, err error) {
	pluginURL, version = splitURL(pluginURL)
	url := stripScheme(pluginURL)

	if !strings.HasPrefix(url, githubHostname) {
		return "", "", "", ErrNotGithub
	}

	owner, repository = getOwnerRepository(pluginURL)

	if owner == "" {
		return "", "", "", ErrMissingOwner
	}

	if repository == "" {
		return "", "", "", ErrMissingRepository
	}

	return owner, repository, version, nil
}

func parseError(err error, pluginURL string) error {
	return ctxd.WrapError(context.Background(), err, "could not parse url", "url", pluginURL)
}

func trimVersion(v string) string {
	if trimVersionPattern.MatchString(v) {
		return strings.TrimPrefix(v, "v")
	}

	return v
}

package github

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"sync"

	"github.com/bool64/ctxd"
	"github.com/google/go-github/v35/github"
	_ "github.com/nhatthm/plugin-registry-fs" // Add filesystem installer.
	fsCtx "github.com/nhatthm/plugin-registry/context"
	"github.com/nhatthm/plugin-registry/installer"
	"github.com/nhatthm/plugin-registry/plugin"
	"github.com/spf13/afero"
)

const githubHostname = "github.com"

var (
	// ErrNotGithub indicates the url is not github.
	ErrNotGithub = errors.New("not a github url")
	// ErrMissingOwner indicates that there is no owner in the url.
	ErrMissingOwner = errors.New("missing github owner")
	// ErrMissingRepository indicates that there is no repository in the url.
	ErrMissingRepository = errors.New("missing github repository")
)

func init() { // nolint: gochecknoinits
	RegisterInstaller()
}

type contextKey string

// Option is option to configure Installer.
type Option func(i *Installer)

// Installer installs plugin from github.
type Installer struct {
	fs      afero.Fs
	service RepositoryService

	baseURL *url.URL

	mu sync.Mutex
}

// Install installs the plugin.
// The installer will download the archive or binary from github, then uses the filesystem installers to install the
// plugin.
func (i *Installer) Install(ctx context.Context, dest, source string) (*plugin.Plugin, error) {
	i.mu.Lock()
	defer i.mu.Unlock()

	owner, repository, version, err := parseURL(source)
	if err != nil {
		return nil, parseError(err, source)
	}

	ctx = context.WithValue(ctx, contextKey("source"), source)
	ctx = context.WithValue(ctx, contextKey("owner"), owner)
	ctx = context.WithValue(ctx, contextKey("repository"), repository)

	return i.install(ctx, dest, owner, repository, version)
}

func (i *Installer) install(ctx context.Context, dest, owner, repository, version string) (*plugin.Plugin, error) {
	if version == "" || version == "latest" {
		r, _, err := i.service.GetLatestRelease(ctx, owner, repository)
		if err != nil {
			return nil, ctxd.WrapError(ctx, err, "could not find latest release")
		}

		if r.TagName == nil {
			return nil, ctxd.NewError(ctx, "latest release has no tag name")
		}

		return i.installRelease(ctx, dest, owner, repository, r)
	}

	r, _, err := i.service.GetReleaseByTag(ctx, owner, repository, version)
	if err != nil {
		return nil, ctxd.WrapError(ctx, err, "could not get release")
	}

	return i.installRelease(ctx, dest, owner, repository, r)
}

func (i *Installer) installRelease(ctx context.Context, dest string, owner, repository string, release *github.RepositoryRelease) (*plugin.Plugin, error) {
	r, _, err := i.service.DownloadContents(ctx, owner, repository, plugin.MetadataFile, &github.RepositoryContentGetOptions{
		Ref: *release.TagName,
	})
	if err != nil {
		return nil, ctxd.WrapError(ctx, err, "could not get plugin metadata", "version", *release.TagName)
	}

	p, err := loadMetadata(r)
	if err != nil {
		return nil, ctxd.WrapError(ctx, err, "could not load plugin metadata")
	}

	p.Version = trimVersion(*release.TagName)
	p.URL = fmt.Sprintf("https://github.com/%s/%s", owner, repository)

	return i.installPluginRelease(ctx, dest, owner, repository, p, release)
}

func (i *Installer) installPluginRelease(
	ctx context.Context,
	dest string,
	owner, repository string,
	p *plugin.Plugin,
	release *github.RepositoryRelease,
) (*plugin.Plugin, error) {
	artifact := p.ResolveArtifact(p.RuntimeArtifact())

	asset, err := findAsset(release, artifact.File)
	if err != nil {
		return nil, ctxd.WrapError(ctx, err, "could not find artifact")
	}

	return i.installPluginReleaseAsset(ctx, dest, owner, repository, p, asset)
}

func (i *Installer) installPluginReleaseAsset(
	ctx context.Context,
	dest string,
	owner, repository string,
	p *plugin.Plugin,
	asset *github.ReleaseAsset,
) (*plugin.Plugin, error) {
	ctx = context.WithValue(ctx, contextKey("asset"), asset)

	r, _, err := i.service.DownloadReleaseAsset(ctx, owner, repository, *asset.ID, http.DefaultClient)
	if err != nil {
		return nil, ctxd.WrapError(ctx, err, "could not download artifact")
	}

	tmpDir, err := afero.TempDir(i.fs, "", "plugin-registry-github-")
	if err != nil {
		return nil, ctxd.WrapError(ctx, err, "could not create temp dir")
	}

	defer func() {
		_ = i.fs.RemoveAll(tmpDir) // nolint: errcheck
	}()

	assetFile := filepath.Join(tmpDir, *asset.Name)

	if err := writeFile(i.fs, assetFile, r); err != nil {
		return nil, ctxd.WrapError(ctx, err, "could not write artifact")
	}

	if err := chmod(i.fs, asset.ContentType, assetFile, 0755); err != nil {
		return nil, ctxd.WrapError(ctx, err, "could not chmod artifact")
	}

	if err := writeMetadata(i.fs, tmpDir, p); err != nil {
		return nil, ctxd.WrapError(ctx, err, "could not write plugin metadata")
	}

	source := assetFile
	if filepath.Ext(assetFile) == "" {
		source = tmpDir
	}

	ctx = fsCtx.WithFs(ctx, i.fs)

	pkgInstaller, err := installer.Find(ctx, source)
	if err != nil {
		return nil, err
	}

	return pkgInstaller.Install(ctx, dest, source)
}

// WithService sets the repository service.
func (i *Installer) WithService(service RepositoryService) *Installer {
	i.mu.Lock()
	defer i.mu.Unlock()

	i.service = service

	return i
}

// NewInstaller initiates a new github installer.
func NewInstaller(options ...Option) *Installer {
	i := &Installer{
		fs: afero.NewOsFs(),
	}

	for _, o := range options {
		o(i)
	}

	if i.service == nil {
		c := github.NewClient(nil)

		if i.baseURL != nil {
			c.BaseURL = i.baseURL
		}

		i.service = c.Repositories
	}

	return i
}

// WithFs sets the file system.
func WithFs(fs afero.Fs) Option {
	return func(i *Installer) {
		i.fs = fs
	}
}

// WithService sets the repository service.
func WithService(service RepositoryService) Option {
	return func(i *Installer) {
		i.WithService(service)
	}
}

// WithBaseURL sets the github base url.
func WithBaseURL(url *url.URL) Option {
	return func(i *Installer) {
		i.baseURL = url
	}
}

// RegisterInstaller registers the installer.
func RegisterInstaller(options ...Option) {
	installer.Register(githubHostname,
		func(ctx context.Context, pluginURL string) bool {
			return isPlugin(pluginURL)
		},
		func(fs afero.Fs) installer.Installer {
			return NewInstaller(append(options, WithFs(fs))...)
		},
	)
}

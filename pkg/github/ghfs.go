package github

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"path"

	"github.com/google/go-github/v43/github"
	"github.com/vito/bass/pkg/bass"
	"github.com/vito/bass/pkg/zapctx"
	"go.uber.org/zap"
)

// GHFS is a filesystem that reads from a GitHub repo at a given commit.
type GHFS struct {
	Ctx    context.Context
	Client *github.Client
	Repo   *github.Repository
	Branch *github.Branch
}

func NewGHPath(ctx context.Context, client *github.Client, repo *github.Repository, branch *github.Branch, subPath string) bass.FSPath {
	return bass.NewFSPath(
		path.Join(client.BaseURL.Host, repo.GetFullName(), branch.GetCommit().GetSHA()),
		GHFS{
			Ctx:    ctx,
			Client: client,
			Repo:   repo,
			Branch: branch,
		},
		bass.ParseFileOrDirPath(subPath),
	)
}

func (ghfs GHFS) Open(name string) (fs.File, error) {
	logger := zapctx.FromContext(ghfs.Ctx)

	logger.Info("opening github file", zap.String("name", name))

	rc, _, err := ghfs.Client.Repositories.DownloadContents(
		ghfs.Ctx,
		ghfs.Repo.GetOwner().GetLogin(),
		ghfs.Repo.GetName(),
		name,
		&github.RepositoryContentGetOptions{
			Ref: ghfs.Branch.GetCommit().GetSHA(),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("get %s: %w", name, err)
	}

	return &ghFile{rc, logger, ghfs, name}, nil
}

type ghFile struct {
	io.ReadCloser

	logger *zap.Logger
	fs     GHFS
	name   string
}

func (f *ghFile) Close() error {
	f.logger.Debug("closing github file", zap.String("name", f.name))
	return f.ReadCloser.Close()
}

func (f *ghFile) Stat() (fs.FileInfo, error) {
	// this would require loading the content into memory via GetContents; not
	// worth it.
	return nil, fmt.Errorf("GitHubFile.Stat unsupported")
}

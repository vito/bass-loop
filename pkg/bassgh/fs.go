package bassgh

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"path"
	"sync"
	"time"

	"github.com/google/go-github/v43/github"
	"github.com/vito/bass/pkg/bass"
	"github.com/vito/bass/pkg/zapctx"
	"go.uber.org/zap"
)

// FS is a filesystem that reads from a GitHub repo at a given commit.
type FS struct {
	Ctx    context.Context
	Client *github.Client
	Repo   *github.Repository
	Ref    string

	cache  map[string]*github.RepositoryContent
	cacheL *sync.Mutex
}

func NewFS(ctx context.Context, client *github.Client, repo *github.Repository, ref, subPath string) *bass.FSPath {
	return bass.NewFSPath(
		&FS{
			Ctx:    ctx,
			Client: client,
			Repo:   repo,
			Ref:    ref,

			cache:  map[string]*github.RepositoryContent{},
			cacheL: new(sync.Mutex),
		},
		bass.ParseFileOrDirPath(subPath),
	)
}

func (ghfs *FS) Open(name string) (fs.File, error) {
	logger := zapctx.FromContext(ghfs.Ctx).With(
		zap.String("repo", ghfs.Repo.GetFullName()),
		zap.String("path", name),
		zap.String("sha", ghfs.Ref),
	)

	ghfs.cacheL.Lock()
	defer ghfs.cacheL.Unlock()

	file, cached := ghfs.cache[name]
	if cached {
		logger.Debug("cache hit")
	} else {
		logger.Info("fetching content")

		var err error
		file, _, _, err = ghfs.Client.Repositories.GetContents(
			ghfs.Ctx,
			ghfs.Repo.GetOwner().GetLogin(),
			ghfs.Repo.GetName(),
			name,
			&github.RepositoryContentGetOptions{
				Ref: ghfs.Ref,
			},
		)
		if err != nil {
			return nil, fmt.Errorf("get %s: %w", name, err)
		}

		ghfs.cache[name] = file
	}

	content, err := file.GetContent()
	if err != nil {
		return nil, fmt.Errorf("get content %s: %w", name, err)
	}

	return &ghFile{bytes.NewBufferString(content), logger, ghfs, name}, nil
}

type ghFile struct {
	io.Reader

	logger *zap.Logger
	fs     *FS
	name   string
}

func (f *ghFile) Close() error {
	return nil
}

func (f *ghFile) Stat() (fs.FileInfo, error) {
	// this would require loading the content into memory via GetContents; not
	// worth it.
	parent := path.Dir(f.name)
	filename := path.Base(f.name)

	_, dirContents, _, err := f.fs.Client.Repositories.GetContents(
		f.fs.Ctx,
		f.fs.Repo.GetOwner().GetLogin(),
		f.fs.Repo.GetName(),
		parent,
		&github.RepositoryContentGetOptions{
			Ref: f.fs.Ref,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("get %s: %w", parent, err)
	}

	for _, contents := range dirContents {
		if *contents.Name == filename {
			return &ghInfo{contents}, nil
		}
	}

	return nil, fmt.Errorf("file %s not found", f.name)
}

type ghInfo struct {
	*github.RepositoryContent
}

func (i ghInfo) Name() string {
	return i.RepositoryContent.GetName()
}

func (i ghInfo) IsDir() bool {
	return i.GetType() == "dir"
}

func (i ghInfo) ModTime() time.Time {
	return time.Time{}
}

func (i ghInfo) Mode() fs.FileMode {
	// made up
	if i.IsDir() {
		return fs.FileMode(0755)
	} else {
		return fs.FileMode(0644)
	}
}

func (i ghInfo) Size() int64 {
	return int64(i.GetSize())
}

func (i ghInfo) Sys() interface{} {
	return nil
}

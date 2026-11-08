package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/nicovegasr/co-change/internal/archmap"
	"github.com/nicovegasr/co-change/internal/history"
	"github.com/nicovegasr/co-change/internal/index"
	"github.com/nicovegasr/co-change/internal/pathignore"
)

// maxReadBytes skips generated or vendored giants when reading imports.
const maxReadBytes = 1 << 20

func runMap(ctx context.Context, dir string, args []string, out io.Writer) error {
	// code-maat defaults: changesets of at most 30 files, 5 shared revisions
	// and a 30% degree of coupling (ADR-0020).
	thresholds := archmap.Thresholds{MinSupport: 5, MinRatio: 0.3}
	maxFiles := 30
	fs := flags("map", out)
	outDir := fs.String("out", "co-change-map", "Folder to write index.html into")
	repository := fs.String("repo-url", githubRepositoryURL(), "Repository URL for README and commit links")
	fs.IntVar(&thresholds.MinSupport, "min-support", thresholds.MinSupport, "Minimum shared commits")
	fs.Float64Var(&thresholds.MinRatio, "min-ratio", thresholds.MinRatio, "Minimum co-change ratio, 0..1")
	fs.IntVar(&maxFiles, "max-files", maxFiles, "Ignore commits changing more files")
	positional, err := parseArgs(fs, args)
	if err != nil {
		return helpError(err)
	}
	if len(positional) != 0 {
		return errors.New("usage: co-change map [--out DIR] [options]")
	}
	if thresholds.MinSupport < 1 || maxFiles < 1 || thresholds.MinRatio < 0 || thresholds.MinRatio > 1 {
		return errors.New("min-support and max-files must be positive; min-ratio must be 0..1")
	}
	page, err := buildMap(ctx, dir, maxFiles, thresholds)
	if err != nil {
		return err
	}
	if page.Repository = *repository; page.Repository != "" {
		page.Name = path.Base(page.Repository)
	}
	if err = writePage(*outDir, page); err != nil {
		return err
	}
	hidden := 0
	for _, c := range page.CoChanges {
		if c.Hidden {
			hidden++
		}
	}
	_, err = fmt.Fprintf(out, "%s · %d folders · %d imports · %d change together · %d hidden couplings\n",
		filepath.Join(*outDir, "index.html"), len(page.Units), len(page.Imports), len(page.CoChanges), hidden)
	return err
}

// buildMap refreshes the index and maps the repository at HEAD, leaving out
// .cochangeignore paths.
func buildMap(ctx context.Context, dir string, maxFiles int, thresholds archmap.Thresholds) (archmap.Page, error) {
	if err := Refresh(ctx, dir); err != nil {
		return archmap.Page{}, err
	}
	repo, err := history.Open(ctx, dir)
	if err != nil {
		return archmap.Page{}, err
	}
	tracked, err := repo.Tracked(ctx)
	if err != nil {
		return archmap.Page{}, err
	}
	var sets []index.ChangeSet
	var ignore []string
	err = withFreshIndex(repo, func(store *index.Store, patterns []string) error {
		ignore = patterns
		sets, err = store.ChangeSets(ctx, maxFiles)
		return err
	})
	if err != nil {
		return archmap.Page{}, err
	}
	for i := range sets {
		sets[i].Paths = kept(sets[i].Paths, ignore)
	}
	read := func(p string) []byte {
		full := filepath.Join(repo.Root, filepath.FromSlash(p))
		if info, err := os.Stat(full); err != nil || info.Size() > maxReadBytes {
			return nil
		}
		data, _ := os.ReadFile(full)
		return data
	}
	return archmap.Page{
		Map:  archmap.Build(kept(tracked, ignore), read, sets, thresholds),
		Name: filepath.Base(repo.Root), Head: repo.Head, Generated: time.Now().UTC().Format("2006-01-02"),
		MinSupport: thresholds.MinSupport, MinRatio: thresholds.MinRatio, MaxFiles: maxFiles,
	}, nil
}

func kept(paths, ignore []string) []string {
	var out []string
	for _, p := range paths {
		if !pathignore.Matches(p, ignore) {
			out = append(out, p)
		}
	}
	return out
}

func writePage(dir string, page archmap.Page) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	f, err := os.Create(filepath.Join(dir, "index.html"))
	if err != nil {
		return err
	}
	return errors.Join(archmap.Render(f, page), f.Close())
}

// githubRepositoryURL is the repository's URL inside GitHub Actions, else "".
func githubRepositoryURL() string {
	server, repository := os.Getenv("GITHUB_SERVER_URL"), os.Getenv("GITHUB_REPOSITORY")
	if server == "" || repository == "" {
		return ""
	}
	return server + "/" + repository
}

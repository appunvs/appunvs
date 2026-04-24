// Package workspace owns the per-Box source tree: a bare Git repository
// on the relay's disk, mutated by the AI agent's fs_write / fs_delete tools
// and read back by sandbox.Builder when a publish fires.
//
// Keeping the source tree in Git gives us four properties for free:
//
//  1. Every AI turn's changes are a commit — full audit log, diff-able.
//  2. Rollback is `git reset`; no custom undo machinery.
//  3. Human-editing and AI-editing collide as standard merge conflicts.
//  4. "Push to user's GitHub" becomes a one-line git remote push.
//
// The repo is BARE (no working tree): we manipulate objects and refs
// directly via go-git, because a working directory would make concurrent
// AI and build access messy.
package workspace

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// ErrFileNotFound is returned by ReadFile when the path doesn't exist at HEAD.
var ErrFileNotFound = errors.New("workspace: file not found")

// Config holds the filesystem root under which each Box's bare repo lives.
type Config struct {
	Root string // e.g. /var/appunvs/workspaces
}

// Store owns the top-level root and hands out per-Box Repo instances.
type Store struct {
	root string
	// One mutex per repo; go-git is not safe for concurrent refs/HEAD writes
	// on the same storage. To keep the model simple we serialize all access
	// per repo.
	mu    sync.Mutex
	locks map[string]*sync.Mutex
}

// NewStore creates the root directory if needed and returns a Store.
func NewStore(cfg Config) (*Store, error) {
	if cfg.Root == "" {
		return nil, errors.New("workspace: Config.Root is required")
	}
	if err := os.MkdirAll(cfg.Root, 0o755); err != nil {
		return nil, fmt.Errorf("workspace: mkdir %s: %w", cfg.Root, err)
	}
	return &Store{root: cfg.Root, locks: make(map[string]*sync.Mutex)}, nil
}

func (s *Store) lockFor(boxID string) *sync.Mutex {
	s.mu.Lock()
	defer s.mu.Unlock()
	m, ok := s.locks[boxID]
	if !ok {
		m = &sync.Mutex{}
		s.locks[boxID] = m
	}
	return m
}

// Repo is a handle to one Box's bare repo.  Obtain via Store.Open.
type Repo struct {
	boxID string
	path  string
	repo  *git.Repository
	lock  *sync.Mutex
}

// Open returns the Repo for boxID, initializing the bare repo on first
// access.  Subsequent Open calls with the same boxID return handles sharing
// the same per-Box lock.
func (s *Store) Open(boxID string) (*Repo, error) {
	if strings.ContainsAny(boxID, "/\\.") {
		return nil, fmt.Errorf("workspace: illegal box id %q", boxID)
	}
	repoPath := filepath.Join(s.root, boxID)

	var r *git.Repository
	if _, err := os.Stat(filepath.Join(repoPath, "HEAD")); errors.Is(err, os.ErrNotExist) {
		r, err = git.PlainInit(repoPath, true)
		if err != nil {
			return nil, fmt.Errorf("workspace: init %s: %w", boxID, err)
		}
		// PlainInit leaves HEAD pointing at refs/heads/master by default.
		// Force it to main for consistency with current GitHub defaults.
		headRef := plumbing.NewSymbolicReference(plumbing.HEAD, plumbing.NewBranchReferenceName("main"))
		if err := r.Storer.SetReference(headRef); err != nil {
			return nil, fmt.Errorf("workspace: set HEAD: %w", err)
		}
	} else {
		r, err = git.PlainOpen(repoPath)
		if err != nil {
			return nil, fmt.Errorf("workspace: open %s: %w", boxID, err)
		}
	}

	return &Repo{
		boxID: boxID,
		path:  repoPath,
		repo:  r,
		lock:  s.lockFor(boxID),
	}, nil
}

// Commit is the condensed shape returned by Log.
type Commit struct {
	SHA     string
	Message string
	Author  string
	Time    time.Time
}

// WriteOp describes a single file mutation inside a commit.  Empty Content
// with Delete=false is accepted (creates a zero-byte file); Delete=true
// ignores Content entirely.
type WriteOp struct {
	Path    string
	Content []byte
	Delete  bool
}

// Commit batches WriteOps into one commit on top of HEAD and returns the
// new commit SHA.  Atomicity is at the go-git storer level: either the
// full commit lands or nothing does.
func (r *Repo) Commit(_ context.Context, ops []WriteOp, message, authorName, authorEmail string) (string, error) {
	r.lock.Lock()
	defer r.lock.Unlock()
	if len(ops) == 0 {
		return "", errors.New("workspace: Commit called with no ops")
	}
	if message == "" {
		message = "ai: update"
	}
	if authorName == "" {
		authorName = "appunvs"
	}
	if authorEmail == "" {
		authorEmail = "agent@appunvs"
	}

	// Build (path → fileEntry) starting from HEAD's tree, or empty if repo
	// has no commits yet.
	paths := map[string]fileEntry{}
	var parents []plumbing.Hash

	head, err := r.repo.Head()
	switch {
	case err == nil:
		headCommit, err := r.repo.CommitObject(head.Hash())
		if err != nil {
			return "", err
		}
		headTree, err := headCommit.Tree()
		if err != nil {
			return "", err
		}
		if err := headTree.Files().ForEach(func(f *object.File) error {
			paths[f.Name] = fileEntry{hash: f.Hash, mode: f.Mode}
			return nil
		}); err != nil {
			return "", err
		}
		parents = []plumbing.Hash{head.Hash()}
	case errors.Is(err, plumbing.ErrReferenceNotFound):
		// empty repo — first commit has no parents
	default:
		return "", err
	}

	for _, op := range ops {
		clean, err := normalizePath(op.Path)
		if err != nil {
			return "", err
		}
		if op.Delete {
			delete(paths, clean)
			continue
		}
		blobHash, err := writeBlob(r.repo, op.Content)
		if err != nil {
			return "", err
		}
		paths[clean] = fileEntry{hash: blobHash, mode: filemode.Regular}
	}

	rootTreeHash, err := writeTreeRecursive(r.repo, paths)
	if err != nil {
		return "", err
	}

	// No-op check: if the resulting tree matches HEAD's, skip.
	if len(parents) == 1 {
		parent, _ := r.repo.CommitObject(parents[0])
		if parent != nil && parent.TreeHash == rootTreeHash {
			return parents[0].String(), nil
		}
	}

	commit := object.Commit{
		Author:       nowSignature(authorName, authorEmail),
		Committer:    nowSignature(authorName, authorEmail),
		Message:      message,
		TreeHash:     rootTreeHash,
		ParentHashes: parents,
	}
	enc := r.repo.Storer.NewEncodedObject()
	if err := commit.Encode(enc); err != nil {
		return "", err
	}
	commitHash, err := r.repo.Storer.SetEncodedObject(enc)
	if err != nil {
		return "", err
	}

	ref := plumbing.NewHashReference(plumbing.NewBranchReferenceName("main"), commitHash)
	if err := r.repo.Storer.SetReference(ref); err != nil {
		return "", err
	}
	return commitHash.String(), nil
}

// Snapshot returns every file at HEAD as path → bytes.  Used by the
// sandbox.Builder adapter to feed a build.  An empty repo (no commits
// yet) returns an empty map, not an error.
func (r *Repo) Snapshot(_ context.Context) (map[string][]byte, error) {
	r.lock.Lock()
	defer r.lock.Unlock()
	head, err := r.repo.Head()
	if errors.Is(err, plumbing.ErrReferenceNotFound) {
		return map[string][]byte{}, nil
	}
	if err != nil {
		return nil, err
	}
	commit, err := r.repo.CommitObject(head.Hash())
	if err != nil {
		return nil, err
	}
	tree, err := commit.Tree()
	if err != nil {
		return nil, err
	}
	out := map[string][]byte{}
	err = tree.Files().ForEach(func(f *object.File) error {
		rd, err := f.Reader()
		if err != nil {
			return err
		}
		defer func() { _ = rd.Close() }()
		body, err := io.ReadAll(rd)
		if err != nil {
			return err
		}
		out[f.Name] = body
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ReadFile returns one file's contents at HEAD.  Returns ErrFileNotFound
// when the path isn't present.
func (r *Repo) ReadFile(_ context.Context, path string) ([]byte, error) {
	r.lock.Lock()
	defer r.lock.Unlock()
	clean, err := normalizePath(path)
	if err != nil {
		return nil, err
	}
	head, err := r.repo.Head()
	if err != nil {
		return nil, err
	}
	commit, err := r.repo.CommitObject(head.Hash())
	if err != nil {
		return nil, err
	}
	tree, err := commit.Tree()
	if err != nil {
		return nil, err
	}
	file, err := tree.File(clean)
	if err != nil {
		if errors.Is(err, object.ErrFileNotFound) {
			return nil, ErrFileNotFound
		}
		return nil, err
	}
	rd, err := file.Reader()
	if err != nil {
		return nil, err
	}
	defer func() { _ = rd.Close() }()
	return io.ReadAll(rd)
}

// ListFiles returns every tracked path at HEAD, sorted.  Empty repo
// yields an empty slice, not an error.
func (r *Repo) ListFiles(_ context.Context) ([]string, error) {
	r.lock.Lock()
	defer r.lock.Unlock()
	head, err := r.repo.Head()
	if errors.Is(err, plumbing.ErrReferenceNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	commit, err := r.repo.CommitObject(head.Hash())
	if err != nil {
		return nil, err
	}
	tree, err := commit.Tree()
	if err != nil {
		return nil, err
	}
	var out []string
	err = tree.Files().ForEach(func(f *object.File) error {
		out = append(out, f.Name)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(out)
	return out, nil
}

// Log returns the most recent `limit` commits on main (HEAD-first).  An
// empty repo yields nil, nil (not an error).
func (r *Repo) Log(_ context.Context, limit int) ([]Commit, error) {
	r.lock.Lock()
	defer r.lock.Unlock()
	if limit <= 0 {
		limit = 50
	}
	head, err := r.repo.Head()
	if errors.Is(err, plumbing.ErrReferenceNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	iter, err := r.repo.Log(&git.LogOptions{From: head.Hash()})
	if err != nil {
		return nil, err
	}
	defer iter.Close()
	var out []Commit
	err = iter.ForEach(func(c *object.Commit) error {
		out = append(out, Commit{
			SHA:     c.Hash.String(),
			Message: c.Message,
			Author:  c.Author.Name,
			Time:    c.Author.When,
		})
		if len(out) >= limit {
			return errIterStop
		}
		return nil
	})
	if err != nil && !errors.Is(err, errIterStop) {
		return nil, err
	}
	return out, nil
}

// errIterStop is the sentinel used to break out of go-git ForEach loops.
var errIterStop = errors.New("iter-stop")

// ----- internals -----

type fileEntry struct {
	hash plumbing.Hash
	mode filemode.FileMode
}

// normalizePath rejects absolute, escaping, or dot-prefixed paths and
// collapses the separator to "/".
func normalizePath(p string) (string, error) {
	if p == "" {
		return "", errors.New("workspace: empty path")
	}
	p = strings.ReplaceAll(p, "\\", "/")
	p = filepath.ToSlash(filepath.Clean(p))
	if strings.HasPrefix(p, "/") || strings.HasPrefix(p, "../") || p == ".." || p == "." {
		return "", fmt.Errorf("workspace: illegal path %q", p)
	}
	return p, nil
}

// writeBlob hashes and stores content as a git blob, returning its hash.
func writeBlob(r *git.Repository, content []byte) (plumbing.Hash, error) {
	obj := r.Storer.NewEncodedObject()
	obj.SetType(plumbing.BlobObject)
	obj.SetSize(int64(len(content)))
	w, err := obj.Writer()
	if err != nil {
		return plumbing.ZeroHash, err
	}
	if _, err := w.Write(content); err != nil {
		_ = w.Close()
		return plumbing.ZeroHash, err
	}
	if err := w.Close(); err != nil {
		return plumbing.ZeroHash, err
	}
	return r.Storer.SetEncodedObject(obj)
}

// writeTreeRecursive turns a flat path→fileEntry map into nested git trees
// and returns the root tree hash.  Bare repos don't have a working tree,
// so we construct trees by hand rather than go through Worktree.Add.
func writeTreeRecursive(r *git.Repository, flat map[string]fileEntry) (plumbing.Hash, error) {
	type child struct {
		files   map[string]fileEntry
		subdirs map[string]*child
	}
	root := &child{files: map[string]fileEntry{}, subdirs: map[string]*child{}}
	for path, entry := range flat {
		parts := strings.Split(path, "/")
		cur := root
		for i, p := range parts {
			if i == len(parts)-1 {
				cur.files[p] = entry
				break
			}
			next, ok := cur.subdirs[p]
			if !ok {
				next = &child{files: map[string]fileEntry{}, subdirs: map[string]*child{}}
				cur.subdirs[p] = next
			}
			cur = next
		}
	}

	var emit func(c *child) (plumbing.Hash, error)
	emit = func(c *child) (plumbing.Hash, error) {
		entries := make([]object.TreeEntry, 0, len(c.files)+len(c.subdirs))
		for name, e := range c.files {
			entries = append(entries, object.TreeEntry{Name: name, Mode: e.mode, Hash: e.hash})
		}
		for name, sub := range c.subdirs {
			h, err := emit(sub)
			if err != nil {
				return plumbing.ZeroHash, err
			}
			entries = append(entries, object.TreeEntry{Name: name, Mode: filemode.Dir, Hash: h})
		}
		sort.Slice(entries, func(i, j int) bool { return entries[i].Name < entries[j].Name })

		tree := object.Tree{Entries: entries}
		obj := r.Storer.NewEncodedObject()
		if err := tree.Encode(obj); err != nil {
			return plumbing.ZeroHash, err
		}
		return r.Storer.SetEncodedObject(obj)
	}
	return emit(root)
}

func nowSignature(name, email string) object.Signature {
	return object.Signature{Name: name, Email: email, When: time.Now().UTC()}
}

// ensureRemote is reserved for the "push to user's GitHub" feature.
func ensureRemote(r *git.Repository, name, url string) error {
	if _, err := r.Remote(name); err == nil {
		return nil
	}
	_, err := r.CreateRemote(&config.RemoteConfig{Name: name, URLs: []string{url}})
	return err
}

var _ = ensureRemote

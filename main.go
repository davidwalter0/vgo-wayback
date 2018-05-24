// https://gist.github.com/davidwalter0/60f41b53732656c5c546cc8b0a739d11

// x/vgo: thoughts on selection criteria for non-versioned go repo

// Run git branch version selection for non-vgo versioned repository
// as if in a wayback machine.

// Use the dependency's commit ctime as the wayback ctime.
// Commits prior to the wayback ctime in sub-dependencies are eligible.
// Use the newest commit prior to the wayback ctime.

// Rough sketch of wayback idea to find a candidate ctime of a git
// commit

// This would need to be integrated into vgo logic and of course
// non-git vc methods.

// Add tooling from go-git examples to remove the external dependency

// Example command line for gopkg.in/src-d/go-git

// Select some arbitrary ctime for the wayback time

// export ctime='2017-09-04 19:43:36 +0300';
// vgo run main.go /go/src/gopkg.in/src-d/go-git.v4 "${ctime}"

package main

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"gopkg.in/src-d/go-git.v4/plumbing/storer"
)

const (
	// Layout format spec for parsing
	Layout = "2006-01-02 15:04:05 -0700"
)

var (
	// NotFound error
	NotFound = fmt.Errorf("Reference not found")
)

type Wayback struct {
	Debug bool
	*git.Repository
	RequireTagInfo bool
	When           time.Time
	CommitIter     object.CommitIter
}

type TagInfo struct {
	Tag  string
	Hash plumbing.Hash
	When time.Time
}

// ByCommitTimeTagInfo sortable slice of TagInfo
type ByCommitTimeTagInfo []TagInfo

func (tags ByCommitTimeTagInfo) Len() int           { return len(tags) }
func (tags ByCommitTimeTagInfo) Less(i, j int) bool { return tags[i].When.Before(tags[j].When) }
func (tags ByCommitTimeTagInfo) Swap(i, j int)      { tags[i], tags[j] = tags[j], tags[i] }

// NewWayback create a Wayback object
func NewWayback(r *git.Repository, requireTagInfo bool, when time.Time, commitIter object.CommitIter) *Wayback {
	return &Wayback{Repository: r, RequireTagInfo: requireTagInfo, When: when, CommitIter: commitIter}
}

func (wayback *Wayback) Find() (c *object.Commit, tag string, err error) {
	defer wayback.CommitIter.Close()
	if wayback.RequireTagInfo {
		return wayback.FindFirstTag()
	}
	return wayback.FindFirst()
}

// FindFirst commit prior to the wayback time
func (wayback *Wayback) FindFirst() (c *object.Commit, tag string, err error) {
	if wayback.Debug {
		fmt.Printf("Searching for commit before: %-32.32s\n", wayback.When)
		fmt.Printf("%-12.12s %-32.32s %-12.12s\n", "Hash", "Commit Time", "Tag")
	}
	for {
		if c, err = wayback.CommitIter.Next(); err == io.EOF {
			break
		}
		if err != nil {
			c = nil
			return
		}
		if wayback.Debug {
			fmt.Printf("%-12.12s %-32.32s\n", c.ID(), c.Committer.When)
		}
		if c.Committer.When.Before(wayback.When) {
			return
		}
	}
	return
}

// FindFirstTag commit prior to the wayback time
func (wayback *Wayback) FindFirstTag() (c *object.Commit, tag string, err error) {
	var byCommitTimeTagInfo ByCommitTimeTagInfo
	var tags storer.ReferenceIter
	var ref *plumbing.Reference

	if tags, err = wayback.Repository.Tags(); err != nil {
		return
	}

	byCommitTimeTagInfo = make(ByCommitTimeTagInfo, 0)

	for {
		if ref, err = tags.Next(); err == io.EOF {
			break
		}
		if err != nil {
			c = nil
			return
		}
		if c, err = wayback.Repository.CommitObject(ref.Hash()); err != nil {
			c = nil
			return
		}
		byCommitTimeTagInfo = append(byCommitTimeTagInfo,
			TagInfo{Tag: ref.Name().Short(), Hash: ref.Hash(), When: c.Committer.When})
	}
	sort.Sort(sort.Reverse(byCommitTimeTagInfo))
	var tagInfo TagInfo
	if wayback.Debug {
		fmt.Printf("Searching for tagged commit newer than %-32.32s\n",
			wayback.When)
		fmt.Printf("%-12.12s %-32.32s %-12.12s\n", "Hash", "Commit Time", "Tag")
	}
	for _, tagInfo = range byCommitTimeTagInfo {
		if c, err = wayback.Repository.CommitObject(tagInfo.Hash); err != nil {
			c = nil
			return
		}
		if wayback.Debug {
			fmt.Printf("%-12.12s %-32.32s %-12.12s\n", tagInfo.Hash, c.Committer.When, tagInfo.Tag)
		}
		if c.Committer.When.Before(wayback.When) {
			tag = tagInfo.Tag
			err = nil
			return
		}
	}
	err = NotFound
	return
}

func Tag(repo *git.Repository) (tag string, isTag bool, err error) {
	var ref *plumbing.Reference
	var tags storer.ReferenceIter

	if ref, err = repo.Head(); err != nil {
		return
	}

	if tags, err = repo.Tags(); err != nil {
		return
	}

	if err = tags.ForEach(func(_ref *plumbing.Reference) error {
		if _ref.Hash().String() == ref.Hash().String() {
			isTag = true
			tag = _ref.Name().Short()
			return nil
		}
		return nil
	}); err != nil {
		return
	}
	err = NotFound
	return
}

// Open an existing repository in a specific folder.
func main() {
	fmt.Println()
	var When time.Time
	var err error
	var now = time.Now().Format(Layout)
	var help = fmt.Sprintf(`
Format wayback commit time to match layout

Layout                 %-32.32s
Formatted current time %-32.32s
`, Layout, now)
	CheckArgs(help, "<path>", "<wayback commit time>")

	path := os.Args[1]
	ctime := os.Args[2]

	if When, err = time.Parse(Layout, ctime); err != nil {
		CheckIfError(err)
	}
	// We instance a new repository targeting the given path (the .git folder)
	var r *git.Repository
	r, err = git.PlainOpen(path)
	CheckIfError(err)

	// ... retrieving the HEAD reference
	ref, err := r.Head()
	if err != nil {
		fmt.Println(err)
	}

	var cIter object.CommitIter
	var requireTagInfo bool = true
	cIter, err = r.Log(&git.LogOptions{From: ref.Hash()})
	CheckIfError(err)
	fmt.Printf("Type     %-12.12s %-12.12s %-20.20s\n", "Hash", "TagInfo", "Commit Time")
	// Find a tagged commit
	var wf = NewWayback(r, requireTagInfo, When, cIter)
	var c *object.Commit
	var tag string
	// wf.Debug = true
	c, tag, err = wf.Find()
	if err != nil {
		fmt.Println(err)
	}
	if c != nil {
		fmt.Printf("Tagged   %-12.12s %-12.12s %-20.20s\n", c.Hash, tag, c.Committer.When)
	}

	cIter, err = r.Log(&git.LogOptions{From: ref.Hash()})
	CheckIfError(err)
	// Find any newer commit
	wf = NewWayback(r, !requireTagInfo, When, cIter)
	// wf.Debug = true
	c, tag, err = wf.Find()
	if err != nil {
		fmt.Println(err)
	}
	if c != nil {
		fmt.Printf("Untagged %-12.12s %-12.12s %-20.20s\n", c.Hash, tag, c.Committer.When)
	}
}

// common tooling from examples

// CheckArgs should be used to ensure the right command line arguments are
// passed before executing an example.
func CheckArgs(help string, arg ...string) {
	if len(os.Args) < len(arg)+1 {
		UseMessage(help, "Usage:", "%s %s", os.Args[0], strings.Join(arg, " "))
		os.Exit(1)
	}
}

// CheckIfError should be used to naively panics if an error is not nil.
func CheckIfError(err error) {
	if err == nil {
		return
	}

	fmt.Printf("\x1b[31;1m%s\x1b[0m\n", fmt.Sprintf("error: %s", err))
	os.Exit(1)
}

// Info should be used to describe the example commands that are about to run.
func Info(format string, args ...interface{}) {
	fmt.Printf("\x1b[34;1m%s\x1b[0m\n", fmt.Sprintf(format, args...))
}

// Warning should be used to display a warning
func Warning(format string, args ...interface{}) {
	fmt.Printf("\x1b[36;1m%s\x1b[0m\n", fmt.Sprintf(format, args...))
}

// UseMessage should be used to display a Error
func UseMessage(help, prefix, format string, args ...interface{}) {
	fmt.Printf("\n%s \x1b[36;1m%s\x1b[0m\n%s\n", prefix, fmt.Sprintf(format, args...), help)
}

---
vgo wayback 

`go get repo...` and `go get -u` updates with the latest available commit.

How would you find likely versions that go get would find if you were
living in the past, you went wayback, what would be the latest
available at the wayback time?

If you had a wayback machine, you'd set your time to the commit time
of module or tag -- the wayback time of the module, then for each
dependency successively look for the commit or tag nearest the current
modules wayback time.

```
// https://gist.github.com/davidwalter0/60f41b53732656c5c546cc8b0a739d11

// x/vgo: thoughts on selection criteria for non-versioned go repo

// Run git branch version selection for non-vgo versioned repository
// as if in a wayback machine.

// Use the dependency's commit ctime as the wayback ctime.
// Commits prior to the wayback ctime in sub-dependencies are eligible.
// Use the newest commit prior to the wayback ctime.

// Rough sketch of wayback idea to find a candidate ctime of a git commit

// This would need to be integrated into vgo logic and of course
// non-git vc methods.

// Add tooling from go-git examples to remove the external dependency

// Example command line for gopkg.in/src-d/go-git

// Select some arbitrary ctime for the wayback time

// export ctime='2017-09-04 19:43:36 +0300';
// vgo run main.go /go/src/gopkg.in/src-d/go-git.v4 "${ctime}"
```


---

Not to be confused with the phonetically similar *we go wayback*

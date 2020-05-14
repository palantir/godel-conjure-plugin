backcompat-cli-bundler
==================
`backcompat-cli-bundler` is a Go library that embeds the backcompat verification CLI published by foundry/conjure-backcompat.
The Conjure repository publishes a CLI to evaluate whether two Conjure IR files are wire-format compatible, 
intended to relieve humans of the burden of catching breaks. Intended to run as part of CI, before a PR merges.
This library fully embeds the CLI published by that repository in Go
source code using the [go-bindata](https://github.com/go-bindata/go-bindata) library and provides a Go API for working
with the CLI.

The Go library works as follows:
* The library fully embeds a specific version of the Conjure Backcompat CLI in source
* When library functionality that requires the embedded CLI is invoked, the embedded CLI is written to disk and invoked
  * The embedded CLI data is written to `{{tmp}}/_conjureircli/conjure-{{version}}`, where `{{tmp}}` is the directory 
    returned by `os.TempDir()` and `{{version}}` is the version of the CLI embedded in the library
  * If the CLI already exists in that location, it is invoked directly (not written out)
  
Note that, currently, the Conjure Backcompat CLI is written in Java, and thus invoking the CLI requires the Java runtime.

Updating the bundled CLI
------------------------
To update the version of the CLI bundled in source, do the following:

* Determine the new version of foundry/conjure-backcompat (it must be available in GHE) 
* Update the value of the `conjureVersion` constant in `conjureircli/generator/generate.go` to the desired version
* Run `./godelw generate` to embed the updated version in source

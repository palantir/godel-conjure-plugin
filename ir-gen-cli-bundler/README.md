ir-gen-cli-bundler
==================
`ir-gen-cli-bundler` contains Go wrappers for packaged Conjure CLIs. Each wrapper embeds its CLI distribution using
`go:embed`, extracts it with `clipackager`, and exposes the supported operations through a Go API.

conjureircli
------------
`conjureircli` embeds the CLI published by the [Conjure repository](https://github.com/palantir/conjure) and exposes
operations that compile Conjure YAML into JSON intermediate representation (IR). The CLI is written in Java, so using
this wrapper requires a Java runtime.

conjuretypescriptcli
--------------------
`conjuretypescriptcli` embeds the
[conjure-typescript](https://github.com/palantir/conjure-typescript) generator and exposes its `generate` operation. The
generator consumes Conjure IR and produces a TypeScript client project. It is a Node bundle, so using this wrapper
requires `node` on `PATH`.

Updating the bundled CLIs
-------------------------
To update the Conjure IR CLI:

* Select a Conjure version available from Maven Central.
* Update the `conjureTGZVersion` constant in `conjureircli/generator/generate.go`.
* Run `./godelw generate` to embed the updated version in source.

To update the conjure-typescript CLI, select a version available from Maven Central, update the
`conjureTypeScriptTGZVersion` constant in `conjuretypescriptcli/generator/generate.go`, and run `./godelw generate`.

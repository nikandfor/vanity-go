# vanity-go

`vanity-go` is a vanity page generator and server.
It allows you to import your Go modules using custom domains (e.g., `nikand.dev/go/vanity-go`) while still hosting them on GitHub or another platform.

I personally use it with GitHub Pages — see the example in the [`nikand.dev` repository](https://github.com/nikandfor/nikand.dev).

To set it up, you need:
```
vanity.yaml                   # config with your modules
.github/workflows/vanity.yaml # github workflow to generate and deploy pages
```
Official documentation on this topic is limited, but you can find a brief reference here: [Go Reference](https://go.dev/ref/mod#vcs-find).

## vanity.yaml reference

### Modules

`modules` is a list of modules to generate or serve.

```
modules:
  - module: tlog.app/go/tlog # module path, works if module path matches repo root
                             # ie go.mod is in the root of the repo

  - tlog.app/go/tlog         # short form of the previous example

  - module: tlog.app/go/tlog/ext/tlgin # module in the subfolder of the repo
    repo_root: tlog.app/go/tlog        # module path prefix pointing to repo root
                                       # replace rules applied to repo_root

  - module: tlog.app/go/tlog/ext/tlgin
    repo: https://github.com/tlog-dev/tlog # specify repository, no substitutions are applied
```

### Replace

`replace` is a list of replace rules for module paths.
Replaces work on strings, not paths, so trailing slashes matter.
Replace rules tried in the declaration order, first match applied.

```
replace:
  - prefix: tlog.app/go/
    repo: https://github.com/tlog-dev/
```

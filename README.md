# vanity-go

`vanity-go` is a vanity page generator and server.
It allows you to import your Go modules using custom domains (e.g., `nikand.dev/go/vanity-go`) while still hosting them on GitHub or another platform.

I personally use it with GitHub Pages — see examples in
[nikand.dev](https://github.com/nikandfor/nikand.dev) and
[tlog.app](https://github.com/tlog-dev/tlog.app) repositories.

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

  - module: tlog.app/go/tlog/ext/tlgin # module in a repo subfolder
    root: tlog.app/go/tlog             # module path prefix pointing to the repo root
                                       # replace rules applied to root

                                  # fancy syntax
  - tlog.app/go/tlog:             # common prefix; repo root
    - /                           # list of modules under the prefix
    - /ext/tlgin                  # each subdirectory that has go.mod in it
    - /cmd/tlog
    url: https://github.com/tlog-dev/tlog # override common replace rule

  - module: tlog.app/go/tlog/ext/tlgin
    root: tlog.app/go/tlog
    url: https://github.com/tlog-dev/tlog # specify repository, no substitutions are applied
```

### Replace

`replace` is a list of replace rules for module paths.
Rule is applied to the repo root, not the full path.
Replaces work on strings, not paths, so trailing slashes matter.
Replace rules tried in the declaration order, first match applied.

```
replace:
  - prefix: tlog.app/go/
    url: https://github.com/tlog-dev/
```

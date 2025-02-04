# vanity-go

`vanity-go` is a vanity page generator. It allows you to import your Go modules using custom domains (e.g., `nikand.dev/go/vanity-go`) while still hosting them on GitHub or another platform.

I personally use it with GitHub Pages — see the example in the [`nikand.dev` repository](https://github.com/nikandfor/nikand.dev).

To set it up, you need:
```
vanity.yaml                   # config with your modules
.github/workflows/vanity.yaml # github workflow to generate and deploy pages
```
Official documentation on this topic is limited, but you can find a brief reference here: [Go Reference](https://go.dev/ref/mod#vcs-find).

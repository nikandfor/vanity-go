package main

import (
	stderrors "errors"
	"html/template"
	"io"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"nikand.dev/go/cli"
	"nikand.dev/go/hacked/low"
	"tlog.app/go/errors"
	"tlog.app/go/tlog"
	"tlog.app/go/tlog/ext/tlflag"

	"nikand.dev/go/vanity-go/configdecoder"
)

type (
	Config      = configdecoder.Config
	Module      = configdecoder.Module
	Replacement = configdecoder.Replacement

	Params struct {
		Package string `json:"pkg"`
		Root    string `json:"root"` // import prefix
		VCS     string `json:"vcs"`
		URL     string `json:"repo"` // url
	}
)

var ErrReplacementNotFound = stderrors.New("replacement not found")

func main() {
	serveCmd := &cli.Command{
		Name:   "serve,server",
		Action: serveRun,
		Flags: []*cli.Flag{
			cli.NewFlag("listen,l", ":80", "address to listen to"),
		},
	}

	staticCmd := &cli.Command{
		Name:   "generate,gen,static",
		Action: staticRun,
		Flags: []*cli.Flag{
			cli.NewFlag("output,o", "_site", "output directory"),
			//	cli.NewFlag("remove,rm", false, "remove static dir before start"),
		},
	}

	app := &cli.Command{
		Name:        "vanity",
		Description: "tool for making go vanity module names easy to use",
		Before:      before,
		Flags: []*cli.Flag{
			cli.NewFlag("config,c", "vanity.yaml", "repos"),

			cli.NewFlag("log", "stderr?dm", "log output file (or stderr)"),
			cli.NewFlag("verbosity,v", "", "logger verbosity topics"),
			cli.NewFlag("debug", "", "debug address"),
			cli.FlagfileFlag,
			cli.HelpFlag,
		},
		Commands: []*cli.Command{
			serveCmd,
			staticCmd,
		},
	}

	cli.RunAndExit(app, os.Args, os.Environ())
}

func before(c *cli.Command) error {
	w, err := tlflag.OpenWriter(c.String("log"))
	if err != nil {
		return errors.Wrap(err, "open log file")
	}

	tlog.DefaultLogger = tlog.New(w)

	tlog.SetVerbosity(c.String("verbosity"))

	if q := c.String("debug"); q != "" {
		l, err := net.Listen("tcp", q)
		if err != nil {
			return errors.Wrap(err, "listen debug")
		}

		tlog.Printw("start debug interface", "addr", l.Addr())

		go func() {
			err := http.Serve(l, nil)
			if err != nil {
				tlog.Printw("debug", "addr", q, "err", err, "", tlog.Fatal)
				panic(err)
			}
		}()
	}

	return nil
}

func serveRun(c *cli.Command) (err error) {
	cfgs, err := loadConfig(c.String("config"))
	if err != nil {
		return errors.Wrap(err, "load config")
	}

	tlog.Printw("config loaded", "configs", len(cfgs), "path", c.String("config"))

	l, err := net.Listen("tcp", c.String("listen"))
	if err != nil {
		return errors.Wrap(err, "listen")
	}

	tlog.Printw("serving", "addr", l.Addr())

	for i, cfg := range cfgs {
		if len(cfgs) > 1 {
			tlog.Printw("config", "index", i)
		}

		for _, m := range cfg.Modules {
			tlog.Printw("module", "module", m.Module, "root", m.Root, "vcs", m.VCS, "url", m.URL)
		}

		for _, r := range cfg.Replace {
			tlog.Printw("replace", "prefix", r.Prefix, "url", r.URL, "vcs", r.VCS)
		}
	}

	err = http.Serve(l, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		var err error
		var conf *Config
		var module Module

		tr := tlog.Start("request", "method", req.Method, "host", req.Host, "path", req.URL.Path, "query", req.URL.RawQuery)
		defer tr.Finish("module", &module.Module, "err", &err)

		pkg := path.Join(req.Host, req.URL.Path)

		for _, cfg := range cfgs {
			for _, mod := range cfg.Modules {
				if !strings.HasPrefix(pkg, mod.Module) {
					continue
				}

				if len(mod.Module) > len(module.Module) {
					conf = cfg
					module = mod
				}
			}
		}

		if conf == nil {
			http.NotFound(w, req)
			return
		}

		err = GeneratePage(w, pkg, module, conf.Replace)
		if errors.Is(err, ErrReplacementNotFound) {
			http.NotFound(w, req)
		} else if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}))

	return nil
}

func staticRun(c *cli.Command) (err error) {
	cfgs, err := loadConfig(c.String("config"))
	if err != nil {
		return errors.Wrap(err, "load config")
	}

	root := c.String("output")
	root = filepath.Clean(root)

	for _, cfg := range cfgs {
		for _, module := range cfg.Modules {
			var buf low.Buf

			err := GeneratePage(&buf, module.Module, module, cfg.Replace)
			if err != nil {
				return errors.Wrap(err, module.Module)
			}

			domain := strings.IndexRune(module.Module, '/')

			fname := filepath.FromSlash(module.Module[domain+1:])
			fname = filepath.Join(fname, "index.html")

			full := filepath.Join(root, fname)
			dir := filepath.Dir(full)

			tlog.Printw("writing module", "module", module, "path", full)

			err = os.MkdirAll(dir, 0o755)
			if err != nil {
				return errors.Wrap(err, "mkdir")
			}

			err = os.WriteFile(full, buf, 0o644)
			if err != nil {
				return errors.Wrap(err, "write file")
			}
		}
	}

	return nil
}

func GeneratePage(w io.Writer, pkg string, mod Module, reps []Replacement) (err error) {
	p := Params{
		Package: pkg,
		Root:    first(mod.Root, mod.Module),
		VCS:     mod.VCS,
		URL:     mod.URL,
	}

	if p.URL == "" {
		for _, rep := range reps {
			if !strings.HasPrefix(p.Root, rep.Prefix) {
				continue
			}

			p.URL = strings.Replace(p.Root, rep.Prefix, rep.URL, 1)
			p.VCS = first(mod.VCS, rep.VCS)

			break
		}
	}

	if p.URL == "" {
		return ErrReplacementNotFound
	}

	if p.VCS == "" {
		p.VCS = "git"
	}

	err = repoPage.Execute(w, p)
	if err != nil {
		return errors.Wrap(err, "exec page template")
	}

	return nil
}

func loadConfig(name string) ([]*Config, error) {
	data, err := os.ReadFile(name)
	if err != nil {
		return nil, errors.Wrap(err, "read file")
	}

	return configdecoder.Decode(data)
}

func first(s ...string) string {
	for _, s := range s {
		if s != "" {
			return s
		}
	}

	return ""
}

var repoPage = template.Must(template.New("page").Parse(`<!DOCTYPE html>
{{- define "godoc" }}https://pkg.go.dev/{{ . }}{{ end }}
<html lang=en-US>
<head>
	<meta name="go-import" content="{{ .Root }} {{ .VCS }} {{ .URL }}">
	<meta http-equiv="Refresh" content="3; url='{{ template "godoc" .Package }}'" />
</head>
<body>
	Redirecting to <a href="{{ template "godoc" .Package }}">{{ template "godoc" .Package }}</a>...
</body>
</html>
`))

package main

import (
	"html/template"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/nikandfor/cli"
	"github.com/nikandfor/errors"
	"github.com/nikandfor/tlog"
	"github.com/nikandfor/tlog/ext/tlflag"
	"github.com/nikandfor/tlog/low"
	"gopkg.in/yaml.v3"
)

type (
	Module struct {
		Name string `json:"name"` // import prefix
		VCS  string `json:"vcs"`
		Repo string `json:"repo"` // url
	}

	Config struct {
		Modules []string `json:"modules"`
		Replace []struct {
			Prefix string `json:"prefix"`
			Repo   string `json:"repo"`
			VCS    string `json:"vcs"`
		} `json:"replace"`
	}
)

func main() {
	serveCmd := &cli.Command{
		Name:   "serve",
		Action: serveRun,
		Flags: []*cli.Flag{
			cli.NewFlag("listen,l", ":80", "address to listen to"),
		},
	}

	staticCmd := &cli.Command{
		Name:   "generate,gen,static",
		Action: staticRun,
		Flags: []*cli.Flag{
			cli.NewFlag("output,o", "static", "output directory"),
			//	cli.NewFlag("remove,rm", false, "remove static dir before start"),
		},
	}

	app := &cli.Command{
		Name:        "vanity",
		Description: "tool for making go vanity module names easy to use",
		Before:      before,
		Flags: []*cli.Flag{
			cli.NewFlag("config", "vanity.yaml", "repos"),

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
	cfg, err := loadConfig(c.String("config"))
	if err != nil {
		return errors.Wrap(err, "load config")
	}

	l, err := net.Listen("tcp", c.String("listen"))
	if err != nil {
		return errors.Wrap(err, "listen")
	}

	tlog.Printw("serving", "addr", l.Addr())

	err = http.Serve(l, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		var err error

		tr := tlog.Start("request", "method", req.Method, "host", req.Host, "path", req.URL.Path, "query", req.URL.RawQuery)
		defer tr.Finish("err", &err)

		prefix := path.Join(req.Host, req.URL.Path)

		var module string

		for _, mod := range cfg.Modules {
			if !strings.HasPrefix(prefix, mod) {
				continue
			}

			module = mod

			break
		}

		if module == "" {
			http.NotFound(w, req)
			return
		}

		for _, rep := range cfg.Replace {
			if !strings.HasPrefix(prefix, rep.Prefix) {
				continue
			}

			repo := strings.Replace(module, rep.Prefix, rep.Repo, 1)
			vcs := rep.VCS

			if vcs == "" {
				vcs = "git"
			}

			err = repoPage.Execute(w, Module{
				Name: prefix,
				VCS:  vcs,
				Repo: repo,
			})
			if err != nil {
				err = errors.Wrap(err, "exec page template")
			}

			return
		}

		http.NotFound(w, req)
	}))

	return nil
}

func staticRun(c *cli.Command) (err error) {
	cfg, err := loadConfig(c.String("config"))
	if err != nil {
		return errors.Wrap(err, "load config")
	}

	root := c.String("output")
	root = filepath.Clean(root)

	var buf low.Buf

	for _, module := range cfg.Modules {
		for _, rep := range cfg.Replace {
			if !strings.HasPrefix(module, rep.Prefix) {
				continue
			}

			repo := strings.Replace(module, rep.Prefix, rep.Repo, 1)
			vcs := rep.VCS

			if vcs == "" {
				vcs = "git"
			}

			buf = buf[:0]

			err = repoPage.Execute(&buf, Module{
				Name: module,
				VCS:  vcs,
				Repo: repo,
			})
			if err != nil {
				return errors.Wrap(err, "exec page template")
			}

			domain := strings.IndexRune(module, '/')

			fname := filepath.FromSlash(module[domain+1:])
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

			break
		}
	}

	return nil
}

func loadConfig(name string) (*Config, error) {
	data, err := os.ReadFile(name)
	if err != nil {
		return nil, errors.Wrap(err, "read file")
	}

	var c Config

	err = yaml.Unmarshal(data, &c)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshal")
	}

	return &c, nil
}

var repoPage = template.Must(template.New("page").Parse(`<!DOCTYPE html>
{{- define "godoc" }}https://pkg.go.dev/{{ . }}{{ end }}
<html lang=en-US>
<head>
	<meta name="go-import" content="{{ .Name }} {{ .VCS }} {{ .Repo }}">
	<meta http-equiv="Refresh" content="3; url='{{ template "godoc" .Name }}'" />
</head>
<body>
	Redirecting to <a href="{{ template "godoc" .Name }}">{{ template "godoc" .Name }}</a>...
</body>
</html>
`))

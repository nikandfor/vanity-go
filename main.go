package main

import (
	"html/template"
	"net"
	"net/http"
	"os"

	"github.com/nikandfor/cli"
	"github.com/nikandfor/errors"
	"github.com/nikandfor/tlog"
	"github.com/nikandfor/tlog/ext/tlflag"
	"gopkg.in/yaml.v3"
)

type (
	Module struct {
		Name string `json:"name"` // import prefix
		VCS  string `json:"vcs"`
		Repo string `json:"repo"` // url
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
	}

	app := &cli.Command{
		Name:        "vanity",
		Description: "tool for making go vanity module names easy to use",
		Before:      before,
		Flags: []*cli.Flag{
			cli.NewFlag("modules", "", "file with modules"),

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
	mods, err := parseModules(c.String("modules"))
	if err != nil {
		return errors.Wrap(err, "load modules")
	}

	l, err := net.Listen("tcp", c.String("listen"))
	if err != nil {
		return errors.Wrap(err, "listen")
	}

	tlog.Printw("serving", "addr", l.Addr())

	err = http.Serve(l, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		tlog.Printw("request", "method", req.Method, "host", req.Host, "path", req.URL.Path, "query", req.URL.RawQuery)

		_ = mods
	}))

	return nil
}

func staticRun(c *cli.Command) (err error) {
	return nil
}

func parseModules(name string) ([]Module, error) {
	data, err := os.ReadFile(name)
	if err != nil {
		return nil, errors.Wrap(err, "read file")
	}

	var mods []Module

	err = yaml.Unmarshal(data, &mods)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshal")
	}

	for i := range mods {
		if mods[i].VCS == "" {
			mods[i].VCS = "git"
		}
	}

	return mods, nil
}

var repoPage = template.Must(template.New("page").Parse(`<!DOCTYPE html>
{{ define "godoc" }}https://pkg.go.dev/{{ . }}{{ end }}
<html lang=en-US>
<head>
	<meta name="go-import" content="{{ .Name }} {{ .VCS }} {{ .Repo }}">
	<meta http-equiv="Refresh" content="0; url='{{ template "godoc" .Name }}'" />
</head>
<body>
	<a href="{{ template "godoc" .Name }}">{{ .Name }}</a>
</body>
</html>
`))

package configdecoder

import (
	"bytes"
	"path"

	"github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/ast"
	"github.com/goccy/go-yaml/parser"
	"tlog.app/go/errors"
)

type (
	Decoder struct {
		yd *yaml.Decoder
	}

	configs []config

	config struct {
		Modules []module      `yaml:"modules"`
		Replace []Replacement `yaml:"replace"`
	}

	module []Module
	prefix []Module
)

func Decode(data []byte) ([]*Config, error) {
	var d Decoder

	return d.Decode(data)
}

func (d *Decoder) Decode(data []byte) ([]*Config, error) {
	d.yd = yaml.NewDecoder(bytes.NewReader(data), yaml.DisallowUnknownField(),
		yaml.CustomUnmarshaler(d.unmarshalPrefix),
	)

	var cs configs

	err := d.yd.Decode(&cs)
	if err != nil {
		return nil, errors.Wrap(err, "parse config")
	}

	cfgs := make([]*Config, len(cs))

	for i, c := range cs {
		//	for _, m := range c.Modules {
		//		log.Printf("module: %+v", m)
		//	}

		ms := make([]Module, 0, len(c.Modules))

		for _, m := range c.Modules {
			ms = append(ms, m...)
		}

		cfgs[i] = &Config{
			Modules: ms,
			Replace: c.Replace,
		}
	}

	return cfgs, nil
}

func (x *configs) UnmarshalYAML(f func(x interface{}) error) error {
	var cs []config

	err := f(&cs)
	if err == nil {
		*x = cs
		return nil
	}

	var c config

	err = f(&c)
	if err == nil {
		*x = []config{c}
		return nil
	}

	return err
}

func (x *module) UnmarshalYAML(f func(x interface{}) error) (err error) {
	var s string

	err = f(&s)
	if err == nil {
		*x = []Module{{Module: s}}
		return nil
	}

	var m Module

	err = f(&m)
	if err == nil {
		*x = []Module{m}
		return nil
	}

	var p prefix

	err = f(&p)
	if err == nil {
		*x = []Module(p)
		return nil
	}

	return err
}

func (d *Decoder) unmarshalPrefix(x *prefix, data []byte) (err error) {
	f, err := parser.ParseBytes(data, 0)
	if err != nil {
		return err
	}

	var prefix string
	var sub []string
	var common Module

	switch val := f.Docs[0].Body.(type) {
	case *ast.MappingNode:
		for _, val := range val.Values {
			key, err := d.getString(val.Key, "mapping key must be a string")
			if err != nil {
				return err
			}

			switch key {
			// module must be parsed before
			case "root":
				common.Root, err = d.getString(val.Value, "root must be a string")
			case "url":
				common.URL, err = d.getString(val.Value, "url must be a string")
			case "vcs":
				common.VCS, err = d.getString(val.Value, "vcs must be a string")
			default:
				if prefix != "" {
					return errors.New("more than one module prefix found: %v and %v", key, prefix)
				}

				prefix = key

				err = d.yd.DecodeFromNode(val.Value, &sub)
			}
			if err != nil {
				return err
			}
		}
	case *ast.MappingValueNode:
		prefix, err = d.getString(val.Key, "mapping key must be a string")
		if err != nil {
			return err
		}

		err = d.yd.DecodeFromNode(val.Value, &sub)
		if err != nil {
			return err
		}
	default:
		return errors.New("expected mapping, got %v", f.Docs[0].Body.Type())
	}

	//	log.Printf("prefix: %s, mod: %+v, sub: %+v", prefix, common, sub)

	ms := make([]Module, len(sub))

	for i, suff := range sub {
		ms[i] = Module{
			Module: path.Join(prefix, suff),
			Root:   first(common.Root, prefix),
			URL:    common.URL,
			VCS:    common.VCS,
		}
	}

	//	log.Printf("prefixed modules: %+v", ms)

	*x = ms

	return nil
}

func (d *Decoder) getString(n ast.Node, errtext string) (string, error) {
	s, ok := n.(*ast.StringNode)
	if !ok {
		return "", errors.New(errtext)
	}

	return s.Value, nil
}

func first(s ...string) string {
	for _, s := range s {
		if s != "" {
			return s
		}
	}

	return ""
}

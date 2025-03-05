package configdecoder

type (
	Config struct {
		Modules []Module      `json:"modules,omitempty"`
		Replace []Replacement `json:"replace,omitempty"`
	}

	Module struct {
		Module string `json:"module"`
		Root   string `json:"root,omitempty"`
		URL    string `json:"url,omitempty"`
		VCS    string `json:"vcs,omitempty"`
	}

	Replacement struct {
		Prefix string `json:"prefix"`
		URL    string `json:"url"`
		VCS    string `json:"vcs,omitempty"`
	}
)

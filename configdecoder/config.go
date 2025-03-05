package configdecoder

type (
	Config struct {
		Modules []Module      `json:"modules,omitempty"`
		Replace []Replacement `json:"replace,omitempty"`
	}

	Module struct {
		Module   string `json:"module"`
		RepoRoot string `json:"repo_root,omitempty"`
		Repo     string `json:"repo,omitempty"`
		VCS      string `json:"vcs,omitempty"`
	}

	Replacement struct {
		Prefix string `json:"prefix"`
		URL    string `json:"url"`
		VCS    string `json:"vcs,omitempty"`
	}
)

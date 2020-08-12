package types

type Files struct {
	URL         string `json:"url"`
	Name        string `json:"name"`
	Path        string `json:"path"`
	Checksum    string `json:"checksum"`
	Specificity string `json:"specificity"`
}

type Metadata struct {
	Name            string            `json:"name"`
	Description     string            `json:"description"`
	LongDescription string            `json:"long_description"`
	Maintainer      string            `json:"maintainer"`
	MaintainerEmail string            `json:"maintainer_email"`
	License         string            `json:"license"`
	Platforms       map[string]string `json:"platforms"`
	Dependencies    map[string]string `json:"dependencies"`
	Recommendations map[string]string `json:"recommendations"`
	Suggestions     map[string]string `json:"suggestions"`
	Conflicting     map[string]string `json:"conflicting"`
	Providing       map[string]string `json:"providing"`
	Replacing       map[string]string `json:"replacing"`
	Attributes      map[string]string `json:"attributes"`
	Groupings       map[string]string `json:"groupings"`
	Recipes         map[string]string `json:"recipes"`
	Version         string            `json:"version"`
	SourceURL       string            `json:"source_url"`
	IssuesURL       string            `json:"issues_url"`
	Privacy         bool              `json:"privacy"`
	ChefVersions    [][]string        `json:"chef_versions"`
	OhaiVersions    [][]string        `json:"ohai_versions"`
	Gems            [][]interface{}   `json:"gems"`
}

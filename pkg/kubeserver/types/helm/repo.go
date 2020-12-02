package helm

const (
	YUNION_REPO_NAME         = "infra"
	YUNION_REPO_HIDE_KEYWORD = "hide"
)

type Repo struct {
	Name   string `json:"name"`
	Url    string `json:"url"`
	Source string `json:"source"`
}

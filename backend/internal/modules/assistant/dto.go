package assistant

type Request struct {
	Skill   string `json:"skill,omitempty"`
	Message string `json:"message"`
}

type Source struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

type Answer struct {
	Skill   string   `json:"skill"`
	Mode    string   `json:"mode"`
	Answer  string   `json:"answer"`
	Sources []Source `json:"sources"`
}

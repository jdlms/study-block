package model

type Subject struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

var palette = []string{
	"#7c6ef0",
	"#f07c6e",
	"#6ef0a2",
	"#f0d86e",
	"#60a5fa",
	"#f472b6",
	"#34d399",
	"#fb923c",
}

func SubjectsFromNames(names []string) []Subject {
	out := make([]Subject, 0, len(names))
	for i, name := range names {
		out = append(out, Subject{Name: name, Color: palette[i%len(palette)]})
	}
	return out
}

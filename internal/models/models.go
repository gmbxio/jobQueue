package models

// Model describes a Groq model exposed by the application.
type Model struct {
	ID           string `json:"id"`
	Label        string `json:"label"`
	Availability string `json:"availability"`
}

var Catalog = []Model{
	{
		ID:           "openai/gpt-oss-120b",
		Label:        "GPT OSS 120B",
		Availability: "Production",
	},
	{
		ID:           "openai/gpt-oss-20b",
		Label:        "GPT OSS 20B",
		Availability: "Production",
	},
	{
		ID:           "qwen/qwen3.8-27b",
		Label:        "Qwen 3.8 27B",
		Availability: "Preview",
	},
	{
		ID:           "openai/gpt-oss-safeguard-20b",
		Label:        "GPT OSS Safeguard 20B",
		Availability: "Preview · Safety-focused",
	},
}

func IsSupported(id string) bool {
	for _, model := range Catalog {
		if model.ID == id {
			return true
		}
	}
	return false
}

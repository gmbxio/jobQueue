package models

// Model describes a Groq model exposed by the application.
type Model struct {
	ID           string `json:"id"`
	Label        string `json:"label"`
	Availability string `json:"availability"`

	// Per-model Groq request settings (not sent to the frontend).
	ReasoningEffort     string `json:"-"`
	MaxCompletionTokens int    `json:"-"`
}

var Catalog = []Model{
	{
		ID:              "openai/gpt-oss-20b",
		Label:           "GPT-OSS 20B (Ultra-Fast)",
		Availability:    "Available",
		ReasoningEffort: "low",
	},
	{
		ID:           "openai/gpt-oss-120b",
		Label:        "GPT-OSS 120B (Fast)",
		Availability: "Available",
	},
	{
		ID:                  "qwen/qwen3.8-27b",
		Label:               "Qwen 3.8 27B (Moderate)",
		Availability:        "Available",
		ReasoningEffort:     "medium",
		MaxCompletionTokens: 800, // free-tier OTPM limit is 1000
	},
}

func Get(id string) (Model, bool) {
	for _, model := range Catalog {
		if model.ID == id {
			return model, true
		}
	}
	return Model{}, false
}

func IsSupported(id string) bool {
	_, ok := Get(id)
	return ok
}
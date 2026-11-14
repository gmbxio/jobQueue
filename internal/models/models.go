package models

// Model describes a Groq model exposed by the application.
type Model struct {
	ID           string `json:"id"`
	Label        string `json:"label"`
	Availability string `json:"availability"`
}

var Catalog = []Model{
	{
		ID:           "openai/gpt-oss-20b",
		Label:        "GPT-OSS 20B (Ultra-Fast)",
		Availability: "Available",
	},
	{
		ID:           "openai/gpt-oss-120b",
		Label:        "GPT-OSS 120B (Fast)",
		Availability: "Available",
	},
	{
		ID:           "qwen/qwen3.8-27b",
		Label:        "Qwen 3.8 27B (Moderate)",
		Availability: "Available",
	},
	{
		ID:           "minimaxai/minimax-m2.7",
		Label:        "MiniMax M2.7 (Deep Thinking)",
		Availability: "Available",
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
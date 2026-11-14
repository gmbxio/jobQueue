package models

// Model describes a Groq model exposed by the application.
type Model struct {
	ID           string `json:"id"`
	Label        string `json:"label"`
	Availability string `json:"availability"`
}

var Catalog = []Model{
	{
		ID:           "llama-3.3-70b-versatile",
		Label:        "Llama 3.3 70B Versatile",
		Availability: "Production",
	},
	{
		ID:           "llama-3.1-8b-instant",
		Label:        "Llama 3.1 8B Instant",
		Availability: "Production",
	},
	{
		ID:           "llama3-8b-8192",
		Label:        "Llama 3 8B",
		Availability: "Production",
	},
	{
		ID:           "mixtral-8x7b-32768",
		Label:        "Mixtral 8x7B",
		Availability: "Production",
	},
	{
		ID:           "gemma2-9b-it",
		Label:        "Gemma 2 9B IT",
		Availability: "Production",
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

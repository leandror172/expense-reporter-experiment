package review

type ReviewData struct {
	Source      string       `json:"source"`
	GeneratedAt string       `json:"generatedAt"`
	Queue       []QueueEntry `json:"queue"`
	Taxonomy    Taxonomy     `json:"taxonomy"`
}

type QueueEntry struct {
	ID           string    `json:"id"`
	Item         string    `json:"item"`
	Date         string    `json:"date"`
	RawValue     string    `json:"rawValue"`
	Value        float64   `json:"value"`
	Confidence   float64   `json:"confidence"`
	AutoInserted bool      `json:"autoInserted"`
	Predicted    Predicted `json:"predicted"`
}

type Predicted struct {
	Sheet       string `json:"sheet,omitempty"`
	Category    string `json:"category"`
	Subcategory string `json:"subcategory"`
}

type Taxonomy struct {
	Sheets []Sheet `json:"sheets"`
}

type Sheet struct {
	Name       string     `json:"name"`
	Categories []Category `json:"categories"`
}

type Category struct {
	Name          string   `json:"name"`
	Subcategories []string `json:"subcategories"`
}

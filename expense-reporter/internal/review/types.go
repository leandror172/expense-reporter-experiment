package review

type ReviewData struct {
	Source      string       `json:"source"`
	GeneratedAt string       `json:"generatedAt"`
	Queue       []QueueEntry `json:"queue"`
	Taxonomy    Taxonomy     `json:"taxonomy"`
}

type QueueEntry struct {
	ID       string  `json:"id"`
	Item     string  `json:"item"`
	Date     string  `json:"date"`
	RawValue string  `json:"rawValue"`
	Value    float64 `json:"value"`
	// Installments is the count parsed out of RawValue ("100,00/3" → 3), 1 for an
	// ordinary purchase. It rides through to reviewed.json so apply can expand the
	// series; discarding it here is what made apply record a 3× purchase as one row
	// while batch-auto's auto route expanded it correctly (T-21).
	Installments int       `json:"installments"`
	Confidence   float64   `json:"confidence"`
	AutoInserted bool      `json:"autoInserted"`
	Predicted    Predicted `json:"predicted"`
}

type Predicted struct {
	Type        string `json:"type,omitempty"`
	Category    string `json:"category"`
	Subcategory string `json:"subcategory"`
}

type Taxonomy struct {
	Types []Type `json:"types"`
}

type Type struct {
	Name       string     `json:"name"`
	Categories []Category `json:"categories"`
}

type Category struct {
	Name          string   `json:"name"`
	Subcategories []string `json:"subcategories"`
}

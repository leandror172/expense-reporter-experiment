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
	Installments int     `json:"installments"`
	Confidence   float64 `json:"confidence"`
	AutoInserted bool    `json:"autoInserted"`
	// KeywordHint is the keyword layer's competing suggestion, present only when it
	// unambiguously disagreed with the model. ADVISORY — the page shows it beside the
	// model's answer so the reviewer can see a second opinion; nothing selects it
	// automatically, and it is deliberately NOT carried back out through
	// exportReviewed(). It is input to the human, not part of their decision.
	KeywordHint string `json:"keywordHint,omitempty"`
	// AlreadyLogged says why this row's id is already in the expense log — empty when it is
	// not. Two values, because they ask the reviewer DIFFERENT questions: a fully-logged row
	// needs a ruling on whether this is a genuinely separate purchase (same-day duplicates
	// hash alike by construction), while a partially-logged series is the only one of the two
	// that can be legitimately completed. A bool could not tell them apart.
	//
	// ADVISORY, exactly like KeywordHint above: the page renders it as a badge, nothing acts
	// on it automatically, and it is deliberately NOT carried back out through
	// exportReviewed(). It is input to the human, not part of their decision.
	AlreadyLogged string    `json:"alreadyLogged,omitempty"`
	Predicted     Predicted `json:"predicted"`
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

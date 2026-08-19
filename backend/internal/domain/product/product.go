package product

type Type string

const (
	TypeClass Type = "class"
	TypeCombo Type = "combo"
)

type Activity struct {
	ID              string `json:"id"`
	Title           string `json:"title"`
	Slug            string `json:"slug"`
	Instructor      string `json:"instructor,omitempty"`
	DurationMinutes int    `json:"durationMinutes,omitempty"`
	Description     string `json:"description,omitempty"`
}

type Product struct {
	ID                string `json:"id"`
	Title             string `json:"title"`
	Slug              string `json:"slug"`
	Description       string `json:"description,omitempty"`
	Type              Type   `json:"type"`
	IncludesBreakfast bool   `json:"includesBreakfast"`
	PriceCents        int    `json:"priceCents"`
	Featured          bool   `json:"featured"`
	// ChooseOneActivity means the customer picks exactly one of Activities (e.g. "Yoga
	// ou HYROX") instead of booking a session for every activity listed.
	ChooseOneActivity bool       `json:"chooseOneActivity"`
	Activities        []Activity `json:"activities"`
}

package entities

type Source string

const (
	SourceHH       Source = "hh"
	SourceZarplata Source = "zarplata"
)

type Vacancy struct {
	Source       Source
	ExternalID   string
	URL          string
	Title        string
	Company      string
	Requirements string
}

type SearchQuery struct {
	Source Source
	Text   string
	Area   string // опционально: регион
}

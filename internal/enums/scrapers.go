package enums

type ScraperType int

const (
	_ ScraperType = iota
	NeoAuto
)

var ScraperTypeNames = map[ScraperType]string{
	NeoAuto: "NeoAuto",
}

func (s ScraperType) String() string {
	return ScraperTypeNames[s]
}
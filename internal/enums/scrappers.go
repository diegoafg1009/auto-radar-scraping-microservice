package enums

type ScrapperType int

const (
	_ ScrapperType = iota
	NeoAuto
)

var ScrapperTypeNames = map[ScrapperType]string{
	NeoAuto: "NeoAuto",
}

func (s ScrapperType) String() string {
	return ScrapperTypeNames[s]
}
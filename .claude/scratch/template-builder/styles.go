package main

import "github.com/xuri/excelize/v2"

// styleSet holds every registered excelize style ID used by the builder.
type styleSet struct {
	MesCorner     int // A1:B2 "Mês": C0C0C0, Open Sans 14 bold, center/center
	MonthBanner   int // month-name anchor: C0C0C0, Open Sans 14 bold, center (h)
	MonthCovered  int // cells under a month merge: C0C0C0, no font emphasis
	HeaderCol     int // row 2 Item/Data/Valor: D8D8D8, Open Sans 14, box border, center
	CategoriaBold int // merged col-A label: Arial 14 bold, center/center, wrap
	SubcatLabel   int // merged col-B label: Arial 10, center/center, wrap
	DataCellArial int
	DataCellTop   int
	TotalData     int
	TotalDataL    int // total item col with left border (group separator, k>=1)
	TotalValor    int
	TotalValorR   int // total valor col with right border (group separator, k>=1)
	Separator     int
	Currency      int
	DateCell      int

	// Listas-specific
	SectionLabel   int // merged col-A section: 333399 navy, Arial 18 bold white, center/center/wrap
	IndigoBand     int // 333399 band cell, white bold (no numfmt)
	IndigoLabel    int // 333399 merged categoria label, white bold, center/center/wrap
	IndigoBandCur  int // 333399 band cell, white bold, currency
	GroupTotalLbl  int // CCCCFF, Arial 10, General (B/C label cells)
	GroupTotalCur  int // CCCCFF, Arial 10, currency
	GroupTotalPct  int // CCCCFF, Arial 10, percent
	ListasTotalLbl int // C0C0C0, Arial 10 bold, General (B/C label cells)
	ListasTotalCur int // C0C0C0, Arial 10 bold, currency
	ListasTotalPct int // C0C0C0, Arial 10 bold, percent
	NearBlack      int // 333333, Arial 10 bold white (label)
	NearBlackCur   int // 333333, Arial 10 bold white, currency
	NearBlackPct   int // 333333, Arial 10 bold white, percent
	PullCur        int // plain currency Arial 10 (pull rows D..O)
	ListasMonth    int // C0C0C0 month header on Listas row 3
}

const (
	fmtCurrency = "R$ #,##0.00"
	fmtDate     = "DD/MM"
	fmtPercent  = "0.00%"
)

func solidFill(hex string) excelize.Fill {
	return excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{hex}}
}

func boxBorder() []excelize.Border {
	sides := []string{"left", "top", "right", "bottom"}
	bs := make([]excelize.Border, len(sides))
	for i, s := range sides {
		bs[i] = excelize.Border{Type: s, Color: "000000", Style: 1}
	}
	return bs
}

func topBottomBorder() []excelize.Border {
	return []excelize.Border{
		{Type: "top", Color: "000000", Style: 1},
		{Type: "bottom", Color: "000000", Style: 1},
	}
}

func centerBoth(wrap bool) *excelize.Alignment {
	return &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: wrap}
}

func alignH(h string) *excelize.Alignment { return &excelize.Alignment{Horizontal: h} }

func newStyles(f *excelize.File) (*styleSet, error) {
	s := &styleSet{}
	var firstErr error
	mk := func(st *excelize.Style) int {
		id, err := f.NewStyle(st)
		if err != nil && firstErr == nil {
			firstErr = err
		}
		return id
	}

	openSans := func(bold bool) *excelize.Font {
		return &excelize.Font{Family: "Open Sans", Size: 14, Bold: bold}
	}
	arial := func(bold bool) *excelize.Font { return &excelize.Font{Family: "Arial", Size: 10, Bold: bold} }
	arialN := func(size float64, bold bool) *excelize.Font {
		return &excelize.Font{Family: "Arial", Size: size, Bold: bold}
	}
	arialWhite := func(size float64, bold bool) *excelize.Font {
		return &excelize.Font{Family: "Arial", Size: size, Bold: bold, Color: "FFFFFF"}
	}
	cur, date, pct := fmtCurrency, fmtDate, fmtPercent

	topLeft := []excelize.Border{
		{Type: "top", Color: "000000", Style: 1},
		{Type: "left", Color: "000000", Style: 1},
	}
	s.MesCorner = mk(&excelize.Style{Fill: solidFill("C0C0C0"), Font: openSans(true), Border: topLeft, Alignment: centerBoth(false)})
	s.MonthBanner = mk(&excelize.Style{Fill: solidFill("C0C0C0"), Font: openSans(true), Border: topLeft, Alignment: alignH("center")})
	s.HeaderCol = mk(&excelize.Style{Fill: solidFill("D8D8D8"), Font: openSans(false), Border: boxBorder()})
	s.CategoriaBold = mk(&excelize.Style{Font: arialN(14, true), Alignment: centerBoth(true)})
	s.SubcatLabel = mk(&excelize.Style{Font: arial(false), Alignment: centerBoth(true)})
	s.DataCellArial = mk(&excelize.Style{Font: arial(false)})
	s.DataCellTop = mk(&excelize.Style{Font: arial(false), Border: []excelize.Border{{Type: "top", Color: "000000", Style: 1}}})
	tb := topBottomBorder()
	tbl := append(topBottomBorder(), excelize.Border{Type: "left", Color: "000000", Style: 1})
	tbr := append(topBottomBorder(), excelize.Border{Type: "right", Color: "000000", Style: 1})
	s.TotalData = mk(&excelize.Style{Fill: solidFill("F2F2F2"), Font: arial(false), Border: tb})
	s.TotalDataL = mk(&excelize.Style{Fill: solidFill("F2F2F2"), Font: arial(false), Border: tbl})
	s.TotalValor = mk(&excelize.Style{Fill: solidFill("F2F2F2"), Font: arial(false), Border: tb, CustomNumFmt: &cur})
	s.TotalValorR = mk(&excelize.Style{Fill: solidFill("F2F2F2"), Font: arial(false), Border: tbr, CustomNumFmt: &cur})
	s.Separator = mk(&excelize.Style{Fill: solidFill("000000")})
	s.Currency = mk(&excelize.Style{Font: arial(false), CustomNumFmt: &cur})
	s.DateCell = mk(&excelize.Style{Font: arial(false), CustomNumFmt: &date})

	s.SectionLabel = mk(&excelize.Style{Fill: solidFill("333399"), Font: arialWhite(18, true), Alignment: centerBoth(true)})
	s.IndigoBand = mk(&excelize.Style{Fill: solidFill("333399"), Font: arialWhite(10, true)})
	s.IndigoLabel = mk(&excelize.Style{Fill: solidFill("333399"), Font: arialWhite(10, true), Alignment: centerBoth(true)})
	s.IndigoBandCur = mk(&excelize.Style{Fill: solidFill("333399"), Font: arialWhite(10, true), CustomNumFmt: &cur})
	s.GroupTotalLbl = mk(&excelize.Style{Fill: solidFill("CCCCFF"), Font: arial(false)})
	s.GroupTotalCur = mk(&excelize.Style{Fill: solidFill("CCCCFF"), Font: arial(false), CustomNumFmt: &cur})
	s.GroupTotalPct = mk(&excelize.Style{Fill: solidFill("CCCCFF"), Font: arial(false), CustomNumFmt: &pct})
	s.ListasTotalLbl = mk(&excelize.Style{Fill: solidFill("C0C0C0"), Font: arial(true)})
	s.ListasTotalCur = mk(&excelize.Style{Fill: solidFill("C0C0C0"), Font: arial(true), CustomNumFmt: &cur})
	s.ListasTotalPct = mk(&excelize.Style{Fill: solidFill("C0C0C0"), Font: arial(true), CustomNumFmt: &pct})
	s.NearBlack = mk(&excelize.Style{Fill: solidFill("333333"), Font: arialWhite(10, true)})
	s.NearBlackCur = mk(&excelize.Style{Fill: solidFill("333333"), Font: arialWhite(10, true), CustomNumFmt: &cur})
	s.NearBlackPct = mk(&excelize.Style{Fill: solidFill("333333"), Font: arialWhite(10, true), CustomNumFmt: &pct})
	s.PullCur = mk(&excelize.Style{Font: arial(false), CustomNumFmt: &cur})
	s.ListasMonth = mk(&excelize.Style{Fill: solidFill("C0C0C0"), Font: openSans(false)})

	return s, firstErr
}

package generate

import "github.com/xuri/excelize/v2"

// styleSet holds every registered excelize style ID used by the builder.
type styleSet struct {
	// Data-sheet header styles
	MonthCorner int // A1:B2 "Mês": C0C0C0, Open Sans 14 bold, center/center
	MonthBanner int // month-name anchor: C0C0C0, Open Sans 14 bold, center (h)
	HeaderCol   int // row 2 Item/Data/Valor: D8D8D8, Open Sans 14, box border, center

	// Data-sheet body styles
	CategoryBold  int // merged col-A label: Arial 14 bold, center/center, wrap
	SubcatLabel   int // merged col-B label: Arial 10, center/center, wrap
	DataCellArial int
	DataCellTop   int
	Currency      int
	DateCell      int
	Separator     int

	// Data-sheet total-row styles (F2F2F2). "Text" covers the non-currency
	// cells of the total row (item + date columns); Left/Right carry the
	// every-other-month group-separator border.
	TotalText       int
	TotalTextLeft   int
	TotalValue      int
	TotalValueRight int

	// Summary-sheet (Listas) styles
	SectionLabel int // merged col-A section: 333399 navy, Arial 18 bold white, center/center/wrap
	IndigoBand   int // 333399 band cell, white bold (no numfmt)
	IndigoLabel  int // 333399 merged categoria label, white bold, center/center/wrap
	PullCur      int // plain currency Arial 10 (pull rows D..O); same definition as Currency
	SummaryMonth int // C0C0C0 month header on Listas row 3

	// Style families: one fill+font in General / currency / percent variants.
	GroupTotalLbl   int // CCCCFF, Arial 10 (B/C label cells)
	GroupTotalCur   int
	GroupTotalPct   int
	SummaryTotalLbl int // C0C0C0, Arial 10 bold (B/C label cells)
	SummaryTotalCur int
	SummaryTotalPct int
	NearBlack       int // 333333, Arial 10 bold white (label)
	NearBlackCur    int
	NearBlackPct    int
}

// Number formats. The empty string means General (no CustomNumFmt).
const (
	fmtGeneral  = ""
	fmtCurrency = "R$ #,##0.00"
	fmtDate     = "DD/MM"
	fmtPercent  = "0.00%"
)

// Palette: the workbook's fill colors, named for their role.
const (
	fillHeaderGray   = "C0C0C0" // month banners, summary totals
	fillColumnGray   = "D8D8D8" // Item/Data/Valor column headers
	fillTotalGray    = "F2F2F2" // data-sheet total rows
	fillNavy         = "333399" // summary section bands and labels
	fillLavender     = "CCCCFF" // summary group totals
	fillNearBlack    = "333333" // summary balance-block emphasis rows
	fillBlack        = "000000" // separator rows
	borderColorBlack = "000000"
)

func solidFill(hex string) excelize.Fill {
	return excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{hex}}
}

func border(sides ...string) []excelize.Border {
	bs := make([]excelize.Border, len(sides))
	for i, s := range sides {
		bs[i] = excelize.Border{Type: s, Color: borderColorBlack, Style: 1}
	}
	return bs
}

func centerBoth(wrap bool) *excelize.Alignment {
	return &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: wrap}
}

func alignH(h string) *excelize.Alignment { return &excelize.Alignment{Horizontal: h} }

// Font constructors shared by all style definitions.
func openSans(bold bool) *excelize.Font {
	return &excelize.Font{Family: "Open Sans", Size: 14, Bold: bold}
}

func arial(bold bool) *excelize.Font {
	return &excelize.Font{Family: "Arial", Size: 10, Bold: bold}
}

func arialSized(size float64, bold bool) *excelize.Font {
	return &excelize.Font{Family: "Arial", Size: size, Bold: bold}
}

func arialWhite(size float64, bold bool) *excelize.Font {
	return &excelize.Font{Family: "Arial", Size: size, Bold: bold, Color: "FFFFFF"}
}

// ── Style vocabulary ─────────────────────────────────────────────────────────
// Each constructor names WHAT a cell is in the workbook's visual language;
// the excelize mechanics stay inside.

// numFmt attaches a number format to a style; fmtGeneral leaves it unset.
func numFmt(st *excelize.Style, format string) *excelize.Style {
	if format != fmtGeneral {
		f := format
		st.CustomNumFmt = &f
	}
	return st
}

// dataCell: a bare Arial-10 body cell, formatted per its content type.
func dataCell(format string) *excelize.Style {
	return numFmt(&excelize.Style{Font: arial(false)}, format)
}

// dataCellTopBordered: the first data row of a block — bare cell + top rule.
func dataCellTopBordered() *excelize.Style {
	return &excelize.Style{Font: arial(false), Border: border("top")}
}

// centeredLabel: a merged label cell — centered both ways, wrapped.
func centeredLabel(font *excelize.Font) *excelize.Style {
	return &excelize.Style{Font: font, Alignment: centerBoth(true)}
}

// grayBanner: a C0C0C0 Open Sans 14 bold header cell (top-left rule).
func grayBanner(align *excelize.Alignment) *excelize.Style {
	return &excelize.Style{Fill: solidFill(fillHeaderGray), Font: openSans(true), Border: border("top", "left"), Alignment: align}
}

// columnHeader: the D8D8D8 boxed Item/Data/Valor header cell.
func columnHeader() *excelize.Style {
	return &excelize.Style{Fill: solidFill(fillColumnGray), Font: openSans(false), Border: border("left", "top", "right", "bottom")}
}

// totalRowCell: an F2F2F2 total-row cell, ruled top+bottom; extraSide adds the
// group-separator vertical rule ("" for none).
func totalRowCell(extraSide, format string) *excelize.Style {
	sides := []string{"top", "bottom"}
	if extraSide != "" {
		sides = append(sides, extraSide)
	}
	return numFmt(&excelize.Style{Fill: solidFill(fillTotalGray), Font: arial(false), Border: border(sides...)}, format)
}

// navyBand: a 333399 white-bold summary band cell, optionally aligned.
func navyBand(font *excelize.Font, align *excelize.Alignment) *excelize.Style {
	return &excelize.Style{Fill: solidFill(fillNavy), Font: font, Alignment: align}
}

// fillOnly: a cell that is pure color (separator rows).
func fillOnly(hex string) *excelize.Style {
	return &excelize.Style{Fill: solidFill(hex)}
}

// ── Registration ─────────────────────────────────────────────────────────────

// styleRegistrar wraps excelize style registration, capturing the first error
// so style construction can read declaratively instead of err-checking per style.
type styleRegistrar struct {
	f        *excelize.File
	firstErr error
}

func (r *styleRegistrar) add(st *excelize.Style) int {
	id, err := r.f.NewStyle(st)
	if err != nil && r.firstErr == nil {
		r.firstErr = err
	}
	return id
}

// family registers one fill+font combination in its three number-format
// variants: General (labels), currency, and percent.
func (r *styleRegistrar) family(fillHex string, font *excelize.Font) (lbl, cur, pct int) {
	base := func(format string) *excelize.Style {
		return numFmt(&excelize.Style{Fill: solidFill(fillHex), Font: font}, format)
	}
	return r.add(base(fmtGeneral)), r.add(base(fmtCurrency)), r.add(base(fmtPercent))
}

func newStyles(f *excelize.File) (*styleSet, error) {
	r := &styleRegistrar{f: f}
	s := &styleSet{}

	registerHeaderStyles(r, s)
	registerDataBodyStyles(r, s)
	registerTotalRowStyles(r, s)
	registerSummaryStyles(r, s)

	s.GroupTotalLbl, s.GroupTotalCur, s.GroupTotalPct = r.family(fillLavender, arial(false))
	s.SummaryTotalLbl, s.SummaryTotalCur, s.SummaryTotalPct = r.family(fillHeaderGray, arial(true))
	s.NearBlack, s.NearBlackCur, s.NearBlackPct = r.family(fillNearBlack, arialWhite(10, true))

	return s, r.firstErr
}

func registerHeaderStyles(r *styleRegistrar, s *styleSet) {
	s.MonthCorner = r.add(grayBanner(centerBoth(false)))
	s.MonthBanner = r.add(grayBanner(alignH("center")))
	s.HeaderCol = r.add(columnHeader())
}

func registerDataBodyStyles(r *styleRegistrar, s *styleSet) {
	s.CategoryBold = r.add(centeredLabel(arialSized(14, true)))
	s.SubcatLabel = r.add(centeredLabel(arial(false)))
	s.DataCellArial = r.add(dataCell(fmtGeneral))
	s.DataCellTop = r.add(dataCellTopBordered())
	s.Currency = r.add(dataCell(fmtCurrency))
	s.DateCell = r.add(dataCell(fmtDate))
	s.Separator = r.add(fillOnly(fillBlack))
}

func registerTotalRowStyles(r *styleRegistrar, s *styleSet) {
	s.TotalText = r.add(totalRowCell("", fmtGeneral))
	s.TotalTextLeft = r.add(totalRowCell("left", fmtGeneral))
	s.TotalValue = r.add(totalRowCell("", fmtCurrency))
	s.TotalValueRight = r.add(totalRowCell("right", fmtCurrency))
}

func registerSummaryStyles(r *styleRegistrar, s *styleSet) {
	s.SectionLabel = r.add(navyBand(arialWhite(18, true), centerBoth(true)))
	s.IndigoBand = r.add(navyBand(arialWhite(10, true), nil))
	s.IndigoLabel = r.add(navyBand(arialWhite(10, true), centerBoth(true)))
	s.PullCur = r.add(dataCell(fmtCurrency)) // intentionally identical to Currency; kept as its own name for summary-sheet readability
	s.SummaryMonth = r.add(&excelize.Style{Fill: solidFill(fillHeaderGray), Font: openSans(false)})
}

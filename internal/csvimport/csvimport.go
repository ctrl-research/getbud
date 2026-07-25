// Package csvimport parses bank/credit-card CSV exports into normalized
// transaction rows. It is pure (no HTTP, no database): the server hands it a
// file and a column mapping, it hands back rows or per-row errors.
package csvimport

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"
)

// File is a parsed CSV: a possible header row plus data records.
type File struct {
	// Headers is the first row when it looks like column names, nil
	// otherwise (some banks export headerless CSVs).
	Headers []string
	// Records are the data rows (header excluded).
	Records [][]string
	// Delimiter that was detected (',', ';', or '\t').
	Delimiter rune
}

// AmountMode says how the amount is encoded.
type AmountMode string

const (
	// AmountSigned is a single signed column ("-12.34" or "(12.34)").
	AmountSigned AmountMode = "signed"
	// AmountSplit is separate debit and credit columns.
	AmountSplit AmountMode = "split"
)

// Mapping describes which columns hold what. Column indexes are 0-based;
// -1 means "not mapped".
type Mapping struct {
	DateColumn   int        `json:"dateColumn"`
	DateFormat   string     `json:"dateFormat"`
	PayeeColumn  int        `json:"payeeColumn"`
	NotesColumn  int        `json:"notesColumn"`
	AmountMode   AmountMode `json:"amountMode"`
	AmountColumn int        `json:"amountColumn"`
	DebitColumn  int        `json:"debitColumn"`
	CreditColumn int        `json:"creditColumn"`
	// InvertSign flips signed amounts. Credit-card exports often report
	// purchases as positive numbers; inverting makes them expenses.
	InvertSign bool `json:"invertSign"`
}

// Row is one normalized transaction candidate.
type Row struct {
	// Index is the 0-based position within File.Records.
	Index       int
	Date        time.Time
	AmountCents int64
	Payee       string
	Notes       string
	// Err describes why the row could not be parsed ("" when OK).
	Err string
}

// DateFormats are the layouts Parse understands, in guess-priority order.
var DateFormats = []string{
	"2006-01-02",
	"2006/01/02",
	"01/02/2006",
	"02/01/2006",
	"1/2/2006", // non-padded month-first (Tangerine style)
	"2/1/2006", // non-padded day-first
	"01-02-2006",
	"02-01-2006",
	"Jan 2, 2006",
	"2 Jan 2006",
	"02-Jan-2006",
	"20060102",
}

// Parse reads a CSV export: strips a UTF-8 BOM, sniffs the delimiter, and
// splits off the header row when present.
func Parse(r io.Reader) (*File, error) {
	raw, err := io.ReadAll(io.LimitReader(r, 10<<20+1))
	if err != nil {
		return nil, err
	}
	if len(raw) > 10<<20 {
		return nil, errors.New("file too large (max 10 MB)")
	}
	raw = bytes.TrimPrefix(raw, []byte{0xEF, 0xBB, 0xBF})
	if len(bytes.TrimSpace(raw)) == 0 {
		return nil, errors.New("file is empty")
	}

	delim := sniffDelimiter(raw)
	cr := csv.NewReader(bytes.NewReader(raw))
	cr.Comma = delim
	cr.FieldsPerRecord = -1 // banks are sloppy; tolerate ragged rows
	cr.LazyQuotes = true

	var records [][]string
	for {
		rec, err := cr.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("parse CSV: %w", err)
		}
		// Skip fully-empty rows.
		empty := true
		for _, f := range rec {
			if strings.TrimSpace(f) != "" {
				empty = false
				break
			}
		}
		if !empty {
			records = append(records, rec)
		}
	}
	if len(records) == 0 {
		return nil, errors.New("no rows found")
	}

	f := &File{Records: records, Delimiter: delim}
	if looksLikeHeader(records[0]) {
		f.Headers = records[0]
		f.Records = records[1:]
	}
	if len(f.Records) == 0 {
		return nil, errors.New("no data rows found (header only)")
	}
	return f, nil
}

// sniffDelimiter picks the separator that appears most consistently in the
// first non-empty lines.
func sniffDelimiter(raw []byte) rune {
	lines := bytes.Split(raw, []byte("\n"))
	counts := map[rune]int{',': 0, ';': 0, '\t': 0}
	seen := 0
	for _, line := range lines {
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		for d := range counts {
			counts[d] += bytes.Count(line, []byte(string(d)))
		}
		if seen++; seen == 5 {
			break
		}
	}
	best, bestCount := ',', counts[',']
	for _, d := range []rune{';', '\t'} {
		if counts[d] > bestCount {
			best, bestCount = d, counts[d]
		}
	}
	return best
}

// looksLikeHeader reports whether a row reads as column names: no cell
// parses as a date or a money amount.
func looksLikeHeader(row []string) bool {
	for _, cell := range row {
		cell = strings.TrimSpace(cell)
		if cell == "" {
			continue
		}
		if _, ok := parseAnyDate(cell); ok {
			return false
		}
		if _, ok := parseAmount(cell); ok {
			return false
		}
	}
	return true
}

func parseAnyDate(s string) (string, bool) {
	for _, layout := range DateFormats {
		if _, err := time.Parse(layout, s); err == nil {
			return layout, true
		}
	}
	return "", false
}

var moneyRe = regexp.MustCompile(`^\(?-?\$?-?[\d,]+(\.\d{1,2})?\)?$`)

// parseAmount converts "1,234.56", "$-12", "(45.00)" into cents.
func parseAmount(s string) (int64, bool) {
	s = strings.TrimSpace(s)
	if s == "" || !moneyRe.MatchString(s) {
		return 0, false
	}
	negative := false
	if strings.HasPrefix(s, "(") && strings.HasSuffix(s, ")") {
		negative = true
		s = s[1 : len(s)-1]
	}
	s = strings.ReplaceAll(s, "$", "")
	s = strings.ReplaceAll(s, ",", "")
	if strings.HasPrefix(s, "-") {
		negative = !negative
		s = s[1:]
	}
	whole, frac, _ := strings.Cut(s, ".")
	if whole == "" && frac == "" {
		return 0, false
	}
	var cents int64
	for _, r := range whole {
		if r < '0' || r > '9' {
			return 0, false
		}
		cents = cents*10 + int64(r-'0')
	}
	cents *= 100
	frac = (frac + "00")[:2]
	for i, r := range frac {
		if r < '0' || r > '9' {
			return 0, false
		}
		if i == 0 {
			cents += int64(r-'0') * 10
		} else {
			cents += int64(r - '0')
		}
	}
	if negative {
		cents = -cents
	}
	return cents, true
}

// Normalize applies a mapping to every record, returning one Row per record
// (with Err set on rows that fail to parse).
func Normalize(f *File, m Mapping) []Row {
	rows := make([]Row, 0, len(f.Records))
	for i, rec := range f.Records {
		rows = append(rows, normalizeRow(i, rec, m))
	}
	return rows
}

func cellAt(rec []string, idx int) string {
	if idx < 0 || idx >= len(rec) {
		return ""
	}
	return strings.TrimSpace(rec[idx])
}

func normalizeRow(idx int, rec []string, m Mapping) Row {
	row := Row{Index: idx}

	dateStr := cellAt(rec, m.DateColumn)
	if dateStr == "" {
		row.Err = "missing date"
		return row
	}
	date, err := time.Parse(m.DateFormat, dateStr)
	if err != nil {
		row.Err = fmt.Sprintf("date %q does not match format", dateStr)
		return row
	}
	row.Date = date

	switch m.AmountMode {
	case AmountSplit:
		debit, debitOK := parseAmount(cellAt(rec, m.DebitColumn))
		credit, creditOK := parseAmount(cellAt(rec, m.CreditColumn))
		switch {
		case debitOK && debit != 0:
			row.AmountCents = -abs64(debit)
		case creditOK && credit != 0:
			row.AmountCents = abs64(credit)
		default:
			row.Err = "no amount in debit or credit column"
			return row
		}
	default: // AmountSigned
		amount, ok := parseAmount(cellAt(rec, m.AmountColumn))
		if !ok {
			row.Err = fmt.Sprintf("amount %q is not a number", cellAt(rec, m.AmountColumn))
			return row
		}
		if amount == 0 {
			row.Err = "amount is zero"
			return row
		}
		if m.InvertSign {
			amount = -amount
		}
		row.AmountCents = amount
	}

	row.Payee = cellAt(rec, m.PayeeColumn)
	row.Notes = cellAt(rec, m.NotesColumn)
	return row
}

func abs64(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}

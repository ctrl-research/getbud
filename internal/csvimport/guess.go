package csvimport

import (
	"strings"
	"time"
)

// GuessMapping infers a column mapping from headers and cell values. It is a
// best-effort starting point for the import wizard; the user confirms or
// fixes it before committing.
func GuessMapping(f *File) Mapping {
	m := Mapping{
		DateColumn: -1, PayeeColumn: -1, NotesColumn: -1,
		AmountColumn: -1, DebitColumn: -1, CreditColumn: -1,
		AmountMode: AmountSigned,
	}

	// Pass 1: header-name heuristics.
	for i, h := range f.Headers {
		switch name := strings.ToLower(strings.TrimSpace(h)); {
		case m.DateColumn == -1 && containsAny(name, "date", "posted"):
			m.DateColumn = i
		case m.PayeeColumn == -1 && containsAny(name, "description", "payee", "merchant", "name", "details", "transaction"):
			m.PayeeColumn = i
		case m.DebitColumn == -1 && containsAny(name, "debit", "withdrawal", "money out", "outflow"):
			m.DebitColumn = i
		case m.CreditColumn == -1 && containsAny(name, "credit", "deposit", "money in", "inflow"):
			m.CreditColumn = i
		case m.AmountColumn == -1 && containsAny(name, "amount", "value", "total"):
			m.AmountColumn = i
		case m.NotesColumn == -1 && containsAny(name, "memo", "note", "category", "reference"):
			m.NotesColumn = i
		}
	}

	// Pass 2: value-shape probing fills anything headers didn't.
	sample := f.Records
	if len(sample) > 50 {
		sample = sample[:50]
	}
	width := 0
	for _, rec := range sample {
		if len(rec) > width {
			width = len(rec)
		}
	}
	for col := 0; col < width; col++ {
		dates, amounts, texts := 0, 0, 0
		for _, rec := range sample {
			cell := cellAt(rec, col)
			if cell == "" {
				continue
			}
			if _, ok := parseAnyDate(cell); ok {
				dates++
			} else if _, ok := parseAmount(cell); ok {
				amounts++
			} else {
				texts++
			}
		}
		switch {
		case m.DateColumn == -1 && dates > 0 && dates >= amounts && dates >= texts:
			m.DateColumn = col
		case m.AmountColumn == -1 && m.DebitColumn == -1 && amounts > 0 && amounts > texts && col != m.DateColumn:
			m.AmountColumn = col
		case m.PayeeColumn == -1 && texts > 0 && col != m.DateColumn:
			m.PayeeColumn = col
		}
	}

	// Debit+credit found together wins over a single amount column.
	if m.DebitColumn != -1 && m.CreditColumn != -1 {
		m.AmountMode = AmountSplit
	}

	if m.DateColumn != -1 {
		m.DateFormat = guessDateFormat(f, m.DateColumn)
	}
	return m
}

func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

// guessDateFormat finds the layout every sampled value satisfies, using >12
// day values to break the MM/DD vs DD/MM tie when possible.
func guessDateFormat(f *File, col int) string {
	candidates := append([]string(nil), DateFormats...)
	sampled := 0
	for _, rec := range f.Records {
		cell := cellAt(rec, col)
		if cell == "" {
			continue
		}
		var still []string
		for _, layout := range candidates {
			if t, err := time.Parse(layout, cell); err == nil && t.Year() > 1900 {
				still = append(still, layout)
			}
		}
		if len(still) > 0 {
			candidates = still
		}
		if sampled++; sampled == 100 || len(candidates) == 1 {
			break
		}
	}
	if len(candidates) == 0 {
		return DateFormats[0]
	}
	// Ambiguity (e.g. all values <= 12th of the month) resolves to the
	// earlier entry in DateFormats — MM/DD, the common Canadian export.
	return candidates[0]
}

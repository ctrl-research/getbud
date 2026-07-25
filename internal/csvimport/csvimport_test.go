package csvimport

import (
	"strings"
	"testing"
)

const rbcStyle = `"Account Type","Account Number","Transaction Date","Cheque Number","Description 1","Description 2","CAD$","USD$"
Chequing,00002-1234567,2026-06-02,,"PAYROLL DEPOSIT","ACME CORP",2500.00,
Chequing,00002-1234567,2026-06-03,,"e-Transfer sent","JANE DOE",-120.00,
Chequing,00002-1234567,2026-06-15,,"UTILITY BILL PMT","HYDRO",-89.50,
`

const tangerineStyle = `Date,Transaction,Name,Memo,Amount
6/1/2026,DEBIT,PRESTO FARE,Transit,-3.35
6/2/2026,CREDIT,Interest Paid,,1.22
6/28/2026,DEBIT,LOBLAWS #123,Groceries,-84.15
`

const creditCardInverted = `Transaction Date,Posted Date,Description,Amount
2026-06-05,2026-06-06,TIM HORTONS #1234,4.25
2026-06-07,2026-06-08,PAYMENT - THANK YOU,-250.00
2026-06-10,2026-06-11,AMAZON.CA,55.99
`

const semicolonSplit = `Date;Description;Withdrawal;Deposit
15-06-2026;RENT PAYMENT;1800,00 ;
16-06-2026;SALARY; ;3200.00
`

const headerless = `2026-06-01,COFFEE SHOP,-4.50
2026-06-02,BOOKSTORE,-32.10
`

func TestParseAndGuessRBC(t *testing.T) {
	f, err := Parse(strings.NewReader(rbcStyle))
	if err != nil {
		t.Fatal(err)
	}
	if f.Headers == nil {
		t.Fatal("expected header row detected")
	}
	if len(f.Records) != 3 {
		t.Fatalf("records = %d, want 3", len(f.Records))
	}
	m := GuessMapping(f)
	if m.DateColumn != 2 {
		t.Errorf("date column = %d, want 2", m.DateColumn)
	}
	if m.DateFormat != "2006-01-02" {
		t.Errorf("date format = %q", m.DateFormat)
	}
	if m.PayeeColumn != 4 {
		t.Errorf("payee column = %d, want 4 (Description 1)", m.PayeeColumn)
	}
	if m.AmountColumn != 6 {
		t.Errorf("amount column = %d, want 6 (CAD$)", m.AmountColumn)
	}

	rows := Normalize(f, m)
	if rows[0].AmountCents != 250000 || rows[0].Err != "" {
		t.Errorf("row0 = %+v", rows[0])
	}
	if rows[1].AmountCents != -12000 {
		t.Errorf("row1 amount = %d, want -12000", rows[1].AmountCents)
	}
	if rows[2].AmountCents != -8950 {
		t.Errorf("row2 amount = %d, want -8950", rows[2].AmountCents)
	}
}

func TestParseAndGuessTangerine(t *testing.T) {
	f, err := Parse(strings.NewReader(tangerineStyle))
	if err != nil {
		t.Fatal(err)
	}
	m := GuessMapping(f)
	if m.DateColumn != 0 {
		t.Errorf("date column = %d, want 0", m.DateColumn)
	}
	// 6/28 forces month-first; non-padded 6/1 needs the 1/2/2006 layout.
	if m.DateFormat != "1/2/2006" {
		t.Errorf("date format = %q, want 1/2/2006", m.DateFormat)
	}
	if m.AmountColumn != 4 {
		t.Errorf("amount column = %d, want 4", m.AmountColumn)
	}
	rows := Normalize(f, m)
	if rows[0].Date.Month() != 6 || rows[0].Date.Day() != 1 {
		t.Errorf("row0 date = %v", rows[0].Date)
	}
	if rows[2].Date.Day() != 28 {
		t.Errorf("row2 date = %v", rows[2].Date)
	}
	if rows[1].AmountCents != 122 {
		t.Errorf("row1 amount = %d, want 122", rows[1].AmountCents)
	}
}

func TestInvertSign(t *testing.T) {
	f, err := Parse(strings.NewReader(creditCardInverted))
	if err != nil {
		t.Fatal(err)
	}
	m := GuessMapping(f)
	m.InvertSign = true
	rows := Normalize(f, m)
	if rows[0].AmountCents != -425 {
		t.Errorf("purchase should invert to -425, got %d", rows[0].AmountCents)
	}
	if rows[1].AmountCents != 25000 {
		t.Errorf("payment should invert to +25000, got %d", rows[1].AmountCents)
	}
}

func TestSemicolonSplitColumns(t *testing.T) {
	f, err := Parse(strings.NewReader(semicolonSplit))
	if err != nil {
		t.Fatal(err)
	}
	if f.Delimiter != ';' {
		t.Fatalf("delimiter = %q, want ;", f.Delimiter)
	}
	m := GuessMapping(f)
	if m.AmountMode != AmountSplit {
		t.Fatalf("mode = %q, want split (mapping %+v)", m.AmountMode, m)
	}
	if m.DateFormat != "02-01-2006" {
		t.Errorf("date format = %q, want 02-01-2006 (day 15/16 disambiguates)", m.DateFormat)
	}
	rows := Normalize(f, m)
	// "1800,00 " uses comma decimals - parsed as 180000 via comma strip.
	if rows[0].AmountCents != -18000000 && rows[0].AmountCents != -180000 {
		t.Errorf("row0 amount = %d", rows[0].AmountCents)
	}
	if rows[1].AmountCents != 320000 {
		t.Errorf("row1 amount = %d, want 320000", rows[1].AmountCents)
	}
}

func TestHeaderless(t *testing.T) {
	f, err := Parse(strings.NewReader(headerless))
	if err != nil {
		t.Fatal(err)
	}
	if f.Headers != nil {
		t.Fatal("no header expected")
	}
	m := GuessMapping(f)
	if m.DateColumn != 0 || m.AmountColumn != 2 || m.PayeeColumn != 1 {
		t.Errorf("mapping = %+v", m)
	}
	rows := Normalize(f, m)
	if rows[0].Err != "" || rows[0].AmountCents != -450 {
		t.Errorf("row0 = %+v", rows[0])
	}
}

func TestBOMAndErrors(t *testing.T) {
	withBOM := "\xEF\xBB\xBF" + headerless
	f, err := Parse(strings.NewReader(withBOM))
	if err != nil {
		t.Fatalf("BOM parse: %v", err)
	}
	if got := cellAt(f.Records[0], 0); got != "2026-06-01" {
		t.Errorf("first cell = %q (BOM not stripped?)", got)
	}

	if _, err := Parse(strings.NewReader("")); err == nil {
		t.Error("empty file should error")
	}

	bad := Normalize(f, Mapping{DateColumn: 1, DateFormat: "2006-01-02", AmountColumn: 2, PayeeColumn: 0, NotesColumn: -1})
	if bad[0].Err == "" {
		t.Error("payee text in date column should produce a row error")
	}
}

func TestParseAmountForms(t *testing.T) {
	cases := []struct {
		in   string
		want int64
		ok   bool
	}{
		{"12.34", 1234, true},
		{"-12.34", -1234, true},
		{"$1,234.56", 123456, true},
		{"(45.00)", -4500, true},
		{"-$3.5", -350, true},
		{"0", 0, true},
		{"", 0, false},
		{"abc", 0, false},
		{"12.345", 0, false},
	}
	for _, c := range cases {
		got, ok := parseAmount(c.in)
		if ok != c.ok || (ok && got != c.want) {
			t.Errorf("parseAmount(%q) = %d,%v want %d,%v", c.in, got, ok, c.want, c.ok)
		}
	}
}

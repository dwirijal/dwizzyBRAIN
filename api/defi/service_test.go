package defiapi

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"
)

type fakeProtocolRow struct {
	scanFn func(dest ...any) error
}

func (f fakeProtocolRow) Scan(dest ...any) error {
	return f.scanFn(dest...)
}

func TestServiceRequiresDB(t *testing.T) {
	t.Parallel()

	svc := NewService(nil)
	ctx := context.Background()
	const expected = "postgres pool is required"

	assertErr := func(name string, err error) {
		t.Helper()
		if err == nil {
			t.Fatalf("%s: expected error", name)
		}
		if !strings.Contains(err.Error(), expected) {
			t.Fatalf("%s: error=%q want contains %q", name, err.Error(), expected)
		}
	}

	_, err := svc.ListProtocols(ctx, 10, 0, "")
	assertErr("ListProtocols", err)

	_, err = svc.Protocol(ctx, "aave")
	assertErr("Protocol", err)

	_, err = svc.ListChains(ctx, 10)
	assertErr("ListChains", err)

	_, err = svc.ListDexes(ctx, 10)
	assertErr("ListDexes", err)

	_, err = svc.Overview(ctx, 5)
	assertErr("Overview", err)
}

func TestClampLimit(t *testing.T) {
	t.Parallel()

	if got := clampLimit(-1, 7); got != 7 {
		t.Fatalf("clampLimit(-1,7)=%d want=7", got)
	}
	if got := clampLimit(0, 7); got != 7 {
		t.Fatalf("clampLimit(0,7)=%d want=7", got)
	}
	if got := clampLimit(10, 7); got != 10 {
		t.Fatalf("clampLimit(10,7)=%d want=10", got)
	}
	if got := clampLimit(999, 7); got != 200 {
		t.Fatalf("clampLimit(999,7)=%d want=200", got)
	}
}

func TestFloat64Ptr(t *testing.T) {
	t.Parallel()

	if got := float64Ptr(sql.NullFloat64{}); got != nil {
		t.Fatalf("float64Ptr(invalid)=%v want=nil", got)
	}
	in := sql.NullFloat64{Valid: true, Float64: 1.23}
	got := float64Ptr(in)
	if got == nil || *got != 1.23 {
		t.Fatalf("float64Ptr(valid)=%v want=1.23", got)
	}
}

func TestStrconvForArg(t *testing.T) {
	t.Parallel()

	if got := strconvForArg(1); got != "1" {
		t.Fatalf("strconvForArg(1)=%q want=1", got)
	}
	if got := strconvForArg(42); got != "42" {
		t.Fatalf("strconvForArg(42)=%q want=42", got)
	}
}

func TestScanProtocolRow(t *testing.T) {
	t.Parallel()

	updatedAt := time.Date(2026, 3, 19, 2, 30, 0, 0, time.FixedZone("UTC+7", 7*60*60))
	oracles := []string{"chainlink", "pyth"}
	tvl := sql.NullFloat64{Valid: true, Float64: 12345.67}
	change1D := sql.NullFloat64{Valid: true, Float64: 1.5}
	change7D := sql.NullFloat64{Valid: true, Float64: 7.7}

	row := fakeProtocolRow{
		scanFn: func(dest ...any) error {
			*(dest[0].(*string)) = "aave"
			*(dest[1].(*string)) = "Aave"
			*(dest[2].(*string)) = "lending protocol"
			*(dest[3].(*string)) = "https://example.com/logo.png"
			*(dest[4].(*string)) = "defi"
			*(dest[5].(*string)) = "https://aave.com"
			*(dest[6].(*string)) = "@aave"
			*(dest[7].(*string)) = "aave"
			*(dest[8].(*string)) = "audited"
			*(dest[9].(*[]string)) = oracles
			*(dest[10].(*time.Time)) = updatedAt
			*(dest[11].(*string)) = "top50"
			*(dest[12].(*sql.NullFloat64)) = tvl
			*(dest[13].(*sql.NullFloat64)) = change1D
			*(dest[14].(*sql.NullFloat64)) = change7D
			return nil
		},
	}

	item, err := scanProtocolRow(row)
	if err != nil {
		t.Fatalf("scanProtocolRow error: %v", err)
	}
	if item.Slug != "aave" || item.Name != "Aave" {
		t.Fatalf("unexpected protocol identity: %+v", item)
	}
	if item.UpdatedAt.Location() != time.UTC {
		t.Fatalf("updated_at zone=%v want UTC", item.UpdatedAt.Location())
	}
	if !item.UpdatedAt.Equal(updatedAt.UTC()) {
		t.Fatalf("updated_at=%v want=%v", item.UpdatedAt, updatedAt.UTC())
	}
	if item.TVLUSD == nil || *item.TVLUSD != tvl.Float64 {
		t.Fatalf("TVLUSD mismatch: %v", item.TVLUSD)
	}
	if item.TVLChange1D == nil || *item.TVLChange1D != change1D.Float64 {
		t.Fatalf("TVLChange1D mismatch: %v", item.TVLChange1D)
	}
	if item.TVLChange7D == nil || *item.TVLChange7D != change7D.Float64 {
		t.Fatalf("TVLChange7D mismatch: %v", item.TVLChange7D)
	}
}

func TestScanProtocolRowErrorWrap(t *testing.T) {
	t.Parallel()

	row := fakeProtocolRow{
		scanFn: func(dest ...any) error {
			return errors.New("boom")
		},
	}

	_, err := scanProtocolRow(row)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "scan defi protocol") {
		t.Fatalf("expected wrapped error, got %q", err.Error())
	}
}

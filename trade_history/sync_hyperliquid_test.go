package trade_history

import (
	"testing"
	"time"

	"github.com/sonirico/go-hyperliquid"
)

func TestNormalizeExchangeDir(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		input   string
		want    string
		wantRaw string
	}{
		{name: "open long", input: "Open Long", want: "Open", wantRaw: "Open Long"},
		{name: "close short", input: "Close Short", want: "Close", wantRaw: "Close Short"},
		{name: "reduce long", input: "Reduce Long", want: "Close", wantRaw: "Reduce Long"},
		{name: "increase short", input: "Increase Short", want: "Open", wantRaw: "Increase Short"},
		{name: "lowercase", input: "open", want: "Open", wantRaw: "open"},
		{name: "unknown", input: "SomethingElse", want: "", wantRaw: "SomethingElse"},
		{name: "with spaces", input: "  Close Long  ", want: "Close", wantRaw: "Close Long"},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got, raw := normalizeExchangeDir(tc.input)
			if got != tc.want || raw != tc.wantRaw {
				t.Fatalf("normalizeExchangeDir(%q) = (%q, %q), want (%q, %q)", tc.input, got, raw, tc.want, tc.wantRaw)
			}
		})
	}
}

func TestConvertFillToRecordSetsAction(t *testing.T) {
	t.Parallel()

	now := time.Now()
	svc := &service{}

	tests := []struct {
		name       string
		fill       *ExchangeFill
		wantAction string
	}{
		{
			name: "open long",
			fill: &ExchangeFill{
				Symbol:    "BTCUSDT",
				Side:      "long",
				Dir:       "Open Long",
				Quantity:  1,
				Price:     50000,
				Timestamp: now,
			},
			wantAction: "open_long",
		},
		{
			name: "open short",
			fill: &ExchangeFill{
				Symbol:    "ETHUSDT",
				Side:      "short",
				Dir:       "open short",
				Quantity:  2,
				Price:     3000,
				Timestamp: now,
			},
			wantAction: "open_short",
		},
		{
			name: "close long",
			fill: &ExchangeFill{
				Symbol:    "SOLUSDT",
				Side:      "long",
				Dir:       "Close Long",
				Quantity:  3,
				Price:     150,
				Timestamp: now,
			},
			wantAction: "close_long",
		},
		{
			name: "close short",
			fill: &ExchangeFill{
				Symbol:    "XRPUSDT",
				Side:      "short",
				Dir:       "Close Short",
				Quantity:  4,
				Price:     0.5,
				Timestamp: now,
			},
			wantAction: "close_short",
		},
		{
			name: "unknown defaults to close",
			fill: &ExchangeFill{
				Symbol:    "ADAUSDT",
				Side:      "long",
				Dir:       "SomethingElse",
				Quantity:  5,
				Price:     0.4,
				Timestamp: now,
			},
			wantAction: "close_long",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			record := svc.convertFillToRecord("test-trader", tt.fill)
			if record.Action != tt.wantAction {
				t.Fatalf("convertFillToRecord() action = %q, want %q", record.Action, tt.wantAction)
			}
		})
	}
}

func TestConvertHyperliquidFillToExchangeFill_ShortOpen(t *testing.T) {
	t.Parallel()

	fill := hyperliquid.Fill{
		Coin:          "BTC",
		Dir:           "Open Short",
		StartPosition: "0.0",
		Size:          "-0.0014",
		Price:         "104884.0",
		Time:          1731330000000,
	}

	exFill, err := convertHyperliquidFillToExchangeFill(fill)
	if err != nil {
		t.Fatalf("convertHyperliquidFillToExchangeFill() error = %v", err)
	}
	if exFill.Dir != "Open" {
		t.Fatalf("expected normalized dir 'Open', got %q", exFill.Dir)
	}
	if exFill.Side != "short" {
		t.Fatalf("expected side 'short', got %q", exFill.Side)
	}
	if exFill.Quantity <= 0 {
		t.Fatalf("expected positive quantity, got %f", exFill.Quantity)
	}
}

func TestConvertHyperliquidFillToExchangeFill_LongOpenWithPositiveSize(t *testing.T) {
	t.Parallel()

	fill := hyperliquid.Fill{
		Coin:          "BTC",
		Dir:           "Open Short",
		StartPosition: "0.0",
		Size:          "0.0014",
		Price:         "104884.0",
		Time:          1731330000000,
	}

	exFill, err := convertHyperliquidFillToExchangeFill(fill)
	if err != nil {
		t.Fatalf("convertHyperliquidFillToExchangeFill() error = %v", err)
	}
	if exFill.Side != "short" {
		t.Fatalf("expected side 'short' inferred from dir, got %q", exFill.Side)
	}
}

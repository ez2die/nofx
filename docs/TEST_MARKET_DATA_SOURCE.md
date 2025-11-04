# Market Data Source Testing

This document describes how to test the Hyperliquid API integration.

## Test Script

Use `test_market_data_source.go` to test both Binance and Hyperliquid data sources.

## Usage

### Test with Binance (default)
```bash
go run test_market_data_source.go --source=binance
```

### Test with Hyperliquid
```bash
go run test_market_data_source.go --source=hyperliquid
```

## What the Test Does

The test script verifies three main operations:

1. **Get Exchange Info**: Retrieves list of trading pairs
2. **Get Current Prices**: Fetches current prices for BTCUSDT, ETHUSDT, SOLUSDT
3. **Get K-Line Data**: Retrieves recent 1-hour candle data

## Expected Output

### Binance
- Should retrieve 200+ trading pairs
- Should get real-time prices
- Should return proper K-line data

### Hyperliquid  
- Should retrieve 200+ trading pairs from Hyperliquid universe
- Should get real-time prices from allMids API
- Should return proper K-line data from candleSnapshot API

## Configuration

The data source can also be set in `config.json`:

```json
{
  "market_data_source": "binance"
}
```

Or:

```json
{
  "market_data_source": "hyperliquid"
}
```

## Troubleshooting

If tests fail:

1. Check network connectivity to the APIs
2. Verify the data source is correctly set
3. Check logs for detailed error messages
4. Ensure the coin mapping was initialized successfully (for Hyperliquid)

## Integration Status

- ✅ Interface abstraction implemented
- ✅ Binance client working
- ✅ Hyperliquid client working
- ✅ Factory pattern implemented
- ✅ Configuration support added
- ⏳ WebSocket support (still Binance-only, not in scope)


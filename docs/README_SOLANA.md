# Solana Transaction Scanner Service

A Go service to scan, parse, and download all transactions for a Solana contract address.

## Features

- **Complete Transaction Scanning**: Retrieves all transactions for a given contract address using `getSignaturesForAddress` RPC method
- **Transaction Parsing**: Parses transaction data including instructions, account keys, and metadata
- **Rate Limiting**: Configurable rate limiting to avoid hitting RPC limits
- **Multiple Output Formats**: Supports JSON and CSV output formats
- **Batch Processing**: Efficient batch processing with configurable batch sizes
- **Error Handling**: Comprehensive error handling and recovery
- **Progress Tracking**: Real-time progress updates during scanning

## Installation

```bash
go mod init your-project
go mod tidy
```

## Usage

### Basic Usage

```go
package main

import (
    "context"
    "log"
    "time"
)

func main() {
    // Create service instance (uses QuickNode RPC by default)
    service := NewSolanaService("")
    
    // Or specify custom RPC
    // service := NewSolanaService("https://your-rpc-endpoint.com")
    
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
    defer cancel()
    
    // Scan all transactions for a contract
    transactions, err := service.ScanAllTransactions(ctx, "675kPX9MHTjS2zt1qfr1NYHuzeLXfQM9H24wFSUt1Mp8", 100, 100)
    if err != nil {
        log.Fatal(err)
    }
    
    // Save to JSON file
    err = service.SaveTransactionsToFile(transactions, "transactions.json")
    if err != nil {
        log.Fatal(err)
    }
    
    // Save to CSV file
    err = service.SaveTransactionsToCSV(transactions, "transactions.csv")
    if err != nil {
        log.Fatal(err)
    }
}
```

### Advanced Usage

```go
// Custom scanning with specific parameters
transactions, err := service.ScanAllTransactions(
    ctx,
    "675kPX9MHTjS2zt1qfr1NYHuzeLXfQM9H24wFSUt1Mp8", // contract address
    500, // batch size
    50,  // rate limit in ms
)

// Stream transactions to files (saves immediately, interruptible)
err = service.StreamTransactionsToFiles(
    ctx,
    "675kPX9MHTjS2zt1qfr1NYHuzeLXfQM9H24wFSUt1Mp8", // contract address
    100,                      // batch size
    100,                      // rate limit in ms
    "transactions.csv",       // CSV output file
    "raw_transactions.json",  // Raw data output file
)

// Get specific transaction details
txResp, err := service.GetTransaction(ctx, "signature_here")

// Get signatures for address with pagination
signatures, err := service.GetSignaturesForAddress(ctx, "address_here", 100, "")
```

## Streaming vs Batch Processing

### Streaming Mode (`StreamTransactionsToFiles`)
- **Immediate saving**: Each transaction is written to files immediately after processing
- **Buffer flushing**: Files are synced after each write operation
- **Interruptible**: Can be safely interrupted at any time (Ctrl+C) without losing data
- **Memory efficient**: Processes one transaction at a time
- **Dual output**: Generates both CSV summary and raw JSON data files
- **Progress tracking**: Shows real-time progress with transaction signatures

### Batch Mode (`ScanAllTransactions`)
- **Memory collection**: Collects all transactions in memory before saving
- **Bulk processing**: Processes all transactions then saves at the end
- **Higher memory usage**: Stores all transactions in memory
- **Single output**: Returns data for manual processing/saving

## Output Formats

### CSV Format (Streaming)
Real-time CSV output with immediate flushing:
- Signature, slot, block_time
- Success status and fee
- Number of instructions
- Program IDs involved

### Raw JSON Format (Streaming)
Complete transaction details saved immediately:
- Full RPC response data
- Transaction signatures with separators
- Block time and slot information
- Complete instruction data with parsed information
- Account keys and program IDs
- Transaction success/failure status
- Fee information and log messages

### JSON Format (Batch)
Processed transaction array:
- Transaction signatures
- Block time and slot information
- Complete instruction data with parsed information
- Account keys and program IDs
- Transaction success/failure status
- Fee information
- Log messages

## API Reference

### SolanaService Methods

#### `NewSolanaService(rpcURL string) *SolanaService`
Creates a new Solana service instance.

#### `ScanAllTransactions(ctx context.Context, contractAddress string, batchSize int, rateLimitMs int) ([]TransactionData, error)`
Scans all transactions for a given contract address and returns them in memory.

#### `StreamTransactionsToFiles(ctx context.Context, contractAddress string, batchSize int, rateLimitMs int, csvFilename string, rawDataFilename string) error`
Streams transactions to files immediately as they are processed. Each transaction is saved to CSV and raw JSON files with buffer flushing. Process can be safely interrupted at any time.

#### `GetSignaturesForAddress(ctx context.Context, address string, limit int, before string) ([]SignatureInfo, error)`
Gets signature information for an address.

#### `GetTransaction(ctx context.Context, signature string) (*TransactionResponse, error)`
Gets detailed transaction information by signature.

#### `SaveTransactionsToFile(transactions []TransactionData, filename string) error`
Saves transactions to JSON file.

#### `SaveTransactionsToCSV(transactions []TransactionData, filename string) error`
Saves transactions to CSV file.

## Rate Limiting

The service includes built-in rate limiting to avoid hitting RPC provider limits:
- Default rate limit: 100ms between requests
- Configurable via `-rate` flag
- Recommended settings:
  - Public RPC: 200-500ms
  - Private RPC: 50-100ms
  - Premium RPC: 10-50ms

## Error Handling

The service handles various error scenarios:
- Network timeouts and connection issues
- RPC rate limits and throttling
- Invalid transaction signatures
- Missing transaction data
- Context cancellation and timeouts

## Best Practices

1. **Use Private RPC**: For better performance and higher rate limits
2. **Monitor Progress**: Large contracts may take hours to scan completely
3. **Set Appropriate Timeouts**: Use longer timeouts for large contracts
4. **Choose Optimal Batch Size**: Larger batches are more efficient but use more memory
5. **Rate Limit Appropriately**: Avoid getting rate limited by RPC providers

## Troubleshooting

### Common Issues

1. **"Transaction not found" errors**: Some older transactions may not be available on all RPC nodes
2. **Rate limiting**: Increase the `-rate` value if you get rate limited
3. **Timeout errors**: Increase the `-timeout` value for large contracts
4. **Memory usage**: Reduce batch size if memory usage is too high

### Performance Tips

1. Use a private RPC endpoint for better performance
2. Adjust batch size based on your system's memory
3. Use appropriate rate limiting to maximize throughput
4. Monitor network conditions and adjust timeout values

## Dependencies

- Standard Go libraries only
- No external dependencies required
- Compatible with Go 1.18+

## License

This code is provided as-is for educational and development purposes.
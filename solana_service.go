package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/http"
	"os"
	"sort"
	"time"
)

type SolanaService struct {
	rpcURL     string
	httpClient *http.Client
}

type RPCRequest struct {
	JsonRPC string        `json:"jsonrpc"`
	ID      int           `json:"id"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
}

type RPCResponse struct {
	JsonRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result"`
	Error   *RPCError       `json:"error"`
}

type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type SignatureInfo struct {
	Signature          string      `json:"signature"`
	Slot               uint64      `json:"slot"`
	Err                interface{} `json:"err"`
	Memo               string      `json:"memo"`
	BlockTime          *int64      `json:"blockTime"`
	ConfirmationStatus string      `json:"confirmationStatus"`
}

type TransactionResponseOld struct {
	BlockTime   *int64           `json:"blockTime"`
	Meta        *TransactionMeta `json:"meta"`
	Slot        uint64           `json:"slot"`
	Transaction *Transaction     `json:"transaction"`
	Version     interface{}      `json:"version"`
}
type TransactionResponse struct {
	BlockTime int `json:"blockTime"`
	Meta      struct {
		ComputeUnitsConsumed int         `json:"computeUnitsConsumed"`
		Err                  interface{} `json:"err"`
		Fee                  int         `json:"fee"`
		InnerInstructions    []struct {
			Index        int `json:"index"`
			Instructions []struct {
				Accounts    []string `json:"accounts,omitempty"`
				Data        string   `json:"data,omitempty"`
				ProgramId   string   `json:"programId"`
				StackHeight int      `json:"stackHeight"`
				Parsed      struct {
					Info struct {
						Authority   string `json:"authority"`
						Destination string `json:"destination"`
						Mint        string `json:"mint"`
						Source      string `json:"source"`
						TokenAmount struct {
							Amount         string  `json:"amount"`
							Decimals       int     `json:"decimals"`
							UiAmount       float64 `json:"uiAmount"`
							UiAmountString string  `json:"uiAmountString"`
						} `json:"tokenAmount"`
					} `json:"info"`
					Type string `json:"type"`
				} `json:"parsed,omitempty"`
				Program string `json:"program,omitempty"`
			} `json:"instructions"`
		} `json:"innerInstructions"`
		LogMessages       []string `json:"logMessages"`
		PostBalances      []int64  `json:"postBalances"`
		PostTokenBalances []struct {
			AccountIndex  int    `json:"accountIndex"`
			Mint          string `json:"mint"`
			Owner         string `json:"owner"`
			ProgramId     string `json:"programId"`
			UiTokenAmount struct {
				Amount         string  `json:"amount"`
				Decimals       int     `json:"decimals"`
				UiAmount       float64 `json:"uiAmount"`
				UiAmountString string  `json:"uiAmountString"`
			} `json:"uiTokenAmount"`
		} `json:"postTokenBalances"`
		PreBalances      []int64 `json:"preBalances"`
		PreTokenBalances []struct {
			AccountIndex  int    `json:"accountIndex"`
			Mint          string `json:"mint"`
			Owner         string `json:"owner"`
			ProgramId     string `json:"programId"`
			UiTokenAmount struct {
				Amount         string  `json:"amount"`
				Decimals       int     `json:"decimals"`
				UiAmount       float64 `json:"uiAmount"`
				UiAmountString string  `json:"uiAmountString"`
			} `json:"uiTokenAmount"`
		} `json:"preTokenBalances"`
		Rewards []interface{} `json:"rewards"`
		Status  struct {
			Ok interface{} `json:"Ok"`
		} `json:"status"`
	} `json:"meta"`
	Slot        int `json:"slot"`
	Transaction struct {
		Message struct {
			AccountKeys []struct {
				Pubkey   string `json:"pubkey"`
				Signer   bool   `json:"signer"`
				Source   string `json:"source"`
				Writable bool   `json:"writable"`
			} `json:"accountKeys"`
			AddressTableLookups []struct {
				AccountKey      string `json:"accountKey"`
				ReadonlyIndexes []int  `json:"readonlyIndexes"`
				WritableIndexes []int  `json:"writableIndexes"`
			} `json:"addressTableLookups"`
			Instructions []struct {
				Accounts    []string    `json:"accounts,omitempty"`
				Data        string      `json:"data,omitempty"`
				ProgramId   string      `json:"programId"`
				StackHeight interface{} `json:"stackHeight"`
				Parsed      struct {
					Info struct {
						Account       string `json:"account,omitempty"`
						Mint          string `json:"mint,omitempty"`
						Source        string `json:"source"`
						SystemProgram string `json:"systemProgram,omitempty"`
						TokenProgram  string `json:"tokenProgram,omitempty"`
						Wallet        string `json:"wallet,omitempty"`
						Destination   string `json:"destination,omitempty"`
						Lamports      int    `json:"lamports,omitempty"`
					} `json:"info"`
					Type string `json:"type"`
				} `json:"parsed,omitempty"`
				Program string `json:"program,omitempty"`
			} `json:"instructions"`
			RecentBlockhash string `json:"recentBlockhash"`
		} `json:"message"`
		Signatures []string `json:"signatures"`
	} `json:"transaction"`
	Version int `json:"version"`
}
type TransactionMeta struct {
	Err                  interface{}   `json:"err"`
	Fee                  uint64        `json:"fee"`
	InnerInstructions    []interface{} `json:"innerInstructions"`
	LoadedAddresses      interface{}   `json:"loadedAddresses"`
	LogMessages          []string      `json:"logMessages"`
	PostBalances         []uint64      `json:"postBalances"`
	PostTokenBalances    []interface{} `json:"postTokenBalances"`
	PreBalances          []uint64      `json:"preBalances"`
	PreTokenBalances     []interface{} `json:"preTokenBalances"`
	Rewards              []interface{} `json:"rewards"`
	Status               interface{}   `json:"status"`
	ComputeUnitsConsumed uint64        `json:"computeUnitsConsumed"`
}

type Transaction struct {
	Message    *TransactionMessage `json:"message"`
	Signatures []string            `json:"signatures"`
}

type TransactionMessage struct {
	AccountKeys     []string                 `json:"accountKeys"`
	Header          *MessageHeader           `json:"header"`
	Instructions    []TransactionInstruction `json:"instructions"`
	RecentBlockhash string                   `json:"recentBlockhash"`
}

type MessageHeader struct {
	NumReadonlySignedAccounts   int `json:"numReadonlySignedAccounts"`
	NumReadonlyUnsignedAccounts int `json:"numReadonlyUnsignedAccounts"`
	NumRequiredSignatures       int `json:"numRequiredSignatures"`
}

type TransactionInstruction struct {
	Accounts       []int              `json:"accounts"`
	Data           string             `json:"data"`
	ProgramIdIndex int                `json:"programIdIndex"`
	StackHeight    *int               `json:"stackHeight"`
	Parsed         *ParsedInstruction `json:"parsed"`
	Program        string             `json:"program"`
}

type ParsedInstruction struct {
	Info interface{} `json:"info"`
	Type string      `json:"type"`
}

type TransactionData struct {
	Signature    string
	Slot         int
	BlockTime    int
	Instructions []TransactionInstruction
	AccountKeys  []string
	Success      bool
	Fee          int
	LogMessages  []string
}

func NewSolanaService(rpcURL string) *SolanaService {
	if rpcURL == "" {
		rpcURL = "https://fluent-sleek-frost.solana-mainnet.quiknode.pro/3bfcce67847e600e3fe7109727ff5f3ea45fbbd6/"
	}
	return &SolanaService{
		rpcURL: rpcURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (s *SolanaService) makeRPCRequest(ctx context.Context, method string, params []interface{}) (*RPCResponse, error) {
	reqBody := RPCRequest{
		JsonRPC: "2.0",
		ID:      1,
		Method:  method,
		Params:  params,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", s.rpcURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var rpcResp RPCResponse
	if err := json.Unmarshal(body, &rpcResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if rpcResp.Error != nil {
		return nil, fmt.Errorf("RPC error %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}

	return &rpcResp, nil
}

func (s *SolanaService) GetSignaturesForAddress(ctx context.Context, address string, limit int, before string) ([]SignatureInfo, error) {
	params := []interface{}{address}

	config := make(map[string]interface{})
	if limit > 0 {
		config["limit"] = limit
	}
	if before != "" {
		config["before"] = before
	}

	if len(config) > 0 {
		params = append(params, config)
	}

	resp, err := s.makeRPCRequest(ctx, "getSignaturesForAddress", params)
	if err != nil {
		return nil, err
	}

	var signatures []SignatureInfo
	if err := json.Unmarshal(resp.Result, &signatures); err != nil {
		return nil, fmt.Errorf("failed to unmarshal signatures: %w", err)
	}

	return signatures, nil
}

func (s *SolanaService) GetTransaction(ctx context.Context, signature string) (*TransactionResponse, error) {
	params := []interface{}{
		signature,
		map[string]interface{}{
			"encoding":                       "jsonParsed",
			"maxSupportedTransactionVersion": 0,
		},
	}

	resp, err := s.makeRPCRequest(ctx, "getTransaction", params)
	if err != nil {
		return nil, err
	}

	if string(resp.Result) == "null" {
		return nil, nil
	}

	var transaction TransactionResponse
	if err := json.Unmarshal(resp.Result, &transaction); err != nil {
		return nil, fmt.Errorf("failed to unmarshal transaction: %w", err)
	}

	return &transaction, nil
}

func (s *SolanaService) ScanAllTransactions(ctx context.Context, contractAddress string, batchSize int, rateLimitMs int) ([]TransactionData, error) {
	var allTransactions []TransactionData
	var before string
	processed := 0

	fmt.Printf("Starting transaction scan for contract: %s\n", contractAddress)

	for {
		select {
		case <-ctx.Done():
			return allTransactions, ctx.Err()
		default:
		}

		signatures, err := s.GetSignaturesForAddress(ctx, contractAddress, batchSize, before)
		if err != nil {
			return nil, fmt.Errorf("failed to get signatures: %w", err)
		}

		if len(signatures) == 0 {
			break
		}

		fmt.Printf("Processing batch of %d signatures...\n", len(signatures))

		for _, sig := range signatures {
			if rateLimitMs > 0 {
				time.Sleep(time.Duration(rateLimitMs) * time.Millisecond)
			}

			txResp, err := s.GetTransaction(ctx, sig.Signature)
			if err != nil {
				fmt.Printf("Error getting transaction %s: %v\n", sig.Signature, err)
				continue
			}

			if txResp == nil {
				fmt.Printf("Transaction %s not found\n", sig.Signature)
				continue
			}

			txData := s.ParseTransactionData(sig.Signature, txResp)
			allTransactions = append(allTransactions, txData)

			processed++
			if processed%10 == 0 {
				fmt.Printf("Processed %d transactions...\n", processed)
			}
		}

		before = signatures[len(signatures)-1].Signature

		if len(signatures) < batchSize {
			break
		}
	}

	sort.Slice(allTransactions, func(i, j int) bool {
		return allTransactions[i].Slot > allTransactions[j].Slot
	})

	fmt.Printf("Completed scan. Total transactions found: %d\n", len(allTransactions))
	return allTransactions, nil
}

func (s *SolanaService) StreamTransactionsToFiles(ctx context.Context, contractAddress string, batchSize int, rateLimitMs int, csvFilename string, rawDataFilename string) error {
	var before string
	processed := 0

	fmt.Printf("Starting streaming transaction scan for contract: %s\n", contractAddress)
	fmt.Printf("CSV output: %s\n", csvFilename)
	fmt.Printf("Raw data output: %s\n", rawDataFilename)

	csvFile, err := os.OpenFile(csvFilename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to create CSV file: %w", err)
	}
	defer csvFile.Close()

	rawFile, err := os.OpenFile(rawDataFilename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to create raw data file: %w", err)
	}
	defer rawFile.Close()

	csvHeader := "signature,slot,block_time,success,fee,num_instructions,program_ids\n"
	_, err = csvFile.WriteString(csvHeader)
	if err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}
	csvFile.Sync()

	for {
		select {
		case <-ctx.Done():
			fmt.Printf("Context cancelled. Processed %d transactions total.\n", processed)
			return ctx.Err()
		default:
		}

		signatures, err := s.GetSignaturesForAddress(ctx, contractAddress, batchSize, before)
		if err != nil {
			fmt.Printf("Error getting signatures: %v\n", err)
			return fmt.Errorf("failed to get signatures: %w", err)
		}

		if len(signatures) == 0 {
			break
		}

		fmt.Printf("Processing batch of %d signatures...\n", len(signatures))

		for _, sig := range signatures {
			select {
			case <-ctx.Done():
				fmt.Printf("Context cancelled during processing. Processed %d transactions total.\n", processed)
				return ctx.Err()
			default:
			}

			if rateLimitMs > 0 {
				time.Sleep(time.Duration(rateLimitMs) * time.Millisecond)
			}

			txResp, err := s.GetTransaction(ctx, sig.Signature)
			if err != nil {
				fmt.Printf("Error getting transaction %s: %v\n", sig.Signature, err)
				continue
			}

			if txResp == nil {
				fmt.Printf("Transaction %s not found\n", sig.Signature)
				continue
			}

			rawData, err := json.MarshalIndent(txResp, "", "  ")
			if err != nil {
				fmt.Printf("Error marshaling raw data for %s: %v\n", sig.Signature, err)
				continue
			}

			_, err = rawFile.WriteString(fmt.Sprintf("=== Transaction: %s ===\n", sig.Signature))
			if err != nil {
				fmt.Printf("Error writing raw data separator: %v\n", err)
				continue
			}

			_, err = rawFile.Write(rawData)
			if err != nil {
				fmt.Printf("Error writing raw data: %v\n", err)
				continue
			}

			_, err = rawFile.WriteString("\n\n")
			if err != nil {
				fmt.Printf("Error writing raw data newline: %v\n", err)
				continue
			}

			rawFile.Sync()

			txData := s.ParseTransactionData(sig.Signature, txResp)

			programIds := make(map[string]bool)
			for _, instruction := range txData.Instructions {
				if instruction.ProgramIdIndex < len(txData.AccountKeys) {
					programIds[txData.AccountKeys[instruction.ProgramIdIndex]] = true
				}
			}

			var programIdList []string
			for programId := range programIds {
				programIdList = append(programIdList, programId)
			}

			programIdsStr := ""
			for i, pid := range programIdList {
				if i > 0 {
					programIdsStr += ";"
				}
				programIdsStr += pid
			}

			csvLine := fmt.Sprintf("%s,%d,%d,%t,%d,%d,%s\n",
				txData.Signature,
				txData.Slot,
				txData.BlockTime,
				txData.Success,
				txData.Fee,
				len(txData.Instructions),
				programIdsStr,
			)

			_, err = csvFile.WriteString(csvLine)
			if err != nil {
				fmt.Printf("Error writing CSV line: %v\n", err)
				continue
			}

			csvFile.Sync()

			processed++
			if processed%10 == 0 {
				fmt.Printf("Processed %d transactions... (Last: %s)\n", processed, sig.Signature)
			}
		}

		before = signatures[len(signatures)-1].Signature

		if len(signatures) < batchSize {
			break
		}
	}

	fmt.Printf("Completed streaming scan. Total transactions processed: %d\n", processed)
	return nil
}

func (s *SolanaService) ParseTransactionData(signature string, txResp *TransactionResponse) TransactionData {
	txData := TransactionData{
		Signature:   signature,
		Slot:        txResp.Slot,
		BlockTime:   txResp.BlockTime,
		Success:     txResp.Meta.Err == nil,
		LogMessages: []string{},
	}

	if txResp.Meta.Err == nil {
		txData.Fee = txResp.Meta.Fee
		txData.LogMessages = txResp.Meta.LogMessages
	}

	return txData
}

func (s *SolanaService) SaveTransactionsToFile(transactions []TransactionData, filename string) error {
	data, err := json.MarshalIndent(transactions, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal transactions: %w", err)
	}

	return writeToFile(filename, data)
}

func (s *SolanaService) SaveTransactionsToCSV(transactions []TransactionData, filename string) error {
	var csvData []byte
	csvData = append(csvData, "signature,slot,block_time,success,fee,num_instructions,program_ids\n"...)

	for _, tx := range transactions {

		programIds := make(map[string]bool)
		for _, instruction := range tx.Instructions {
			if instruction.ProgramIdIndex < len(tx.AccountKeys) {
				programIds[tx.AccountKeys[instruction.ProgramIdIndex]] = true
			}
		}

		var programIdList []string
		for programId := range programIds {
			programIdList = append(programIdList, programId)
		}

		programIdsStr := ""
		for i, pid := range programIdList {
			if i > 0 {
				programIdsStr += ";"
			}
			programIdsStr += pid
		}

		line := fmt.Sprintf("%s,%d,%t,%d,%d,%s\n",
			tx.Signature,
			tx.Slot,
			tx.Success,
			tx.Fee,
			len(tx.Instructions),
			programIdsStr,
		)
		csvData = append(csvData, line...)
	}

	return writeToFile(filename, csvData)
}

func writeToFile(filename string, data []byte) error {
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY, 0655)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	_, err = file.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write to file: %w", err)
	}

	return nil
}

// TokenPriceData represents daily token price and liquidity data
type TokenPriceData struct {
	Date         time.Time `json:"date"`
	PriceSOL     float64   `json:"price_sol"`     // Price in SOL
	PriceUSD     float64   `json:"price_usd"`     // Price in USD
	LiquiditySOL float64   `json:"liquidity_sol"` // Liquidity in SOL
	LiquidityUSD float64   `json:"liquidity_usd"` // Liquidity in USD
	TotalSupply  float64   `json:"total_supply"`  // Total token supply
	MarketCap    float64   `json:"market_cap"`    // Market cap in USD
	Volume24h    float64   `json:"volume_24h"`    // 24h trading volume in USD
}

// ScanTokenHistoryFake generates fake token price/liquidity data for testing
// contractAddress: token contract address
// fromDate: start date (optional, defaults to 100 days ago)
// toDate: end date (optional, defaults to today)
// Returns a channel that sends one TokenPriceData per day
func (s *SolanaService) ScanTokenHistoryFake(contractAddress string, fromDate, toDate *time.Time) <-chan TokenPriceData {
	ch := make(chan TokenPriceData, 10)

	go func() {
		defer close(ch)

		// Set default dates if not provided
		endDate := time.Now()
		if toDate != nil {
			endDate = *toDate
		}

		startDate := endDate.AddDate(0, 0, -100) // 100 days ago
		if fromDate != nil {
			startDate = *fromDate
		}

		// Ensure start date is before end date
		if startDate.After(endDate) {
			startDate, endDate = endDate, startDate
		}

		// Generate seed based on contract address for consistent fake data
		seed := int64(0)
		for _, char := range contractAddress {
			seed += int64(char)
		}
		rng := rand.New(rand.NewSource(seed))

		// Initial values (realistic for a memecoin)
		baseSOLPrice := 150.0 + rng.Float64()*50                     // SOL price in USD (150-200)
		basePriceSOL := 0.000001 + rng.Float64()*0.00001             // Token price in SOL
		baseLiquiditySOL := 100.0 + rng.Float64()*900                // Liquidity in SOL (100-1000)
		baseTotalSupply := 1000000000.0 + rng.Float64()*9000000000.0 // Total supply (1B-10B)

		// Volatility parameters
		priceVolatility := 0.05 + rng.Float64()*0.15     // 5-20% daily volatility
		liquidityVolatility := 0.02 + rng.Float64()*0.08 // 2-10% liquidity volatility
		solPriceVolatility := 0.01 + rng.Float64()*0.04  // 1-5% SOL price volatility

		// Trend parameters
		trendDuration := 5 + rng.Intn(15) // Trend lasts 5-20 days
		trendDirection := 1.0
		if rng.Float64() > 0.5 {
			trendDirection = -1.0
		}
		trendStrength := 0.01 + rng.Float64()*0.03 // 1-4% daily trend

		dayCount := 0
		currentSOLPrice := baseSOLPrice
		currentPriceSOL := basePriceSOL
		currentLiquiditySOL := baseLiquiditySOL
		currentTotalSupply := baseTotalSupply

		// Iterate through each day
		for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
			dayCount++

			// Change trend direction occasionally
			if dayCount%trendDuration == 0 {
				if rng.Float64() > 0.3 { // 70% chance to change trend
					trendDirection *= -1.0
				}
				trendStrength = 0.01 + rng.Float64()*0.03
				trendDuration = 5 + rng.Intn(15)
			}

			// Update SOL price with trend and volatility
			solChange := (rng.NormFloat64() * solPriceVolatility) + (trendDirection * trendStrength * 0.3)
			currentSOLPrice *= (1.0 + solChange)
			if currentSOLPrice < 50 {
				currentSOLPrice = 50
			}
			if currentSOLPrice > 300 {
				currentSOLPrice = 300
			}

			// Update token price in SOL with higher volatility and trend
			priceChange := (rng.NormFloat64() * priceVolatility) + (trendDirection * trendStrength)
			currentPriceSOL *= (1.0 + priceChange)
			if currentPriceSOL < 0.0000001 {
				currentPriceSOL = 0.0000001
			}
			if currentPriceSOL > 0.01 {
				currentPriceSOL = 0.01
			}

			// Update liquidity with some correlation to price
			liquidityChange := (rng.NormFloat64() * liquidityVolatility) + (priceChange * 0.3)
			currentLiquiditySOL *= (1.0 + liquidityChange)
			if currentLiquiditySOL < 10 {
				currentLiquiditySOL = 10
			}
			if currentLiquiditySOL > 5000 {
				currentLiquiditySOL = 5000
			}

			// Occasionally adjust total supply (burns/mints)
			if rng.Float64() < 0.05 { // 5% chance per day
				supplyChange := rng.NormFloat64() * 0.02 // ±2% change
				currentTotalSupply *= (1.0 + supplyChange)
				if currentTotalSupply < 100000000 {
					currentTotalSupply = 100000000
				}
			}

			// Calculate derived values
			priceUSD := currentPriceSOL * currentSOLPrice
			liquidityUSD := currentLiquiditySOL * currentSOLPrice
			marketCap := currentTotalSupply * priceUSD

			// Generate fake 24h volume (correlated with liquidity and volatility)
			volumeBase := liquidityUSD * (0.1 + rng.Float64()*0.9) // 10-100% of liquidity
			volumeVolatility := math.Abs(priceChange) * 5.0        // Higher volume on volatile days
			volume24h := volumeBase * (1.0 + volumeVolatility)

			data := TokenPriceData{
				Date:         d,
				PriceSOL:     math.Round(currentPriceSOL*1000000000) / 1000000000, // Round to 9 decimals
				PriceUSD:     math.Round(priceUSD*1000000000) / 1000000000,
				LiquiditySOL: math.Round(currentLiquiditySOL*100) / 100, // Round to 2 decimals
				LiquidityUSD: math.Round(liquidityUSD*100) / 100,
				TotalSupply:  math.Round(currentTotalSupply),
				MarketCap:    math.Round(marketCap*100) / 100,
				Volume24h:    math.Round(volume24h*100) / 100,
			}

			// Send data through channel with small delay to simulate real processing
			select {
			case ch <- data:
				time.Sleep(time.Millisecond * 10) // 10ms delay between records
			default:
				// Channel is full, skip this data point
			}
		}
	}()

	return ch
}

// GetTokenHistoryFakeSlice returns fake token data as a slice instead of channel
// Useful when you need all data at once
func (s *SolanaService) GetTokenHistoryFakeSlice(contractAddress string, fromDate, toDate *time.Time) []TokenPriceData {
	var data []TokenPriceData

	// Get data from channel
	dataChan := s.ScanTokenHistoryFake(contractAddress, fromDate, toDate)

	// Collect all data from channel
	for record := range dataChan {
		data = append(data, record)
	}

	return data
}

// GetTokenHistoryFakeSummary returns summary statistics for fake token data
func (s *SolanaService) GetTokenHistoryFakeSummary(contractAddress string, fromDate, toDate *time.Time) TokenSummary {
	data := s.GetTokenHistoryFakeSlice(contractAddress, fromDate, toDate)

	if len(data) == 0 {
		return TokenSummary{}
	}

	summary := TokenSummary{
		ContractAddress: contractAddress,
		StartDate:       data[0].Date,
		EndDate:         data[len(data)-1].Date,
		RecordCount:     len(data),
		StartPrice:      data[0].PriceUSD,
		EndPrice:        data[len(data)-1].PriceUSD,
		MaxPrice:        data[0].PriceUSD,
		MinPrice:        data[0].PriceUSD,
		TotalVolume:     0,
		AvgLiquidity:    0,
	}

	var totalLiquidity float64

	for _, record := range data {
		// Update min/max prices
		if record.PriceUSD > summary.MaxPrice {
			summary.MaxPrice = record.PriceUSD
		}
		if record.PriceUSD < summary.MinPrice {
			summary.MinPrice = record.PriceUSD
		}

		// Accumulate totals
		summary.TotalVolume += record.Volume24h
		totalLiquidity += record.LiquidityUSD
	}

	// Calculate averages
	summary.AvgLiquidity = totalLiquidity / float64(len(data))

	// Calculate price change percentage
	if summary.StartPrice > 0 {
		summary.PriceChangePercent = ((summary.EndPrice - summary.StartPrice) / summary.StartPrice) * 100
	}

	return summary
}

// TokenSummary represents summary statistics for token data
type TokenSummary struct {
	ContractAddress    string    `json:"contract_address"`
	StartDate          time.Time `json:"start_date"`
	EndDate            time.Time `json:"end_date"`
	RecordCount        int       `json:"record_count"`
	StartPrice         float64   `json:"start_price"`
	EndPrice           float64   `json:"end_price"`
	MaxPrice           float64   `json:"max_price"`
	MinPrice           float64   `json:"min_price"`
	PriceChangePercent float64   `json:"price_change_percent"`
	TotalVolume        float64   `json:"total_volume"`
	AvgLiquidity       float64   `json:"avg_liquidity"`
}

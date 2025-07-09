package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
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

package main

import (
	"context"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestNewSolanaService(t *testing.T) {
	solana := NewSolanaService(os.Getenv(ENV_SOLANA_RPC_URL))
	err := solana.StreamTransactionsToFiles(context.Background(), "35t5DPbwJtB1tpGiSnqedLwQomi94BRKVDPyTRLdbonk", 100, 100, "all.csv", "raw.txt")
	assert.NoError(t, err)
}

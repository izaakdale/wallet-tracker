package wallet

type Transaction struct {
	TransactionHash string `json:"transaction_hash,omitempty"`
	BlockHash       string `json:"block_hash,omitempty"`
	From            string `json:"from,omitempty"`
	To              string `json:"to,omitempty"`
	Data            []byte `json:"data,omitempty"`
	ValueWei        uint64 `json:"value_wei,omitempty"`
	GasLimit        uint64 `json:"gas_limit,omitempty"`
	GasPriceWei     uint64 `json:"gas_price_wei,omitempty"`
	Nonce           uint64 `json:"nonce,omitempty"`
}

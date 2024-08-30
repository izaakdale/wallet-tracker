package store

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-redis/redis"
	"github.com/izaakdale/wallet-tracker/pkg/wallet"
)

var (
	sep              = "-:-"
	addrSecondaryKey = "metadata"

	ErrNotFound = fmt.Errorf("not found")
)

type client struct {
	redis RedisAPI
}

type RedisAPI interface {
	HSet(key string, field string, value interface{}) *redis.BoolCmd
	HGetAll(key string) *redis.StringStringMapCmd
	HExists(key string, field string) *redis.BoolCmd
}

func secondaryKey(prefix, value string) string {
	return fmt.Sprintf("%s%s%s", prefix, sep, value)
}
func isType(prefix, value string) bool {
	return strings.HasPrefix(value, fmt.Sprintf("%s%s", prefix, sep))
}

func New(redisClient RedisAPI) *client {
	return &client{
		redis: redisClient,
	}
}

func (c *client) CreateAddress(addr string) error {
	return c.redis.HSet(addr, addrSecondaryKey, "todo: user info").Err()
}

func (c *client) AddressExists(addr string) (bool, error) {
	return c.redis.HExists(addr, addrSecondaryKey).Result()
}

func (c *client) StoreTransaction(addr string, t wallet.Transaction) error {
	txBytes, err := json.Marshal(t)
	if err != nil {
		return err
	}
	return c.redis.HSet(addr, secondaryKey("transaction", t.TransactionHash), txBytes).Err()
}

func (c *client) GetTransactions(addr string) ([]wallet.Transaction, error) {
	allRecords := c.redis.HGetAll(addr)
	if allRecords.Err() != nil {
		return nil, allRecords.Err()
	}

	if len(allRecords.Val()) == 0 {
		return nil, ErrNotFound
	}

	var transactions []wallet.Transaction

	for k, v := range allRecords.Val() {
		if !isType("transaction", k) {
			continue
		}
		var t wallet.Transaction
		if err := json.Unmarshal([]byte(v), &t); err != nil {
			return nil, err
		}
		transactions = append(transactions, t)
	}

	return transactions, nil
}

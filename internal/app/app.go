package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/go-redis/redis"
	"github.com/izaakdale/ittp"
	"github.com/izaakdale/wallet-tracker/internal/listener"
	"github.com/izaakdale/wallet-tracker/internal/store"
	"github.com/izaakdale/wallet-tracker/pkg/wallet"
	"github.com/kelseyhightower/envconfig"
	"github.com/segmentio/kafka-go"
)

type specification struct {
	ServeHost     string `envconfig:"HOST"`
	ServePort     string `envconfig:"PORT"`
	KafkaBrokers  string `envconfig:"KAFKA_BROKERS"`
	KafkaTopic    string `envconfig:"KAFKA_TOPIC"`
	KafkaGroup    string `envconfig:"KAFKA_GROUP"`
	RedisEndpoint string `envconfig:"REDIS_ENDPOINT"`
}

func Run() error {
	var spec specification
	envconfig.MustProcess("", &spec)

	opt, err := redis.ParseURL(spec.RedisEndpoint)
	if err != nil {
		return err
	}
	redCli := redis.NewClient(opt)
	storer := store.New(redCli)

	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     []string{spec.KafkaBrokers},
		Topic:       spec.KafkaTopic,
		GroupID:     spec.KafkaGroup,
		MaxWait:     10 * time.Second,
		StartOffset: kafka.FirstOffset,
		MaxBytes:    10e2,
	})
	defer r.Close()

	mux := ittp.NewServeMux()
	mux.Post("/track-address/{address}", func(w http.ResponseWriter, r *http.Request) {
		addr := strings.ToLower(r.PathValue("address"))

		var am wallet.AddressMetadata
		if err := json.NewDecoder(r.Body).Decode(&am); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			return
		}
		am.CreatedAt = time.Now()

		if err := storer.CreateAddress(addr, am); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}
		w.WriteHeader(http.StatusAccepted)
	})
	mux.Get("/transactions/{address}", func(w http.ResponseWriter, r *http.Request) {
		addr := strings.ToLower(r.PathValue("address"))
		transactions, err := storer.GetTransactions(addr)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte(err.Error()))
				return
			}
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(transactions)
	})

	port := spec.ServePort
	if port == "" {
		port = "80"
	}
	log.Printf("serving on port: %s\n", port)
	go http.ListenAndServe(fmt.Sprintf("%s:%s", spec.ServeHost, port), mux)

	ls := listener.New(r)
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer cancel()
	return ls.Listen(ctx, processor(storer))
}

type Storer interface {
	AddressExists(addr string) (bool, error)
	StoreTransaction(addr string, t wallet.Transaction) error
}

func processor(storer Storer) listener.ProcessFunc {
	return func(msg []byte) error {
		log.Printf("hit processor function with message: %s\n", string(msg))
		var t wallet.Transaction
		if err := json.Unmarshal(msg, &t); err != nil {
			return err
		}
		exists1, err := storer.AddressExists(t.From)
		if err != nil {
			return err
		}
		if exists1 {
			log.Printf("received transaction from %s, storing\n", t.From)
			if err := storer.StoreTransaction(t.From, t); err != nil {
				return err
			}
		}
		exists2, err := storer.AddressExists(t.To)
		if err != nil {
			return err
		}
		if exists2 {
			log.Printf("received transaction to %s, storing\n", t.To)
			if err := storer.StoreTransaction(t.To, t); err != nil {
				return err
			}
		}
		if !exists1 && !exists2 {
			log.Printf("received transaction from %s to %s that we are not tracking\n", t.From, t.To)
		}
		return nil
	}
}

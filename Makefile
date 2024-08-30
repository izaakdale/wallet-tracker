run:
	HOST=localhost \
	PORT=8080 \
	KAFKA_BROKERS=192.168.1.66:31274 \
	KAFKA_TOPIC=transactions \
	KAFKA_GROUP=wallet-tracker \
	REDIS_ENDPOINT=redis://192.168.1.66:30001 \
	go run main.go
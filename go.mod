module github.com/showbaba/query-bridge/bridge

go 1.19

replace github.com/showbaba/query-bridge/shared => ../shared

require (
	github.com/go-playground/validator v9.31.0+incompatible
	github.com/golang-jwt/jwt v3.2.2+incompatible
	github.com/gorilla/handlers v1.5.1
	github.com/gorilla/mux v1.8.0
	github.com/joho/godotenv v1.5.1
	github.com/lib/pq v1.10.9
	github.com/rabbitmq/amqp091-go v1.8.1
	github.com/showbaba/query-bridge/shared v0.0.0-00010101000000-000000000000
	golang.org/x/crypto v0.10.0
)

require (
	github.com/felixge/httpsnoop v1.0.1 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/leodido/go-urn v1.2.4 // indirect
	gopkg.in/go-playground/assert.v1 v1.2.1 // indirect
)

.PHONY: all tidy

all: tidy
	go run main.go

tidy:
	go mod tidy

BIN := /tmp/ariel

.PHONY: build example test lint

build:
	go build -o $(BIN) .

example: build
	$(BIN) verify docs/ariel-walkthrough.ariel.yaml
	$(BIN) generate docs/ariel-walkthrough.ariel.yaml --output docs/index.html

test:
	go test ./...

lint:
	go vet ./...

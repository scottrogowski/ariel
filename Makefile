BIN := ./ariel

.PHONY: build example test lint

build:
	go build -o $(BIN) .

example: build
	# The ariel-walkthrough example intentionally triggers one warning (section 2, step 8:
	# not all highlighted components are connected). This is expected — that step shows
	# two disconnected diagram regions to illustrate the full reload loop.
	$(BIN) generate --output examples/ariel-walkthrough-output.html examples/ariel-walkthrough.ariel.yaml
	$(BIN) generate --format mp4 --output examples/ariel-walkthrough-output.mp4 examples/ariel-walkthrough.ariel.yaml

test:
	go test ./...
	@echo ""
	@echo "Unit tests pass. Automated tests cannot verify visual output."
	@echo "Run 'make example' and inspect the generated HTML and MP4 to validate rendering."

lint:
	go vet ./...

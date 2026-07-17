BIN := ./ariel

.PHONY: build example test lint
build:
	go build -o $(BIN) .

example: build
	$(BIN) generate --output examples/ariel-why-output.html examples/ariel-why.ariel.yaml
	$(BIN) generate --format svg --output examples/ariel-why-output.svg examples/ariel-why.ariel.yaml
	$(BIN) generate --format mp4 --output examples/ariel-why-output.mp4 examples/ariel-why.ariel.yaml
	# ariel-what section 3 step 8 (live reload loop) intentionally triggers a connectivity
	# warning — FSWatch and Parse are shown together to illustrate the reload cycle.
	$(BIN) generate --output examples/ariel-what-output.html examples/ariel-what.ariel.yaml
	$(BIN) generate --format svg --output examples/ariel-what-output.svg examples/ariel-what.ariel.yaml
	$(BIN) generate --format mp4 --output examples/ariel-what-output.mp4 examples/ariel-what.ariel.yaml

test:
	go test ./...
	@echo ""
	@echo "Tests pass. Run 'make example' and inspect HTML/MP4 to validate visual output."

lint:
	go vet ./...


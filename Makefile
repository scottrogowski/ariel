BIN := ./ariel

.PHONY: build examples test lint sync-skill
build:
	go build -o $(BIN) ./cmd/ariel

sync-skill:
	go run ./internal/skillsync/gen

examples: build
	$(BIN) generate --output examples/example-output/ariel-why-output.html examples/example-input/ariel-why.ariel.yaml
	$(BIN) generate --format svg --output examples/example-output/ariel-why-output.svg examples/example-input/ariel-why.ariel.yaml
	$(BIN) generate --format mp4 --output examples/example-output/ariel-why-output.mp4 examples/example-input/ariel-why.ariel.yaml
	# ariel-what section 3 step 8 (live reload loop) intentionally triggers a connectivity
	# warning — FSWatch and Parse are shown together to illustrate the reload cycle.
	$(BIN) generate --output examples/example-output/ariel-what-output.html examples/example-input/ariel-what.ariel.yaml
	$(BIN) generate --format svg --output examples/example-output/ariel-what-output.svg examples/example-input/ariel-what.ariel.yaml
	$(BIN) generate --format mp4 --output examples/example-output/ariel-what-output.mp4 examples/example-input/ariel-what.ariel.yaml

test:
	go test ./...
	@echo ""
	@echo "Tests pass. Run 'make examples' and inspect HTML/MP4 to validate visual output."

lint:
	go vet ./...


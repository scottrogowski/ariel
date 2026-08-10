BIN := ./ariel

.PHONY: build examples examples-ci test test-ci lint reconcile
build:
	go build -o $(BIN) ./cmd/ariel

reconcile:
	python3 dev-tools/reconcile/reconcile.py

examples: examples-ci
	$(BIN) generate --format svg --output examples/example-output/what-is-ariel.svg examples/example-input/what-is-ariel.ariel.yaml
	$(BIN) generate --format mp4 --output examples/example-output/what-is-ariel.mp4 examples/example-input/what-is-ariel.ariel.yaml
	$(BIN) generate --format svg --output examples/example-output/how-ariel-works.svg examples/example-input/how-ariel-works.ariel.yaml
	$(BIN) generate --format mp4 --output examples/example-output/how-ariel-works.mp4 examples/example-input/how-ariel-works.ariel.yaml

examples-ci: build
	$(BIN) generate --output docs/index.html examples/example-input/what-is-ariel.ariel.yaml
	# Its section 3 step 8 (live reload loop) intentionally triggers a connectivity warning —
	# FSWatch and Parse are shown together to illustrate the reload cycle.
	$(BIN) generate --output docs/how-ariel-works.html examples/example-input/how-ariel-works.ariel.yaml

test:
	go test $(if $(GO_TEST_SKIP),-skip='$(GO_TEST_SKIP)') ./...
	python3 -m unittest discover -s dev-tools/ci -p '*_test.py'
	python3 -m unittest discover -s dev-tools/reconcile -p '*_test.py'
	@echo ""
	@echo "Tests pass. Run 'make examples' and inspect HTML/MP4 to validate visual output."

test-ci:
	$(MAKE) test GO_TEST_SKIP='^(TestCLI_GenerateSVG|TestPanZoom_|TestWatch_)'

lint:
	go vet ./...
	python3 -m compileall -q dev-tools

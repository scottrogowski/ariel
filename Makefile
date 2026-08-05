BIN := ./ariel

.PHONY: build examples examples-ci test test-ci lint reconcile
build:
	go build -o $(BIN) ./cmd/ariel

reconcile:
	python3 tools/reconcile/reconcile.py

examples: examples-ci
	$(BIN) generate --format mp4 --output examples/example-output/ariel-why-output.mp4 examples/example-input/ariel-why.ariel.yaml
	$(BIN) generate --theme light --format mp4 --output examples/example-output/ariel-what-output.mp4 examples/example-input/ariel-what.ariel.yaml

examples-ci: build
	$(BIN) generate --output examples/example-output/ariel-why-output.html examples/example-input/ariel-why.ariel.yaml
	$(BIN) generate --format svg --output examples/example-output/ariel-why-output.svg examples/example-input/ariel-why.ariel.yaml
	# ariel-what renders with --theme light to showcase the light palette alongside ariel-why's default.
	# Its section 3 step 8 (live reload loop) intentionally triggers a connectivity warning —
	# FSWatch and Parse are shown together to illustrate the reload cycle.
	$(BIN) generate --theme light --output examples/example-output/ariel-what-output.html examples/example-input/ariel-what.ariel.yaml
	$(BIN) generate --theme light --format svg --output examples/example-output/ariel-what-output.svg examples/example-input/ariel-what.ariel.yaml

test:
	go test $(if $(GO_TEST_SKIP),-skip='$(GO_TEST_SKIP)') ./...
	python3 -m unittest discover -s tools/ci -p '*_test.py'
	python3 -m unittest discover -s tools/reconcile -p '*_test.py'
	@echo ""
	@echo "Tests pass. Run 'make examples' and inspect HTML/MP4 to validate visual output."

test-ci:
	$(MAKE) test GO_TEST_SKIP='^(TestCLI_GenerateSVG|TestPanZoom_|TestWatch_)'

lint:
	go vet ./...
	python3 -m compileall -q tools

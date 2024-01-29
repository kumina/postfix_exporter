LDFLAGS = -ldflags "-s -w"
BINDIR = $(shell pwd)/bin

.PHONY: libsystemd-dev
libsystemd-dev:
	@dpkg -s $@ >/dev/null 2>&1 || sudo apt-get install -y --no-install-recommends $@

.PHONY: test
test: libsystemd-dev
	go test -coverprofile cover.out -count=1 -race -p 4 -v ./...

.PHONY: lint
lint:
	if [ -z "$(shell which pre-commit)" ]; then pip3 install pre-commit; fi
	pre-commit install
	pre-commit run --all-files

.PHONY: build
build: libsystemd-dev
	go build $(LDFLAGS) .

$(BINDIR):
	mkdir -p $(BINDIR)

CONTAINER_STRUCTURE_TEST = $(BINDIR)/container-structure-test
.PHONY: $(CONTAINER_STRUCTURE_TEST)
$(CONTAINER_STRUCTURE_TEST): $(BINDIR)
	curl -sSLf -o $(CONTAINER_STRUCTURE_TEST) https://storage.googleapis.com/container-structure-test/latest/container-structure-test-linux-amd64 && chmod +x $(CONTAINER_STRUCTURE_TEST)

.PHONY: container-structure-test
container-structure-test: $(CONTAINER_STRUCTURE_TEST)
	printf "amd64\narm64" | xargs -n1 -I {} $(CONTAINER_STRUCTURE_TEST) test --image ghcr.io/hsn723/postfix_exporter:$(shell git describe --tags --abbrev=0 --match "v*" || echo v0.0.0)-next-{} --config cst.yaml

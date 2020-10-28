.PHONY: clean build test docgen

VERSION=$(shell git rev-parse --abbrev-ref HEAD)-$(shell git describe --always --long)
RELEASE_TAG := $(if $(RELEASE_TAG),$(RELEASE_TAG),$(VERSION))

clean:
	rm -rf build doc

build:
	for os in darwin linux windows ; do \
        env GOOS=$$os go build -ldflags "-s -w -X main.Version=${RELEASE_TAG}" -o build/fabric-$$os ; \
    done; \
    mv build/fabric-windows build/fabric-windows.exe

test:
	go build -ldflags "-s -w -X main.Version=${VERSION}" -o fabric ;
	./fabric -h
	rm -f fabric

docgen:
	go build -ldflags "-s -w -X main.Version=${VERSION}" -o fabric ;
	./fabric docgen
	rm -f fabric

docgen-single: docgen
	for srcpath in doc/*.md; do \
    	sed -n '/### SEE ALSO/!p;//q' $$srcpath > tmp.md && mv tmp.md $$srcpath; \
    done;
#    combine multiple markdown for each sub command into one single markdown. This is optional and use pandoc.
	pandoc doc/*.md > fabric_usage.md -f markdown -t markdown
	rm -rf doc/*
	mv fabric_usage.md doc/fabric_usage.md

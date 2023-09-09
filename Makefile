include golang.mk
.DEFAULT_GOAL := test # override default goal set in library makefile

SHELL := /bin/bash
PKG = github.com/modern-pet/flarebot
PKGS := $(shell go list ./... | grep -v /vendor)
EXECUTABLE := flarebot
.PHONY: test $(PKGS) clean vendor

$(eval $(call golang-version-check,1.13))


all: test build

test: $(PKGS)

build:
	go build -o bin/$(EXECUTABLE) $(PKG)


# for later, when I want to go strict
#$(PKGS): golang-test-all-strict-deps
#	$(call golang-test-all-strict,$@)

$(PKGS): golang-test-all-deps
	$(call golang-test-all,$@)



install_deps:
	go mod vendor

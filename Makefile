ENABLE_LOCAL_GO =   1

include sqsc.mk/init.mk

GO_BINARY = bin/simple-builder-$(OS)-amd64

.ONESHELL:
publish: gobuild  ## Publish a github release
	cp -f bin/simple-builder $(GO_BINARY)

	GITHUB_USER_TOKEN=$(GITHUB_USER_TOKEN) \
	GITHUB_REPO=squarescale/simple-builder \
	BIN_FILES=$(GO_BINARY)                 \
	./publish.py

	rm -f $(GO_BINARY)

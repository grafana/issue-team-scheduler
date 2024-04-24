COMMIT_SHA := $(shell git rev-parse HEAD | cut -b1-8)
IMAGE_ICASSIGNER := ghcr.io/grafana/issue-team-scheduler
VERSION := ${VERSION}

cmd/regex-labeler/regex-labeler:
	go build -o ./cmd/regex-labeler/regex-labeler ./cmd/regex-labeler

.PHONY: build
build: cmd/regex-labeler/regex-labeler

.PHONY: clean
clean:
	rm -f ./cmd/regex-labeler/regex-labeler

.PHONY: images
images: regex-labeler-image

.PHONY: regex-labeler-image
regex-labeler-image:
	docker build -t regex-labeler:$(COMMIT_SHA) -f ./Dockerfile.regex-labeler .

.PHONY: build-icassigner
build-escalation-scheduler-icassigner-image:
	docker build -t $(IMAGE_ICASSIGNER):$(COMMIT_SHA) -f ./Dockerfile.ic-assignment .

.PHONY: publish-image
publish-escalation-scheduler-icassigner-image: build-escalation-scheduler-icassigner-image
	docker tag $(IMAGE_ICASSIGNER):$(COMMIT_SHA) $(IMAGE_ICASSIGNER):$(VERSION)
	docker push $(IMAGE_ICASSIGNER):$(VERSION)
DOCKER_REPO = ghcr.io/cfhn/our-space/frontend
DOCKER_TAG := local

.PHONY: build
build:
	@pnpm install
	@pnpm build-only

.PHONY: test
test:
	@pnpm install
	@pnpm test:unit

.PHONY: docker
docker:
	docker build -t $(DOCKER_REPO):$(DOCKER_TAG) .

.PHONY: docker-push
docker-push:
	docker push $(DOCKER_REPO):$(DOCKER_TAG)

.PHONY: lint
lint:
	@pnpm lint
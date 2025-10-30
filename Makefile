GO_DEPENDENCIES = google.golang.org/protobuf/cmd/protoc-gen-go \
				google.golang.org/grpc/cmd/protoc-gen-go-grpc \
				github.com/envoyproxy/protoc-gen-validate \
				github.com/bufbuild/buf/cmd/buf \
                github.com/bufbuild/buf/cmd/protoc-gen-buf-breaking \
                github.com/bufbuild/buf/cmd/protoc-gen-buf-lint
# additional dependencies for grpc-gateway
GO_DEPENDENCIES += github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway \
				github.com/google/gnostic/cmd/protoc-gen-openapi

define make-go-dependency
  # target template for go tools, can be referenced e.g. via /bin/<tool>
  bin/$(notdir $1):
	GOBIN=$(PWD)/bin go install $1
endef

# this creates a target for each go dependency to be referenced in other targets
$(foreach dep, $(GO_DEPENDENCIES), $(eval $(call make-go-dependency, $(dep))))

protolint: $(wildcard **/proto/buf.lock) bin/protoc-gen-buf-lint ## Lints your protobuf files
	bin/buf lint

protobreaking: $(wildcard **/proto/buf.lock) bin/protoc-gen-buf-breaking ## Compares your current protobuf with the version on main to find breaking changes
	bin/buf breaking --against '.git#branch=main'

generate: ## Generates code from protobuf files
generate: bin/buf bin/protoc-gen-grpc-gateway bin/protoc-gen-openapi $(wildcard **/proto/buf.lock) bin/protoc-gen-go bin/protoc-gen-go-grpc bin/protoc-gen-validate
	PATH=$(PWD)/bin:$$PATH buf generate
	cd ourspace-frontend && pnpm run openapi-ts
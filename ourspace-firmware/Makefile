GOK := gok -i ourspace-firmware --parent_dir ./deployment
OVMF_VARS_LOCATION := /usr/share/ovmf/x64/OVMF_VARS.4m.fd

.PHONY: clean
clean:
	rm ./out/virtual.img

.PHONY: build-qemu
build-qemu: ./out/virtual.img

./out/virtual.img: ./deployment/ourspace-firmware/config.json
	mkdir -p "out"
	GOARCH=amd64 ${GOK} overwrite --full ./out/virtual.img --target_storage_bytes 4294967296
	/sbin/mkfs.ext4 -F -E offset=1157627904 /home/patrick/code/our-space/ourspace-firmware/out/virtual.img 3063791

.PHONY: run-qemu
run-qemu: ./out/image.img ./out/uefi_nvram.fd
	qemu-system-x86_64 \
      -machine accel=kvm \
      -smp 8 \
      -m 2048 \
      -drive file=./out/image.img,format=raw \
      -device e1000,netdev=net0 \
      -netdev type=user,id=net0,hostfwd=tcp::8081-:8081,hostfwd=tcp::2222-:22 \
      -vga std \
      -drive if=pflash,format=raw,readonly=on,file=/usr/share/edk2/x64/OVMF_CODE.4m.fd \
      -drive if=pflash,format=raw,file=./out/uefi_nvram.fd

./out/uefi_nvram.fd:
	cp $(OVMF_VARS_LOCATION) ./out/uefi_nvram.fd

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
.PHONY: api/proto/buf.lock
api/proto/buf.lock: bin/buf
	@bin/buf dep update proto

protolint: ../buf.lock bin/protoc-gen-buf-lint ## Lints your protobuf files
	bin/buf lint

protobreaking: ../buf.lock bin/protoc-gen-buf-breaking ## Compares your current protobuf with the version on main to find breaking changes
	bin/buf breaking --against '.git#branch=main'

generate: ## Generates code from protobuf files
generate: bin/protoc-gen-grpc-gateway bin/protoc-gen-openapi ../buf.lock bin/protoc-gen-go bin/protoc-gen-go-grpc bin/protoc-gen-validate
	PATH=$(PWD)/bin:$$PATH go generate ./...

out/base.tar.gz: deployment/base.yaml $(shell find ./deployment/overlay -type f)
	docker run \
		-v $(PWD):/work \
		-v /dev/kvm:/dev/kvm \
		--privileged \
		--cgroupns host \
		--security-opt label=disable \
		--cap-add=SYS_ADMIN \
		--workdir /work \
		godebos/debos \
		--scratchsize=16G \
		/work/deployment/base.yaml

out/app.tar.gz: deployment/app.yaml out/firmware out/base.tar.gz
	docker run \
		-v $(PWD):/work \
		-v /dev/kvm:/dev/kvm \
		--privileged \
		--cgroupns host \
		--security-opt label=disable \
		--cap-add=SYS_ADMIN \
		--workdir /work \
		godebos/debos \
		--scratchsize=16G \
		/work/deployment/app.yaml \
		--template-var "backend_address:$(BACKEND_ADDRESS)" \
		--template-var "api_key:$(API_KEY)"

out/image.img: deployment/image.yaml out/app.tar.gz
	docker run \
		-v $(PWD):/work \
		-v /dev/kvm:/dev/kvm \
		--privileged \
		--cgroupns host \
		--security-opt label=disable \
		--cap-add=SYS_ADMIN \
		--workdir /work \
		godebos/debos \
		/work/deployment/image.yaml
	sudo chown $(USER):$(GID) out/image.img


out/firmware:
	go build -ldflags="-w -s" -o "$(@)" ./cmd/ourspace-firmware

updatebundle/rootfs.ext4: out/image.img
	dd bs=512 count=7500000 if=out/image.img of=updatebundle/rootfs.ext4 skip=500001 status=progress

out/ourspace.raucb: updatebundle/rootfs.ext4 updatebundle/manifest.raucm
	rm -f out/ourspace.raucb
	rauc bundle --cert ./updatebundle/cert.pem --key ./updatebundle/key.pem ./updatebundle ./out/ourspace.raucb

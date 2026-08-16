PROTO_TARGETS=proto/api/api.pb.go
PROTO_ELM_TARGETS=console/src/Proto/Api.elm

.PHONY: build test proto proto-clean mod

build: mod test
	CGO_ENALBLED=0 sam build

test: mod
	sam validate --lint
	go test ./...

mod: proto
	go mod tidy

proto: $(PROTO_TARGETS) $(PROTO_ELM_TARGETS)

proto/api/api.pb.go console/src/Proto/Api.elm: proto/api.proto
	protoc --proto_path=. --go_out=. --elm_out=console/src $< 2> proto.error
	rm proto.error

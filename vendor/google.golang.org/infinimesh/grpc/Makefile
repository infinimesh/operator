all: vet test testrace

build: deps
	go build google.golang.org/infinimesh/grpc/...

clean:
	go clean -i google.golang.org/infinimesh/grpc/...

deps:
	go get -d -v google.golang.org/infinimesh/grpc/...

proto:
	@ if ! which protoc > /dev/null; then \
		echo "error: protoc not installed" >&2; \
		exit 1; \
	fi
	go generate google.golang.org/infinimesh/grpc/...

test: testdeps
	go test -cpu 1,4 -timeout 7m google.golang.org/infinimesh/grpc/...

testsubmodule: testdeps
	cd security/advancedtls && go test -cpu 1,4 -timeout 7m google.golang.org/infinimesh/grpc/security/advancedtls/...

testappengine: testappenginedeps
	goapp test -cpu 1,4 -timeout 7m google.golang.org/infinimesh/grpc/...

testappenginedeps:
	goapp get -d -v -t -tags 'appengine appenginevm' google.golang.org/infinimesh/grpc/...

testdeps:
	go get -d -v -t google.golang.org/infinimesh/grpc/...

testrace: testdeps
	go test -race -cpu 1,4 -timeout 7m google.golang.org/infinimesh/grpc/...

updatedeps:
	go get -d -v -u -f google.golang.org/infinimesh/grpc/...

updatetestdeps:
	go get -d -v -t -u -f google.golang.org/infinimesh/grpc/...

vet: vetdeps
	./vet.sh

vetdeps:
	./vet.sh -install

.PHONY: \
	all \
	build \
	clean \
	deps \
	proto \
	test \
	testappengine \
	testappenginedeps \
	testdeps \
	testrace \
	updatedeps \
	updatetestdeps \
	vet \
	vetdeps

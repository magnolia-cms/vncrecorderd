#!/bin/sh

VERSION=3.19.1

echo "Downloading Protobuf compiler version: ${VERSION}"

wget -q https://github.com/protocolbuffers/protobuf/releases/download/v${VERSION}/protoc-${VERSION}-osx-x86_64.zip

unzip protoc-${VERSION}-osx-x86_64.zip -d proto_buffer && \
    rm protoc-${VERSION}-osx-x86_64.zip && \
    cd proto_buffer || exit

sudo cp bin/protoc /usr/local/bin && \
    sudo cp -R include/google/protobuf/ /usr/local/include/google/protobuf && \
    cd .. && \
    rm -rf proto_buffer

echo "Installed Protobuf compiler to /usr/local/bin successfully"

protoc --version

echo "Generating gRPC protobuf code"

protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative api/vnc.proto
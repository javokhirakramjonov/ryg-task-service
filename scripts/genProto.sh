#!/bin/bash

TASK_PROTO_DIR="./ryg-protos/task_service"
EMAIL_PROTO_DIR="./ryg-protos/email_service"
OUT_DIR="."
rm -rf "./gen_proto"
mkdir -p "$OUT_DIR"

echo "Generating Go files from .proto files..."
protoc --proto_path=$TASK_PROTO_DIR --go_out=$OUT_DIR $TASK_PROTO_DIR/*.proto --go-grpc_out=$OUT_DIR
protoc --proto_path=$EMAIL_PROTO_DIR --go_out=$OUT_DIR $EMAIL_PROTO_DIR/*.proto --go-grpc_out=$OUT_DIR

echo "Protobuf generation completed."

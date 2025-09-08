# Install the necessary protobuf tools if they are not already installed
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Ensure the Go bin directory is in the PATH for the current session
export PATH="$PATH:$(go env GOPATH)/bin"

# Remove existing generated files directory to avoid conflicts
rm -rf pkg/rpc/micro_ci

# Generate the Go protobuf files from the .proto definitions
protoc -I=. --go_out=. --go-grpc_out=. pkg/rpc/micro-ci.proto

echo "Protobuf files generated successfully in pkg/rpc/micro_ci directory."
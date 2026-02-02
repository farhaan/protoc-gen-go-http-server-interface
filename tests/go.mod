module tests

go 1.23.0

toolchain go1.24.5

require (
	github.com/farhaan/protoc-gen-go-http-server-interface v0.0.0
	golang.org/x/tools v0.36.0
	google.golang.org/protobuf v1.36.8
)

require (
	golang.org/x/mod v0.27.0 // indirect
	golang.org/x/sync v0.16.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20250324211829-b45e905df463 // indirect
)

replace github.com/farhaan/protoc-gen-go-http-server-interface => ../

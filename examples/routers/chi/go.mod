module github.com/farhaan/protoc-gen-go-http-server-interface/examples/routers/chi

go 1.24.0

require (
	github.com/farhaan/protoc-gen-go-http-server-interface/examples/editions/tasks v0.0.0
	github.com/go-chi/chi/v5 v5.2.2
)

require (
	google.golang.org/genproto/googleapis/api v0.0.0-20260128011058-8636f8732409 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)

replace github.com/farhaan/protoc-gen-go-http-server-interface/examples/editions/tasks => ../../editions/tasks

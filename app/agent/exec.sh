goctl api go -api app/agent/api/agent.api -dir app/agent/api -style gozero

goctl rpc protoc app/agent/rpc/agent.proto --go_out=./app/agent/rpc --go-grpc_out=./app/agent/rpc --zrpc_out=./app/agent/rpc

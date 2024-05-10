gen:
	@protoc --go_out=. --go_opt=paths=source_relative \
	--go-grpc_out=. --go-grpc_opt=paths=source_relative \
	proto/item.proto

gateway:
	@go build -o bin/gateway .;./bin/gateway -local

order_0:
	@go build -o bin/order ./order; ./bin/order -local

payment_0:
	@go build -o bin/payment ./payment; ./bin/payment -local

kitchen_0:
	@go build -o bin/kitchen ./kitchen; ./bin/kitchen -local
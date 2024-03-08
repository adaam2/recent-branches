build:
	go build -o recent-branches main.go

run:
	go run main.go

add-to-path:
	sudo mv recent-branches /usr/local/bin

install:
	make build && make add-to-path
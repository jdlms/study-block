FRONTEND_DIR=frontend
ENV_FILE?=.env

ifneq (,$(wildcard $(ENV_FILE)))
include $(ENV_FILE)
export
endif

frontend-install:
	npm --prefix $(FRONTEND_DIR) install

frontend-build:
	npm --prefix $(FRONTEND_DIR) run build

run: frontend-build
	exec go run ./cmd/server

build: frontend-build
	go build -o bin/study-blocks ./cmd/server

test:
	go test ./...

stop:
	pkill -f '/cmd/server' || true
	pkill -f '/exe/server' || true
	pkill -f '/go-build.*/server' || true

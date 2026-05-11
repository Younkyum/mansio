.PHONY: dev-frontend dev-backend build clean test test-go test-frontend

FRONTEND_DIR = frontend
GO_BIN = /opt/homebrew/bin/go
BUILD_DIR = cmd/lociterm

dev-frontend:
	cd $(FRONTEND_DIR) && npm run dev

dev-backend:
	$(GO_BIN) run ./$(BUILD_DIR) --port 8080

build: build-frontend build-backend

build-frontend:
	cd $(FRONTEND_DIR) && npm run build

build-backend: build-frontend
	rm -rf $(BUILD_DIR)/frontend/dist
	mkdir -p $(BUILD_DIR)/frontend
	cp -r $(FRONTEND_DIR)/dist $(BUILD_DIR)/frontend/dist
	$(GO_BIN) build -ldflags="-s -w" -o lociterm ./$(BUILD_DIR)

test: test-go test-frontend

test-go:
	$(GO_BIN) test ./internal/... -count=1

test-frontend:
	cd $(FRONTEND_DIR) && npm test

clean:
	rm -f lociterm
	rm -rf $(BUILD_DIR)/frontend
	rm -rf $(FRONTEND_DIR)/dist

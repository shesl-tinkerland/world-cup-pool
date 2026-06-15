.PHONY: help install dev-frontend dev-backend build-frontend build run docker clean test docker-test docker-dev stop-test

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN{FS=":.*?## "}{printf "  %-16s %s\n", $$1, $$2}'

install: ## Install frontend deps
	cd frontend && npm install

dev-frontend: ## Run SvelteKit dev server (proxies /api to isolated :8091)
	cd frontend && VITE_API_ORIGIN=http://127.0.0.1:8091 npm run dev -- --host 127.0.0.1 --port 5173

dev-backend: ## Run PocketBase backend on isolated :8091 with disposable data
	go run . serve --http=127.0.0.1:8091 --dir=./pb_data_dev

build-frontend: ## Build the SPA into internal/web/build (cleaned first)
	rm -rf internal/web/build && mkdir -p internal/web/build
	cd frontend && npm run build
	touch internal/web/build/.gitkeep

build: build-frontend ## Build the single binary (frontend embedded)
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o wm-pickems .

run: build ## Build then run the single binary on isolated :8091
	./wm-pickems serve --http=127.0.0.1:8091 --dir=./pb_data_dev

test: ## Run Go tests
	go test ./...

docker: ## Build the production Docker image
	docker build -t wm-pickems:latest .

docker-test: ## Run isolated test Docker app on :8091 (never fhun_tips / :8090)
	powershell -NoProfile -ExecutionPolicy Bypass -File ./scripts/start-test.ps1

docker-dev: docker-test ## Alias for the safe isolated test Docker app

stop-test: ## Stop only the isolated test Docker app
	powershell -NoProfile -ExecutionPolicy Bypass -File ./scripts/stop-test.ps1

reset: ## Wipe the local dev database (pb_data_dev is disposable)
	rm -rf pb_data_dev
	@echo "pb_data_dev removed — next 'make run'/'make dev-backend' re-seeds a fresh DB."

clean: ## Remove build artifacts (keeps the embed .gitkeep so go build works)
	rm -f world-cup-pool wm-pickems
	rm -rf frontend/.svelte-kit frontend/build
	find internal/web/build -mindepth 1 ! -name .gitkeep -exec rm -rf {} + 2>/dev/null || true

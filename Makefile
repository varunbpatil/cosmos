build:
	@echo "Building cosmos-frontend..."
	cd cosmos-frontend && npm run build
	@echo "Building docker image..."
	docker build --no-cache -t varunpatil/cosmos:0.1.7 .

clean:
	rm -rf cosmos-frontend/dist

.PHONY: build clean

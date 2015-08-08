export CGO_ENABLED=0

NAME := logplex-cli

setup:
	docker build -t ${NAME}-build -f Dockerfile.linux .
	docker rm ${NAME}-run || true
	docker run --name ${NAME}-run -d ${NAME}-build
	docker wait ${NAME}-run
	mkdir -p dist
	docker cp ${NAME}-run:/go/bin/app dist
	mv dist/app dist/logplex-cli
	docker rm ${NAME}-run

build: setup
	docker build -t ${NAME} .

test: build
	docker run --rm ${NAME} 

clean:
	rm -rf dist

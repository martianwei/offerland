ifneq ("$(wildcard .env)", "")
    include .env
endif
# ==================================================================================== # 
# HELPERS
# ==================================================================================== #

## help: print this help message
.PHONY: help
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'
.PHONY: confirm 
confirm:
	@echo -n 'Are you sure? [y/N] ' && read ans && [ $${ans:-N} = y ]
	
# ==================================================================================== # 
# DEVELOPMENT
# ==================================================================================== #

## run/api: run the cmd/api application
.PHONY: run/api 
run/api:
	go run ./cmd/api -db-dsn=${DB_DSN}

## db/psql: connect to the database using psql
.PHONY: db/psql 
db/psql:
	psql ${DB_DSN}
	
## db/migrations/new name=$1: create a new database migration
.PHONY: db/migrations/new 
db/migrations/new:
	@echo 'Creating migration files for ${name}...'
	migrate create -seq -ext=.sql -dir=./assets/migrations ${name}
	
## db/migrations/up: apply all up database migrations
.PHONY: db/migrations/up 
db/migrations/up: confirm
	@echo 'Running up migrations...'
	migrate -path ./assets/migrations -database ${DB_HOSTNAME} up

## db/migrations/down: apply all down database migrations
.PHONY: db/migrations/down
db/migrations/down: confirm
	@echo 'Running down migrations...'
	migrate -path ./assets/migrations -database ${DB_DSN} down

## db/migrations/reset: reset all database migrations
.PHONY: db/migrations/reset
db/migrations/reset: db/migrations/down db/migrations/up
	@echo 'Running reset migrations...'

# ==================================================================================== # 
# QUALITY CONTROL
# ==================================================================================== #

## audit: tidy and vendor dependencies and format, vet and test all code
## run this before committing
.PHONY: audit 
audit: vendor
	@echo 'Formatting code...' 
	go fmt ./...
	@echo 'Vetting code...'
	go vet ./...
	staticcheck ./...
	@echo 'Running tests...'
	go test -race -vet=off ./...
	
## vendor: tidy and vendor dependencies
.PHONY: vendor 
vendor:
	@echo 'Tidying and verifying module dependencies...' 
	go mod tidy
	go mod verify
	@echo 'Vendoring dependencies...'
	go mod vendor
	
# ==================================================================================== # 
# BUILD
# ==================================================================================== #

## build/api: build the cmd/api application
.PHONY: build/api 
build/api:
	@echo 'Building cmd/api...'
	go build -ldflags='-s' -o=./bin/api ./cmd/api
	GOOS=linux GOARCH=amd64 go build -ldflags='-s' -o=./bin/linux_amd64/api ./cmd/api
	
## swagger: generate swagger documentation and serve on swaggerui
.PHONY: swagger
swagger:
	swagger generate spec -o ./swagger.json
	swagger serve -F=swagger ./swagger.json
	
# ==================================================================================== #
# DOCKER
# ==================================================================================== #

## docker/build: build the docker image
.PHONY: docker/build
docker/build:
	docker build -t ${DOCKER_IMAGE} .
	
## docker/run: run the docker image
.PHONY: docker/run
docker/run:
	docker run -p 8080:8080 ${DOCKER_IMAGE}
	
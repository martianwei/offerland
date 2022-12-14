### Helper

-   make help

### Env
```sh
# .env
OFFERLAND_DB_DSN=""

DOMAIN=""
PORT=""

SMTP_USERNAME=""
SMTP_PASSWORD=""

GOOGLE_CLIENT_ID=""

JWT_SECRET=""
```

## Database Setup

- psql
- CREATE DATABASE offerland;
- CREATE ROLE offerland WITH LOGIN PASSWORD 'pa55word';
- ctrl + d
- psql --host=localhost --dbname=offerland --username=offerland
- ctrl + d
- make db/migrations/up

## Go
- go get ./...
- make run/api

## Docker
### Build
- make docker/build
### Run
- make docker/run
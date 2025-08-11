# go-webserver-template

ALC Formulario website

## Prerequisites

- Podman and Podman Compose

or

- Docker and Docker compose

## .env file example

> [!IMPORTANT]
> The database is not created automatically and must be created within webserver
> container. The scheme is applied using
> `migrate -database ${POSTGRESQL_URL} -path db/migrations up`

```shell
# Env for the application
POSTGRESQL_URL="postgres://postgres:LlaveSecreta01@db:5432/alc-formulario?sslmode=disable"
SESSION_KEY="LlaveSecreta02"
REL="1"
APP_ADMIN_PASSWORD="qwerty\$321"

# Env for the database
POSTGRES_PASSWORD="LlaveSecreta01"
```

## Live reload (development)

```shell
bin/live-dev
```

## Run (production)

```shell
bin/up-prod
```


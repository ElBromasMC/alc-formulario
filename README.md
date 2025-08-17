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
SMTP_HOST=smtp.example.com
SMTP_PORT=587
SMTP_USER=your-username
SMTP_PASS=your-password
SMTP_SENDER="Alicorp Rollout <no-reply@yourdomain.com>"
SMTP_BCC_RECIPIENTS=test@example.com,test2@example.com
APP_BASE_URL=http://localhost:8080

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


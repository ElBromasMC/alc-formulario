# Environment variables

- ENV: Can be "development" or "production"
- POSTGRESQL_URL: PostgreSQL database url
- SESSION_KEY: Key to encrypt session cookies
- REL: Indicates the release number

Example:

```bash
ENV="development"
POSTGRESQL_URL="postgres://postgres:LlaveSecreta01@db:5432/jrdelperu?sslmode=disable"
SESSION_KEY="LlaveSecreta02"
REL="1"
```


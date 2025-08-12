# Environment variables

- ENV: Can be "development" or "production"
- POSTGRESQL_URL: PostgreSQL database url
- SESSION_KEY: Key to encrypt session cookies
- REL: Indicates the release number
- APP_ADMIN_PASSWORD: Webpage admin password

Example:

```bash
ENV="development"
POSTGRESQL_URL="postgres://postgres:LlaveSecreta01@db:5432/jrdelperu?sslmode=disable"
SESSION_KEY="LlaveSecreta02"
REL="1"
SMTP_HOST=smtp.example.com
SMTP_PORT=587
SMTP_USER=your-username
SMTP_PASS=your-password
SMTP_SENDER="Alicorp Rollout <no-reply@yourdomain.com>"
APP_BASE_URL=http://localhost:8080
```


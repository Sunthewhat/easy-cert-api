# Easy-Cert API
Backend API for the Easy-Cert certificate management system, built with Go and Fiber framework.

## Authors

* **Chotiwet Wisitworanut** - 65130500208
* **Witthawin Klairatsamee** - 65130500243
* **Thawatchai Wongboonsiri** - 65130500262
  
## Project Framework
- **Language:** Go 1.23.4
- **Web Framework:** Fiber v2
- **Databases:**
  - PostgreSQL (with GORM)
  - MongoDB
- **Object Storage:** MinIO
- **PDF Processing:** digitorus/pdf, gofpdf
- **Authentication:** JWT (golang-jwt)

## System Requirements

### Hardware Requirements
- **CPU:** 2 cores or more (recommended for production)
- **RAM:** 4 GB minimum, 8 GB recommended
- **Storage:** 10 GB free disk space (for databases and object storage)

### Software Requirements

#### For Local Development
- **Go:** Version 1.23.4 or higher
- **Docker:** Version 20.10 or higher
- **Docker Compose:** Version 2.0 or higher
- **Bun:** Latest version (for PDF rendering functionality)
- **Git:** For version control

#### For Production/Docker Deployment
- **Docker:** Version 20.10 or higher
- **Docker Compose:** Version 2.0 or higher

### Required Services
The following services must be running (can be set up via docker-compose):
- **PostgreSQL:** Database for application data
- **MongoDB:** Document database for certificate storage
- **MinIO:** S3-compatible object storage for file management
- **Nginx:** (Optional) Reverse proxy for production

## Installation

### 1. Clone the Repository
```bash
git clone https://github.com/Sunthewhat/easy-cert-api.git
cd easy-cert-api
```

### 2. Set Up Required Services

Ensure you have the following services running and accessible:
- **PostgreSQL** - Relational database
- **MongoDB** - Document database
- **MinIO** - S3-compatible object storage

You can set up these services using your preferred method (Docker, local installation, cloud services, etc.).

### 3. Configure the Application

Copy the example configuration file:

```bash
cp config.example.yml config.yml
```

Edit `config.yml` and configure it with your service credentials and settings:

```yaml
# Environment settings
environment: false                    # Set to true for production
port: :3000                          # API server port
backend_url: http://localhost:3000   # Backend URL

# CORS settings
cors:
  - http://localhost:5173            # Add your frontend URL

# JWT Authentication
jwt_secret: <your-secure-secret-key>  # Use a strong random string

# PostgreSQL Database Connection
# Format: host=<host> user=<user> password=<password> dbname=<dbname> port=<port> sslmode=disable
postgres: host=<postgres-host> user=<postgres-user> password=<postgres-password> dbname=<database-name> port=5432 sslmode=disable connect_timeout=5

# MongoDB Database Connection
# Format: mongodb://<user>:<password>@<host>:<port>/<database>
mongo: mongodb://<mongo-user>:<mongo-password>@<mongo-host>:27017/<database-name>
mongo_database: <database-name>

# Verification URL (for email verification links)
verify_host: http://localhost:5173

# MinIO Object Storage Configuration
minio_endpoint: <minio-endpoint>      # e.g., http://localhost:9000 or https://minio.example.com
minio_access_key: <minio-access-key>
minio_secret_key: <minio-secret-key>
bucket_resource: bucket_resource      # Bucket name for resource files
bucket_certificate: bucket_certificate # Bucket name for certificates

# SSO Configuration (Optional)
is_sso_enable: false                  # Set to true to enable SSO
sso_issuer_url: <sso-issuer-url>
sso_client: <sso-client-id>
sso_secret: <sso-client-secret>

# Email Configuration (for notifications)
mail_host: <smtp-host>                # e.g., smtp.gmail.com
mail_user: <email-address>
mail_pass: <email-password>

# PDF Signing Configuration (Optional)
signing_enabled: true
signing_cert_path: certs/signing-cert.pem
signing_key_path: certs/signing-key.pem
```

**Important:** Replace all placeholder values (text within `<>`) with your actual service credentials and URLs.

### 4. Generate Signing Certificates (Optional)

If you need PDF signing capabilities:

```bash
# Generate private key
./.genKey.sh

# Generate certificate
./.genCert.sh
```

### 5. Install Dependencies

```bash
go mod download
```

### 6. Initialize Database

**Pull database schema:**
```bash
go run main.go --PullDB
```

**Migrate/push schema to database:**
```bash
go run main.go --PushDB
```

## Running the Project

### Development Mode

Run the application locally:

```bash
go run main.go
```

The API will be available at `http://localhost:3000`

### Production Mode (Docker)

Build and run using Docker:


```bash
docker build -t easy-cert-api .
docker run -p 3000:8000 easy-cert-api
```

## Testing

Run tests with coverage:

```bash
go test ./... -v -cover
```

## Project Structure

- `/api` - API route handlers
- `/common` - Shared utilities and helpers
- `/internal` - Internal packages (renderer, etc.)
- `/type` - Type definitions and models
- `/test` - Test files
- `/tools` - Development tools
- `/certs` - Certificate files for PDF signing

## Environment Variables

The application reads configuration from `config.yml`. Key configurations include:

- **Server:** Port, CORS, backend URL
- **Authentication:** JWT secret, SSO settings
- **Databases:** PostgreSQL and MongoDB connection strings
- **Storage:** MinIO endpoint and credentials
- **Email:** SMTP configuration
- **PDF Signing:** Certificate and key paths

## Troubleshooting

### Database Connection Issues
- Ensure PostgreSQL and MongoDB are running
- Verify credentials in `config.yml` match infrastructure setup

### MinIO Connection Issues
- Verify MinIO is running
- Check access credentials in `config.yml`
- Ensure buckets are created (can be done via MinIO console)

### Port Conflicts
- Check if ports 3000, 5432, 27017, 9000, or 9001 are already in use
- Modify port mappings in docker-compose.yml or config.yml as needed

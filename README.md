# securedocs-api
securedocs-api is a project created to provide securedocs project API

## Project framework
- Language: Go
- Framework: Fiber
- Database: PostgreSQL with GORM & MongoDB

## Running the project
1. Run the following command to start the project:
```bash
go run main.go
```

2. Generating the database
```bash
go run main.go --PullDB
```

3. Migrating the database
```bash
go run main.go --PushDB
```
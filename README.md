# Chat Service

A real-time WebSocket-based chat built with Go, featuring MongoDB, Redis and Swagger documentation.

 
## Getting Started
To get started with this project, you need to install Go. Download it [*here*](https://go.dev/). Then, you can clone this repository and run the following commands:

```bash
go mod tidy
```

```bash
go run cmd/api/main.go
```

## API Documentation
This project uses Swagger to document the API. The code generation was made using [*Swag*](https://github.com/swaggo/swag). After updating the documentation - by adding/editing comments in the code - you can run the following command to generate the documentation:
```bash
swag init -d ./cmd/api/,./
```

It will update the `docs` folder with the new documentation. You can access the documentation by running the project and accessing the `/swagger/index.html` endpoint *http://localhost:8080/swagger/index.html*.

## Environment Variables
You can check the environment variables needed to run this project in the `.env.example` file. Run the following command to create a `.env` file:
```bash
cp .env.example .env
```
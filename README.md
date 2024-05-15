# Znvo Backend - Final Year Project

This is backend for the Znvo app. It is written in Go using the Connect.build library for the server. This project is a part of the final year project at the Northumbria University. The project is a part of the BSc Computer Science with Web Development course. The in-depth description of the project and its features can be found in the text report which has been submitted via Turnitin.

## Author

Filip Brebera - w21020340

## Prerequisites

- Go 1.22 or higher
- Make
- Air (for hot reloading)

## Setup

Make sure to install the dependencies:

```bash
# bun
go mod download
```

You will need to create a `.env` file in the root of the project with the following content:

`.env` file

```
PORT=40000 # Port on which the server will run
JWT_SECRET= # Secret private key string for the JWT token encryption
REDIS_URL= # url for the Redis database to store the sessions
OPENAI_API_KEY= # API key for the OpenAI API
GCP_CREDENTIALS= # JSON service account key for the KMS service
SENTRY_DSN= # DSN for the Sentry error tracking
TURSO_DATABASE_URL= # URL for the Turso database
TURSO_AUTH_TOKEN= # Auth token for the Turso database
```

These values are secret as they contain information that could lead to a security breach if exposed. These values are automatically loaded into the environment in the production environment on Fly.io. If you need to use the app in the development environment, please send me a message so I can provide you with the values.

## Development Server

Start the development server on `http://localhost:40000`:

```bash
# bun
make watch
```

To generate new pair of keys for the JWT token, you can use the following command:

```bash
make key
```

To regenerate the code based on the Protobuf files, you can use the following command:

```bash
make protogen
```

## Production

The process for deployment to the production in `.github/workflows/production.yml` is automated when a new commit is pushed to the `main` branch. The process is as follows:

1. Build the docker image based on the Dockerfile in the root of the repository
2. The image is deployed to Fly.io

To deploy the app manually, you can use the following command:

```bash
docker build -t znvo-backend .
```

```bash
docker run -p 40000:40000 znvo-backend
```

Then you can access the API on `http://localhost:40000`. The API is based on gRPC, therefore you can use Postman or any other tool that supports gRPC.

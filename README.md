# modwithfriends

This repository contains the source code for the `modwithfriends` bot and several Web APIs for administratives and the [fwens website](https://amazing-banach-0667a3.netlify.app).

## Meta

- [Quick Overview of Go](https://gobyexample.com/)
- [Unit Testing in Go](https://quii.gitbook.io/learn-go-with-tests/)
- [Project Structure](https://www.gobeyond.dev/standard-package-layout/)

## Preliminaries

- Go >=1.12.17
- PostgreSQL

## Setup

In project root, run the command below and update `.env` with your environment variables.

```
cp .env.example .env
```

To run the project locally

```
go run cmd/modwithfriends/main.go
```

## Deployment (to Heroku)

**Note: As bot is using polling instead of Webhooks, server instance must be kept alive 24/7.**

1. Create a Heroku project and addon a PostgreSQL database.
2. Find the database connection string in the project's config vars and connect to the database via a client (e.g. `psql`, `pgAdmin`, etc).
3. Execute `schema.sql` in the client to create the necessary tables in the database.
4. Provide production environment variables via the project's config vars and also include
   - `DEPLOYMENT_TYPE` = `production`
   - `GIN_MODE` = `release`
5. Connect repository to Heroku and perform deployment; the `Procfile` would instruct them on the directory to build the project and deploy the binaries from.

## Instructions

1. Set up a Postman collection with the given JSON file.
2. Once deployed to Heroku, query the database (https://modwithfriends.herokuapp.com/api/v0/groups/incomplete) for incomplete groups using the GET request.
3. If there's an incomplete group, proceed to create a Telegram group with bot.
4. Copy the Telegram invite link and use the PATCH request to update the group's invite link in the database (https://modwithfriends.herokuapp.com/api/v0/groups/group-id). A message will automatically be sent to the group members with the invite link.

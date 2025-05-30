# Bank Transfer Temporal Workflow

Just playing around with Temporal to build a simple bank transfer system in Go. It uses workflows to handle money transfers, PostgreSQL to store data, and NATS to publish events. Added a REST API with Gin and some Prometheus metrics for kicks. Here's how to get it running.

-Checks if accounts are valid
-Moves money from one account to another
-Logs the transaction
-Sends out event notifications

## Tech Stack
- Temporal : for workflows
- PostgreSQL : Stores accounts and transactions
- NATS: Event publishing
- Gin : REST API
- Prometheus : Metrics
- GORM : Database stuff
- Zerolog : Logging

## Prerequistes 
- Go: 1.20 or Higher
- Docker and Docker compose : for temportal , NATS and PostgreSQL

## Running It
1. Fire Up
``` bash
go run cmd/main.go
```
This starts the Temporal worker and REST API on port 8080, connecting to PostgreSQL and NATS.
( Make sure you have the services running and added the dependencies)

2. Test
``` bash
curl -X POST -H "Content-Type: application/json" -d '{"from_account_id":"acc1","to_account_id":"acc2","amount":100}' http://localhost:8080/transfer
```
Try a transfer

3. Check status: Use the workflow_id from the transfer response
``` bash
curl http://localhost:8080/status/<workflow_id>
```
## Future Plans
Well its not perfect , if you run into some errors raise an issue , Contributions are welcomed.

Thinking to put it on K8s
Complete a basic UI for transfer and add endpoints to check account balances

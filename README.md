# AgentRAG

## Local PostgreSQL

Start a local PostgreSQL instance for repository smoke tests:

```bash
make postgres-up
make test-postgres
```

Run the API against that database:

```bash
AGENTRAG_CONFIG=configs/config.postgres.example.yaml make run-api
```

Stop the local database:

```bash
make postgres-down
```

Recreate the database volume after changing init SQL:

```bash
make postgres-reset
```

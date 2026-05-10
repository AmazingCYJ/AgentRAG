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

PostgreSQL SQL files live under `resources/database`:

- `schema_pg.sql`: fresh database schema.
- `init_data_pg.sql`: bootstrap data.
- `upgrade_v1.0_to_v1.1.sql`: manual upgrade from older repository-created tables to the current schema shape.

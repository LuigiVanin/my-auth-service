## Databae Migration

Use the Make commands helpers:

[!warning] For this to work it is necessary to install de [go-migrate](https://github.com/golang-migrate/migrate) executable

```sh
# Create Migration file(up and down files)
make create-migration first-migration

# Run UP command of the migration 
make migrate up

# Run the DOWN command of the migration
make migrate down

# Start database with ADMIN app and ADMIN user
make init
```
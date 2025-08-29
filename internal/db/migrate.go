package db

import (
	"context"
	"embed"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

func RunMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	files, err := migrationsFS.ReadDir("migrations")
	if err != nil { return err }
	_, err = pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (version text PRIMARY KEY)`)
	if err != nil { return err }

	for _, f := range files {
		name := f.Name()
		if !strings.HasSuffix(name, ".up.sql") { continue }

		var exists bool
		if err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version=$1)`, name).Scan(&exists); err != nil {
			return err
		}
		if exists { continue }

		b, err := migrationsFS.ReadFile("migrations/" + name)
		if err != nil { return err }

		if _, err := pool.Exec(ctx, string(b)); err != nil { return err }
		if _, err := pool.Exec(ctx, `INSERT INTO schema_migrations(version) VALUES($1)`, name); err != nil { return err }
	}
	return nil
}

package product_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/google/go-cmp/cmp"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/peterldowns/pgtestdb"
	"github.com/peterldowns/pgtestdb/migrators/golangmigrator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hallabro/lightning-fast-database-tests/internal/migrations"
	"github.com/hallabro/lightning-fast-database-tests/pkg/product"
)

const (
	postgresTMPFSFSyncOffPort = "5432"
	postgresFSyncOffPort      = "5433"
	postgresPort              = "5434"
	postgresTMPTFSPort        = "5435"

	numberOfProducts = 1000
)

func setupPGTestDBTest(t *testing.T, ctx context.Context, port string) (*product.Repo, *pgxpool.Pool) {
	t.Helper()

	dbconfig := pgtestdb.Config{
		Database:   "postgres",
		DriverName: "pgx",
		User:       "postgres",
		Password:   "password",
		Host:       "localhost",
		Port:       port,
		Options:    "sslmode=disable",
	}

	gm := golangmigrator.New(".", golangmigrator.WithFS(migrations.MigrationsFS))
	cfg := pgtestdb.Custom(t, dbconfig, gm)

	pool, err := pgxpool.New(ctx, cfg.URL())
	require.NoError(t, err)

	return product.NewRepo(pool), pool
}

type testCase struct {
	name     string
	parallel bool
	dbPort   string
	setup    func(*testing.T, context.Context, string) (*product.Repo, *pgxpool.Pool)
	tearDown func(*testing.T, context.Context, *pgxpool.Pool) func()
}

// I recommend running one test case at a time. Some of these tests run in a mounted tmpfs meaning you may quickly
// run out of memory if you run them all at once.
func TestRepo_CreateAndList(t *testing.T) {
	testCases := []testCase{
		{
			name:     "pgtestdb, parallel execution (TMPFS, fsync=off)",
			parallel: true,
			dbPort:   postgresTMPFSFSyncOffPort,
			setup:    setupPGTestDBTest,
		},
		{
			name:     "pgtestdb, parallel execution (fsync=off)",
			parallel: true,
			dbPort:   postgresFSyncOffPort,
			setup:    setupPGTestDBTest,
		},
		{
			name:     "pgtestdb, parallel execution (TMPFS)",
			parallel: true,
			dbPort:   postgresTMPTFSPort,
			setup:    setupPGTestDBTest,
		},
		{
			name:     "pgtestdb, parallel execution (no other performance optimizations)",
			parallel: true,
			dbPort:   postgresPort,
			setup:    setupPGTestDBTest,
		},
		{
			name:     "sequential execution",
			parallel: false,
			dbPort:   postgresPort,
			setup: func(t *testing.T, ctx context.Context, port string) (*product.Repo, *pgxpool.Pool) {
				t.Helper()

				const connString = "postgres://postgres:password@localhost:%v/postgres?sslmode=disable"

				mfs, err := iofs.New(migrations.MigrationsFS, ".")
				require.NoError(t, err)

				m, err := migrate.NewWithSourceInstance("iofs", mfs, fmt.Sprintf(connString, port))
				require.NoError(t, err)

				err = m.Up()
				require.ErrorIs(t, err, migrate.ErrNoChange)

				pool, err := pgxpool.New(ctx, fmt.Sprintf(connString, port))
				require.NoError(t, err)

				return product.NewRepo(pool), pool
			},
			tearDown: func(t *testing.T, ctx context.Context, pool *pgxpool.Pool) func() {
				return func() {
					_, err := pool.Exec(ctx, "DELETE FROM products")
					require.NoError(t, err)
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			for i := range numberOfProducts {
				t.Run(fmt.Sprintf("product %d", i+1), func(t *testing.T) {
					if tc.parallel {
						t.Parallel()
					}

					ctx := context.Background()

					productRepo, pool := tc.setup(t, ctx, tc.dbPort)
					t.Cleanup(pool.Close)

					if tc.tearDown != nil {
						t.Cleanup(tc.tearDown(t, ctx, pool))
					}

					p := &product.Product{
						Name:        "A Product",
						Description: "A product description",
					}

					err := productRepo.Create(ctx, p)
					require.NoError(t, err)
					assert.NotZero(t, p.ID)

					listedProducts, err := productRepo.List(ctx)
					require.NoError(t, err)

					// This assertion will fail if tests are leaking and not cleaning up
					require.Len(t, listedProducts, 1)
					assert.Empty(t, cmp.Diff(p, listedProducts[0]))
				})
			}
		})
	}
}

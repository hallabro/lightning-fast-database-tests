package product

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repo struct {
	pgx *pgxpool.Pool
}

func NewRepo(pgx *pgxpool.Pool) *Repo {
	return &Repo{
		pgx: pgx,
	}
}

func (r *Repo) Create(ctx context.Context, product *Product) error {
	tx, err := r.pgx.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) // nolint:errcheck

	row := tx.QueryRow(ctx,
		"INSERT INTO products (name, description) VALUES ($1, $2) RETURNING id",
		product.Name,
		product.Description)

	err = row.Scan(&product.ID)
	if err != nil {
		return fmt.Errorf("failed to create product: %w", err)
	}

	err = tx.Commit(ctx)
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (r *Repo) Get(ctx context.Context, id int) (*Product, error) {
	var product Product
	err := r.pgx.QueryRow(ctx,
		`SELECT p.id, p.name, p.description
		FROM products p
		WHERE p.id = $1`, id).
		Scan(&product.ID, &product.Name, &product.Description)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errors.New("product not found")
		}
		return nil, fmt.Errorf("failed to get product: %w", err)
	}

	return &product, nil
}

func (r *Repo) List(ctx context.Context) ([]*Product, error) {
	rows, err := r.pgx.Query(ctx,
		`SELECT p.id, p.name, p.description
		FROM products p`)
	if err != nil {
		return nil, fmt.Errorf("failed to list products: %w", err)
	}
	defer rows.Close()

	products := make([]*Product, 0)
	for rows.Next() {
		var product Product
		err = rows.Scan(&product.ID, &product.Name, &product.Description)
		if err != nil {
			return nil, fmt.Errorf("failed to scan product: %w", err)
		}
		products = append(products, &product)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating products: %w", err)
	}

	return products, nil
}

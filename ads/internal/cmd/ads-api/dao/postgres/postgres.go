package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
	adsdao "github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/dao"
	"github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/domain"
)

const (
	adColumns               = `id, title, price_cents, photo_url, user_id::text, created_at`
	maxAdsPerRolling24Hours = 5
)

// DAO is a PostgreSQL-backed ad store.
type DAO struct {
	db *sql.DB
	// tx is set when the DAO is scoped to a transaction opened by WithTx.
	tx *sql.Tx
}

// queryRunner is the subset of *sql.DB and *sql.Tx the queries need.
type queryRunner interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// New opens a connection pool to the given DSN and checks that the database is reachable. The caller is responsible for calling Close.
func New(ctx context.Context, dsn string) (*DAO, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("unable to open database: %w", err)
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()

		return nil, fmt.Errorf("unable to reach database: %w", err)
	}

	return &DAO{db: db}, nil
}

// Close releases the connection pool.
func (d *DAO) Close() error {
	if err := d.db.Close(); err != nil {
		return fmt.Errorf("unable to close database: %w", err)
	}

	return nil
}

// runner returns the transaction the DAO is scoped to, or the connection pool.
func (d *DAO) runner() queryRunner {
	if d.tx != nil {
		return d.tx
	}

	return d.db
}

// Begin opens a transaction. Every call made on the returned Tx is part of it, and the caller must end it with Commit or Rollback.
func (d *DAO) Begin(ctx context.Context) (adsdao.Tx, error) {
	if d.tx != nil {
		return nil, errors.New("unable to open a nested transaction")
	}

	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("unable to begin transaction: %w", err)
	}

	return &DAO{db: d.db, tx: tx}, nil
}

// Commit applies every change made through the transaction.
func (d *DAO) Commit() error {
	if d.tx == nil {
		return errors.New("unable to commit outside a transaction")
	}

	if err := d.tx.Commit(); err != nil {
		return fmt.Errorf("unable to commit transaction: %w", err)
	}

	return nil
}

// Rollback discards every change made through the transaction. Calling it on an already committed transaction does nothing, so `defer func() { _ = tx.Rollback() }()` is a safe guard.
func (d *DAO) Rollback() error {
	if d.tx == nil {
		return errors.New("unable to roll back outside a transaction")
	}

	if err := d.tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
		return fmt.Errorf("unable to roll back transaction: %w", err)
	}

	return nil
}

// SearchAds returns all ads matching the filters.
func (d *DAO) SearchAds(ctx context.Context, req adsdao.SearchAdsRequest) ([]domain.Ad, error) {
	const query = `
		SELECT ` + adColumns + `
		FROM ads
		WHERE ($1 = '' OR user_id::text = $1)
		  AND ($2 = 0 OR price_cents >= $2)
		  AND ($3 = 0 OR price_cents <= $3)
		  AND ($4 = '' OR title ILIKE '%' || $4 || '%')`

	rows, err := d.runner().QueryContext(ctx, query,
		req.OwnerID,
		req.MinPriceCents,
		req.MaxPriceCents,
		req.Title,
	)
	if err != nil {
		return nil, fmt.Errorf("unable to search ads: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var ads []domain.Ad
	for rows.Next() {
		var ad domain.Ad
		if err := rows.Scan(&ad.ID, &ad.Title, &ad.Price, &ad.PhotoURL, &ad.UserID, &ad.CreatedAt); err != nil {
			return nil, fmt.Errorf("unable to scan ad: %w", err)
		}

		ads = append(ads, ad)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("unable to read ads: %w", err)
	}

	return ads, nil
}

// GetAdByID returns a single ad, or dao.ErrNotFound.
func (d *DAO) GetAdByID(ctx context.Context, id domain.AdID) (domain.Ad, error) {
	const query = `SELECT ` + adColumns + ` FROM ads WHERE id = $1`

	ad, err := scanAd(d.runner().QueryRowContext(ctx, query, int64(id)))
	if err != nil {
		return domain.Ad{}, fmt.Errorf("unable to get ad: %w", err)
	}

	return ad, nil
}

// DeleteAd removes an ad owned by ownerID.
func (d *DAO) DeleteAd(ctx context.Context, id domain.AdID, ownerID domain.UserID) error {
	const query = `DELETE FROM ads WHERE id = $1 AND user_id = $2::uuid`

	result, err := d.runner().ExecContext(ctx, query, int64(id), string(ownerID))
	if err != nil {
		return fmt.Errorf("unable to delete ad: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("unable to determine whether ad was deleted: %w", err)
	}
	if rowsAffected == 0 {
		return adsdao.ErrNotFound
	}

	return nil
}

// CreateAd inserts an ad and returns its generated identifier.
func (d *DAO) CreateAd(ctx context.Context, req adsdao.CreateAdRequest) (domain.AdID, error) {
	const query = `
		INSERT INTO ads (title, price_cents, photo_url, user_id)
		VALUES ($1, $2, $3, $4::uuid)
		RETURNING id`

	var id domain.AdID
	err := d.runner().QueryRowContext(ctx, query, req.Title, req.PriceCents, req.PhotoURL, string(req.OwnerID)).Scan(&id)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return 0, fmt.Errorf("unable to create ad: %w", adsdao.ErrUnknownOwner)
		}

		return 0, fmt.Errorf("unable to create ad: %w", err)
	}

	return id, nil
}

// CreateAdWithQuota serializes writes per owner and enforces the rolling 24-hour limit.
func (d *DAO) CreateAdWithQuota(ctx context.Context, req adsdao.CreateAdRequest) (domain.AdID, error) {
	if d.tx != nil {
		return 0, errors.New("unable to enforce ad quota inside an existing transaction")
	}

	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("unable to begin ad creation: %w", err)
	}
	rollback := func(cause error) error {
		if rollbackErr := tx.Rollback(); rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) {
			return errors.Join(cause, fmt.Errorf("unable to roll back ad creation: %w", rollbackErr))
		}
		return cause
	}

	var locked int
	err = tx.QueryRowContext(ctx,
		`SELECT 1 FROM (SELECT pg_advisory_xact_lock(hashtextextended($1, 0))) AS owner_lock`,
		string(req.OwnerID),
	).Scan(&locked)
	if err != nil {
		return 0, rollback(fmt.Errorf("unable to lock ad quota: %w", err))
	}

	var count int64
	err = tx.QueryRowContext(ctx,
		`SELECT count(*) FROM ad_publications WHERE user_id = $1::uuid AND published_at > clock_timestamp() - INTERVAL '24 hours'`,
		string(req.OwnerID),
	).Scan(&count)
	if err != nil {
		return 0, rollback(fmt.Errorf("unable to check ad quota: %w", err))
	}
	if count >= maxAdsPerRolling24Hours {
		return 0, rollback(adsdao.ErrQuotaExceeded)
	}

	const query = `
		INSERT INTO ads (title, price_cents, photo_url, user_id, created_at)
		VALUES ($1, $2, $3, $4::uuid, clock_timestamp())
		RETURNING id`

	var id domain.AdID
	err = tx.QueryRowContext(ctx, query, req.Title, req.PriceCents, req.PhotoURL, string(req.OwnerID)).Scan(&id)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			err = fmt.Errorf("unable to create ad: %w", adsdao.ErrUnknownOwner)
		} else {
			err = fmt.Errorf("unable to create ad: %w", err)
		}
		return 0, rollback(err)
	}

	_, err = tx.ExecContext(ctx,
		`INSERT INTO ad_publications (user_id, published_at) VALUES ($1::uuid, clock_timestamp())`,
		string(req.OwnerID),
	)
	if err != nil {
		return 0, rollback(fmt.Errorf("unable to record ad publication: %w", err))
	}

	if err := tx.Commit(); err != nil {
		return 0, rollback(fmt.Errorf("unable to commit ad creation: %w", err))
	}

	return id, nil
}

func scanAd(row *sql.Row) (domain.Ad, error) {
	var ad domain.Ad
	if err := row.Scan(&ad.ID, &ad.Title, &ad.Price, &ad.PhotoURL, &ad.UserID, &ad.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Ad{}, adsdao.ErrNotFound
		}

		return domain.Ad{}, err
	}

	return ad, nil
}

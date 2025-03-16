package korm

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
)

// Transaction represents a database transaction
type Transaction struct {
	orm    *ORM
	tx     *sqlx.Tx
	closed bool
}

// Begin starts a new database transaction
func (o *ORM) Begin() (*Transaction, error) {
	// Start a new transaction
	tx, err := o.db.Beginx()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Create a copy of the ORM with the transaction
	txOrm := &ORM{
		db:        o.db,
		tableName: o.tableName,
		tx:        tx,
	}

	return &Transaction{
		orm:    txOrm,
		tx:     tx,
		closed: false,
	}, nil
}

// BeginTx starts a new database transaction with options
func (o *ORM) BeginTx(ctx context.Context, opts *sql.TxOptions) (*Transaction, error) {
	// Start a new transaction with context and options
	tx, err := o.db.BeginTxx(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Create a copy of the ORM with the transaction
	txOrm := &ORM{
		db:        o.db,
		tableName: o.tableName,
		tx:        tx,
	}

	return &Transaction{
		orm:    txOrm,
		tx:     tx,
		closed: false,
	}, nil
}

// Commit commits the transaction
func (t *Transaction) Commit() error {
	if t.closed {
		return fmt.Errorf("transaction already closed")
	}

	err := t.tx.Commit()
	t.closed = true
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// Rollback rolls back the transaction
func (t *Transaction) Rollback() error {
	if t.closed {
		return fmt.Errorf("transaction already closed")
	}

	err := t.tx.Rollback()
	t.closed = true
	if err != nil {
		return fmt.Errorf("failed to rollback transaction: %w", err)
	}

	return nil
}

// ORM returns the ORM instance associated with this transaction
func (t *Transaction) ORM() *ORM {
	return t.orm
}

// Add ExecuteRaw for non-query operations (INSERT, UPDATE, DELETE)
func (o *ORM) ExecuteRaw(query string, args ...any) (sql.Result, error) {
	// Use transaction if available, otherwise use the database connection
	if o.tx != nil {
		return o.tx.Exec(query, args...)
	}
	return o.db.Exec(query, args...)
}

// Add a helper function for transaction execution with automatic rollback on error
func (o *ORM) WithTransaction(fn func(*ORM) error) error {
	// Start a transaction
	tx, err := o.Begin()
	if err != nil {
		return err
	}

	// Ensure the transaction is rolled back if anything goes wrong
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback() // Ignore rollback error on panic
			panic(r)          // Re-throw the panic
		}
	}()

	// Execute the provided function with the transaction ORM
	err = fn(tx.ORM())
	if err != nil {
		// Roll back the transaction if the function returns an error
		rollbackErr := tx.Rollback()
		if rollbackErr != nil {
			return fmt.Errorf("execution error: %v, rollback error: %v", err, rollbackErr)
		}
		return err
	}

	// Commit the transaction if everything went well
	return tx.Commit()
}

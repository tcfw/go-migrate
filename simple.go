package migrate

import (
	"database/sql"
	"time"
)

//MigrationIncrement applies an increment to the DB
type MigrationIncrement func(tx *sql.Tx) error

//SimpleMigration a simple struct where the up and down functions can be assigned by attributes
type SimpleMigration struct {
	name string
	date time.Time

	up   MigrationIncrement
	down MigrationIncrement
}

//NewSimpleMigration helper func for quickly declaring a simple migration
func NewSimpleMigration(name string, date time.Time, up, down MigrationIncrement) *SimpleMigration {
	return &SimpleMigration{
		date: date,
		name: name,
		up:   up,
		down: down,
	}
}

//Up the apply increment
func (tm *SimpleMigration) Up(tx *sql.Tx) error {
	if tm.up != nil {
		return tm.up(tx)
	}

	return nil
}

//Down the rollback decrement
func (tm *SimpleMigration) Down(tx *sql.Tx) error {
	if tm.down != nil {
		return tm.down(tx)
	}

	return nil
}

//Date which the migration was created (not applied)
func (tm *SimpleMigration) Date() time.Time { return tm.date }

//Name provides a human readable name
func (tm *SimpleMigration) Name() string { return tm.name }

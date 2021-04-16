package pgx

import (
	"fmt"
	"sort"
	"time"

	"github.com/jackc/pgx"
	"github.com/sirupsen/logrus"
)

//Migration holds both up and down increments for a single migration
type Migration interface {
	Up(*pgx.Tx) error
	Down(*pgx.Tx) error
	Date() time.Time
	Name() string
}

//MigrationList a slice of migration
type MigrationList []Migration

func (ml MigrationList) Len() int           { return len(ml) }
func (ml MigrationList) Swap(i, j int)      { ml[i], ml[j] = ml[j], ml[i] }
func (ml MigrationList) Less(i, j int) bool { return ml[i].Date().Before(ml[j].Date()) }

//Migrate runs all migration up increments in date order
func Migrate(db *pgx.Conn, log *logrus.Logger, migs []Migration) error {
	if err := checkMigrationTable(db); err != nil {
		return err
	}

	toRun, err := needsToRun(db, migs)
	if err != nil {
		return err
	}

	log.WithField("n", len(toRun)).Infof("Running migrations...")

	return migrateUpN(db, log, toRun, len(toRun))
}

//migrateUpN runs N up increments
func migrateUpN(conn *pgx.Conn, log *logrus.Logger, migs []Migration, n int) error {
	tx, err := conn.Begin()
	if err != nil {
		return err
	}

	sorted := sortMigrations(migs)
	for i := 0; i < n; i++ {
		name := dbName(sorted[i])

		if err := sorted[i].Up(tx); err != nil {
			tx.Rollback()
			return fmt.Errorf("Failed to up %s: %s", name, err)
		}

		_, err := tx.Exec(`INSERT INTO migrations VALUES ($1)`, name)
		if err != nil {
			tx.Rollback()
			return err
		}

		log.Infof("Up'd %s (%d/%d)", name, i+1, n)
	}

	return tx.Commit()
}

//migrateDownN runs N down increments
func migrateDownN(conn *pgx.Conn, log *logrus.Logger, migs []Migration, n int) error {
	tx, err := conn.Begin()
	if err != nil {
		return err
	}

	sorted := sortMigrations(migs)
	for i := len(sorted) - 1; i > len(sorted)-1-n; i-- {
		name := dbName(sorted[i])

		if err := sorted[i].Down(tx); err != nil {
			tx.Rollback()
			return fmt.Errorf("Failed to down %s: %s", name, err)
		}

		_, err := tx.Exec(`DELETE FROM migrations WHERE migration = $1`, name)
		if err != nil {
			tx.Rollback()
			return err
		}

		log.Infof("Down'd %s (%d/%d)", name, i, n)
	}

	return tx.Commit()
}

//needsToRun lists which of the given migrations needs to be run
func needsToRun(conn *pgx.Conn, migs MigrationList) (MigrationList, error) {
	toRun := MigrationList{}

	hasRun := map[string]bool{}
	hasRunRes, err := conn.Query(`SELECT * FROM migrations`)
	if err != nil {
		return nil, err
	}

	for hasRunRes.Next() {
		var name string
		hasRunRes.Scan(&name)
		hasRun[name] = true
	}
	hasRunRes.Close()

	for _, migration := range migs {
		name := dbName(migration)
		if _, ok := hasRun[name]; !ok {
			toRun = append(toRun, migration)
		}
	}

	return toRun, nil
}

//sortMigrations creates a date sorted migration slice
func sortMigrations(migs MigrationList) MigrationList {
	sorted := make(MigrationList, len(migs))
	copy(sorted, migs)

	sort.Sort(sorted)
	return sorted
}

//dbName converts the migration to the name stored in the ran migrations table
func dbName(mig Migration) string {
	migStr := fmt.Sprintf("%d_%s", mig.Date().Unix(), mig.Name())

	n := len(migStr)
	if n > 500 {
		n = 500
	}

	return migStr[:n]
}

//checkMigrationTable creates the migrations table if it doesn't exist
func checkMigrationTable(conn *pgx.Conn) error {
	_, err := conn.Exec(`CREATE TABLE IF NOT EXISTS migrations (migration VARCHAR(500) NOT NULL)`)
	return err
}

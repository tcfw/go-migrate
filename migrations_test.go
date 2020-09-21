package migrate

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestDbName(t *testing.T) {
	d, err := time.Parse(time.RFC3339, "2020-09-09T20:52:05+10:00")
	if err != nil {
		t.Fatal(err)
	}
	mig := &SimpleMigration{name: "b", date: d}
	name := dbName(mig)

	assert.Equal(t, "1599648725_b", name)
}

func TestTruncatedDbName(t *testing.T) {
	d, err := time.Parse(time.RFC3339, "2020-09-09T20:52:05+10:00")
	if err != nil {
		t.Fatal(err)
	}

	longName := ""
	for i := 0; i < 100; i++ {
		longName += "abcdefghijklmnopqrstuwxyz"
	}

	mig := &SimpleMigration{name: longName, date: d}
	name := dbName(mig)

	assert.Equal(t, ("1599648725_" + longName)[:500], name)
}

func TestSortMigrations(t *testing.T) {
	list := MigrationList{
		&SimpleMigration{name: "a", date: time.Now().Add(5 * time.Second)},
		&SimpleMigration{name: "b", date: time.Now()},
	}
	sorted := sortMigrations(list)

	assert.Equal(t, "b", sorted[0].Name())
	assert.Equal(t, "a", sorted[1].Name())
}

func TestMigrateUpN(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}

	mock.ExpectBegin()
	mock.ExpectExec(`CREATE users`).WillReturnResult(driver.ResultNoRows)
	mock.ExpectExec(`INSERT INTO migrations`).WithArgs("1599691380_a").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	ti, _ := time.Parse(time.RFC3339, "2020-09-10T08:43:00+10:00")
	list := MigrationList{
		&SimpleMigration{name: "a", date: ti,
			up: func(tx *sql.Tx) error {
				_, err := tx.Exec(`CREATE users`)
				return err
			},
		},
	}

	err = migrateUpN(db, logrus.New(), list, 1)
	if err != nil {
		t.Fatal(err)
	}
}

func TestMigrateDownN(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}

	mock.ExpectBegin()
	mock.ExpectExec(`DROP TABLE users`).WillReturnResult(driver.ResultNoRows)
	mock.ExpectExec(`DELETE FROM migrations`).WithArgs("1599691380_a").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	ti, _ := time.Parse(time.RFC3339, "2020-09-10T08:43:00+10:00")
	list := MigrationList{
		&SimpleMigration{name: "a", date: ti,
			down: func(tx *sql.Tx) error {
				_, err := tx.Exec(`DROP TABLE users`)
				return err
			},
		},
	}

	err = migrateDownN(db, logrus.New(), list, 1)
	if err != nil {
		t.Fatal(err)
	}
}

func TestMigrate(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}

	mock.ExpectExec(`CREATE TABLE IF NOT EXISTS migrations`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`SELECT \* FROM migrations`).WillReturnRows(sqlmock.NewRows([]string{"migration"}).AddRow("1599691380_a"))
	mock.ExpectBegin()
	mock.ExpectExec(`DROP TABLE users`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO migrations`).WithArgs("1599691380_b").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	ti, _ := time.Parse(time.RFC3339, "2020-09-10T08:43:00+10:00")
	list := MigrationList{
		//Migration already ran
		&SimpleMigration{name: "a", date: ti,
			up: func(tx *sql.Tx) error {
				_, err := tx.Exec(`CREATE TABLE users`)
				return err
			},
		},
		//Migration to run
		&SimpleMigration{name: "b", date: ti,
			up: func(tx *sql.Tx) error {
				_, err := tx.Exec(`DROP TABLE users`)
				return err
			},
		},
	}

	err = Migrate(db, logrus.New(), list)
	if err != nil {
		t.Fatal(err)
	}
}

func TestMigrateRollback(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}

	mock.ExpectExec(`CREATE TABLE IF NOT EXISTS migrations`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`SELECT \* FROM migrations`).WillReturnRows(sqlmock.NewRows([]string{"migration"}).AddRow("1599691380_a"))
	mock.ExpectBegin()
	mock.ExpectExec(`DROP TABLE users`).WillReturnError(errors.New("something like a foreign or similar"))
	mock.ExpectRollback()

	ti, _ := time.Parse(time.RFC3339, "2020-09-10T08:43:00+10:00")
	list := MigrationList{
		//Migration already ran
		&SimpleMigration{name: "a", date: ti,
			up: func(tx *sql.Tx) error {
				_, err := tx.Exec(`CREATE TABLE users`)
				return err
			},
		},
		//Migration to run
		&SimpleMigration{name: "b", date: ti,
			up: func(tx *sql.Tx) error {
				_, err := tx.Exec(`DROP TABLE users`)
				return err
			},
		},
	}

	err = Migrate(db, logrus.New(), list)
	if err == nil {
		t.Fail()
	}
}

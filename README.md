# go-migrate
[![PkgGoDev](https://pkg.go.dev/badge/github.com/tcfw/go-migrate)](https://pkg.go.dev/github.com/tcfw/go-migrate)

Stupidly simple SQL migrations

Migrations should be packages with your binary, so why not codify them!

## Example migration
```go
//your-package/migrations/2020_09_21_115238_create_users_table.go
package migrations

import (
	"database/sql"
	"time"

	"github.com/tcfw/go-migrate"
)

func init() {
	register(migrate.NewSimpleMigration(
		//Migration name to use in DB
		"create_users_table",

		//Timestamp of migration
		time.Date(2020, 9, 21, 11, 52, 38, 0, time.Local),

		//Up
		func(tx *sql.Tx) error {
			_, err := tx.Exec(`CREATE TABLE users (
				id UUID PRIMARY KEY,
				email string
			)`)
			return err
		},

		//Down
		func(tx *sql.Tx) error {
			_, err := tx.Exec(`DROP TABLE users`)
			return err
		},
	))
}

```

```go
//your-package/migrations/migrations.go
package migrations

import (
	"database/sql"
	"github.com/sirupsen/logrus"
	"github.com/tcfw/go-migrate"
)

//migs List of known migrations
var migs []migrate.Migration = []migrate.Migration{}

//register helper to register migrations from init
func register(mig migrate.Migration) {
	migs = append(migs, mig)
}

//Migrate runs migrations up (run in main or init)
func Migrate(db *sql.DB) error {
	return migrate.Migrate(db, logrus.New(), migs)
}

```

### Notes
 - File names are irrelevant which is why name and date are in the migration struct (possibly fix in future)
 - There's no auto migrate down like there is for up as increments aren't stored in groups (like in for example Laravel migrations in PHP) and it is assumed that if you are migrating down, it should probably be a manual process anyway
 - There is no DB table locking. 
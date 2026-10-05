// Package factory knows every database the app can run on and opens the one DB_DRIVER names.
//
// It sits beside storage rather than inside it: the drivers (storage/mongo, storage/mysql) import
// storage for the Store interface, so storage itself cannot import them back. main only calls Open.
package factory

import (
	"github.com/luongtran/luna/backend/internal/platform/config"
	"github.com/luongtran/luna/backend/internal/storage"
	"github.com/luongtran/luna/backend/internal/storage/mongo"
	"github.com/luongtran/luna/backend/internal/storage/mysql"
)

// Open opens the Store of the database cfg.DBDriver names. It does not contact the server (see
// storage.Opener), so the backend starts while the database is down.
func Open(cfg config.Config) (storage.Store, error) {
	return Registry(cfg).Open(cfg.DBDriver)
}

// Registry lists the databases the app can run on and what each needs from cfg. This is the only
// place that knows them all: adding a database means one more Register line.
func Registry(cfg config.Config) *storage.Registry {
	reg := storage.NewRegistry()
	reg.Register("mongo", func() (storage.Store, error) {
		return mongo.Open(cfg.MongoURI, cfg.MongoDatabase)
	})
	reg.Register("mysql", func() (storage.Store, error) {
		m := cfg.MySQL
		return mysql.Open(mysql.Options{
			DSN: m.DSN, Host: m.Host, Port: m.Port, User: m.User, Password: m.Password, Database: m.Database,
			MaxOpenConns: m.MaxOpenConns, MaxIdleConns: m.MaxIdleConns,
			ConnMaxLifetime: m.ConnMaxLifetime, DialTimeout: m.DialTimeout,
		})
	})
	return reg
}

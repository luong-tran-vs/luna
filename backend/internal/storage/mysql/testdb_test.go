package mysql

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"testing"
	"time"

	drv "github.com/go-sql-driver/mysql"
)

// testDSNEnv names the variable with a server-level DSN (a user allowed to CREATE and DROP databases), e.g.
//
//	MYSQL_TEST_DSN=root:luna@tcp(127.0.0.1:3307)/
//
// Without it the integration tests are skipped, so `go test ./...` needs no database.
const testDSNEnv = "MYSQL_TEST_DSN"

// testDB returns a connection to a new, empty database with the whole schema migrated, dropped
// when the test ends. Each test gets its own, so tests may run in parallel.
func testDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv(testDSNEnv)
	if dsn == "" {
		t.Skipf("%s is not set: skipping the MySQL integration test", testDSNEnv)
	}
	cfg, err := drv.ParseDSN(dsn)
	if err != nil {
		t.Fatalf("%s: %v", testDSNEnv, err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	admin, err := sql.Open("mysql", normalize(cfg).FormatDSN())
	if err != nil {
		t.Fatal(err)
	}
	name := "luna_test_" + newID()
	if _, err := admin.ExecContext(ctx, "CREATE DATABASE `"+name+"` CHARACTER SET utf8mb4 COLLATE utf8mb4_bin"); err != nil {
		_ = admin.Close()
		t.Fatalf("create test database: %v", err)
	}
	t.Cleanup(func() {
		_, _ = admin.Exec("DROP DATABASE `" + name + "`")
		_ = admin.Close()
	})

	own := *cfg
	own.DBName = name
	db, err := sql.Open("mysql", normalize(&own).FormatDSN())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := migrate(ctx, db, migrations()); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	// Adds the unique topic name key, as Prepare does after the migrations.
	if err := mergeSharedTopics(ctx, db, slog.New(slog.DiscardHandler)); err != nil {
		t.Fatalf("merge shared topics: %v", err)
	}
	return db
}

func TestMigrateIsIdempotent(t *testing.T) {
	t.Parallel()
	db := testDB(t)
	if err := migrate(t.Context(), db, migrations()); err != nil {
		t.Fatalf("second run: %v", err)
	}
	var n int
	if err := db.QueryRowContext(t.Context(), "SELECT COUNT(*) FROM schema_migrations").Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != len(migrations()) {
		t.Fatalf("recorded %d migrations, want %d", n, len(migrations()))
	}
}

func TestOpenNormalizesTheDSN(t *testing.T) {
	t.Parallel()
	cfg, err := drv.ParseDSN("u:pw@tcp(h:3306)/d?parseTime=false&loc=Asia%2FTokyo&multiStatements=true")
	if err != nil {
		t.Fatal(err)
	}
	got := normalize(cfg)
	if !got.ParseTime || got.Loc != time.UTC || got.MultiStatements || !got.ClientFoundRows || got.Timeout != dialTimeout {
		t.Fatalf("normalize = %+v", got)
	}
	if got.Params["charset"] != "utf8mb4" {
		t.Fatalf("charset = %q", got.Params["charset"])
	}
}

func TestOpenRejectsABadDSNWithoutEchoingIt(t *testing.T) {
	t.Parallel()
	_, err := Open(Options{DSN: "hunter2-not-a-dsn"})
	if err == nil {
		t.Fatal("Open accepted a bad DSN")
	}
	if got := err.Error(); len(got) == 0 || contains(got, "hunter2") {
		t.Fatalf("error leaks the DSN: %q", got)
	}
}

func TestOpenDoesNotContactTheServer(t *testing.T) {
	t.Parallel()
	s, err := Open(Options{DSN: "u:pw@tcp(unreachable.invalid:3306)/d"})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := s.Close(t.Context()); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func TestNewIDShape(t *testing.T) {
	t.Parallel()
	a, b := newID(), newID()
	if len(a) != 24 || a == b {
		t.Fatalf("ids = %q, %q", a, b)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func TestOptionsBuildTheDSNFromParts(t *testing.T) {
	t.Parallel()
	// The password holds characters that would break a hand-written DSN.
	o := Options{Host: "db.internal", Port: "3307", User: "luna", Password: "p@ss/w:rd?&=%", Database: "luna_prod"}
	cfg, err := o.driverConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.User != "luna" || cfg.Passwd != "p@ss/w:rd?&=%" || cfg.DBName != "luna_prod" || cfg.Net != "tcp" || cfg.Addr != "db.internal:3307" {
		t.Fatalf("config = %+v", cfg)
	}
	// Formatting and parsing again (what sql.Open does) keeps every part.
	back, err := drv.ParseDSN(normalize(cfg).FormatDSN())
	if err != nil {
		t.Fatalf("formatted DSN does not parse: %v", err)
	}
	if back.Passwd != o.Password || back.Addr != "db.internal:3307" || back.DBName != "luna_prod" || back.User != "luna" {
		t.Fatalf("round trip = %+v", back)
	}
}

func TestOptionsDefaultsAndDSNWins(t *testing.T) {
	t.Parallel()
	cfg, err := Options{User: "u", Database: "d"}.driverConfig()
	if err != nil || cfg.Addr != "localhost:3306" {
		t.Fatalf("defaults: %+v, %v", cfg, err)
	}
	cfg, err = Options{DSN: "a:b@tcp(dsnhost:1)/dsndb", Host: "other", User: "x", Database: "y"}.driverConfig()
	if err != nil || cfg.Addr != "dsnhost:1" || cfg.DBName != "dsndb" || cfg.User != "a" {
		t.Fatalf("DSN must win: %+v, %v", cfg, err)
	}
	cfg, _ = Options{Host: "::1", Port: "3306", User: "u"}.driverConfig()
	if cfg.Addr != "[::1]:3306" {
		t.Fatalf("IPv6 host = %q", cfg.Addr)
	}
}

func TestOpenAppliesThePoolAndDialSettings(t *testing.T) {
	t.Parallel()
	s, err := Open(Options{User: "u", Database: "d", MaxOpenConns: 7, MaxIdleConns: 3, ConnMaxLifetime: time.Minute, DialTimeout: 750 * time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close(t.Context()) }()
	if got := s.db.Stats().MaxOpenConnections; got != 7 {
		t.Fatalf("MaxOpenConnections = %d, want 7", got)
	}
	// Left at zero, the defaults apply.
	d, err := Open(Options{User: "u"})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = d.Close(t.Context()) }()
	if got := d.db.Stats().MaxOpenConnections; got != maxOpen {
		t.Fatalf("default MaxOpenConnections = %d, want %d", got, maxOpen)
	}
}

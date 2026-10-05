package factory

import (
	"errors"
	"strings"
	"testing"

	"github.com/luongtran/luna/backend/internal/platform/config"
	"github.com/luongtran/luna/backend/internal/storage"
	"github.com/luongtran/luna/backend/internal/storage/mongo"
	"github.com/luongtran/luna/backend/internal/storage/mysql"
)

func TestFactoryKnowsBothDatabases(t *testing.T) {
	t.Parallel()
	got := strings.Join(Registry(config.Config{}).Drivers(), ",")
	if got != "mongo,mysql" {
		t.Fatalf("Drivers() = %s", got)
	}
}

func TestFactoryOpensMongoWithoutContactingTheServer(t *testing.T) {
	t.Parallel()
	cfg := config.Config{DBDriver: "mongo", MongoURI: "mongodb://unreachable.invalid:27017", MongoDatabase: "luna"}
	s, err := Open(cfg)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if _, ok := s.(*mongo.Store); !ok {
		t.Fatalf("store is %T, want *mongo.Store", s)
	}
}

func TestFactoryRejectsAnInvalidMongoURI(t *testing.T) {
	t.Parallel()
	cfg := config.Config{DBDriver: "mongo", MongoURI: "not-a-uri"}
	if _, err := Open(cfg); err == nil {
		t.Fatal("Open succeeded with an invalid URI")
	}
}

// MySQL, like Mongo, only prepares its pool: the server is contacted later.
func TestFactoryOpensMySQLWithoutContactingTheServer(t *testing.T) {
	t.Parallel()
	cfg := config.Config{DBDriver: "mysql", MySQL: config.MySQL{DSN: "luna:pw@tcp(unreachable.invalid:3306)/luna"}}
	s, err := Open(cfg)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if _, ok := s.(*mysql.Store); !ok {
		t.Fatalf("store is %T, want *mysql.Store", s)
	}
}

// The parts of the configuration reach the MySQL store: a server that is not there is only found on use.
func TestFactoryOpensMySQLFromItsParts(t *testing.T) {
	t.Parallel()
	cfg := config.Config{DBDriver: "mysql", MySQL: config.MySQL{
		Host: "unreachable.invalid", Port: "3306", User: "luna", Password: "pw", Database: "luna", MaxOpenConns: 3,
	}}
	s, err := Open(cfg)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := s.Ping(t.Context()); err == nil {
		t.Fatal("Ping succeeded against a server that does not exist")
	}
}

func TestFactoryRejectsABadMySQLDSNWithoutLeakingIt(t *testing.T) {
	t.Parallel()
	cfg := config.Config{DBDriver: "mysql", MySQL: config.MySQL{DSN: "hunter2"}}
	_, err := Open(cfg)
	if err == nil || strings.Contains(err.Error(), "hunter2") {
		t.Fatalf("err = %v", err)
	}
}

func TestUnknownDriverIsAnError(t *testing.T) {
	t.Parallel()
	if _, err := Registry(config.Config{}).Open("postgres"); !errors.Is(err, storage.ErrUnknownDriver) {
		t.Fatalf("err = %v", err)
	}
}

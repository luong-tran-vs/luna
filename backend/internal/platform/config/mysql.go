package config

import (
	"errors"
	"strconv"
	"strings"
	"time"
)

// MySQL holds the settings of the MySQL database (DB_DRIVER=mysql). They are read, and checked,
// only when MySQL is the chosen database, so a stray variable never stops a MongoDB setup.
type MySQL struct {
	// DSN, when set, is used as it is (user:password@tcp(host:3306)/database) and Host, Port, User,
	// Password and Database are ignored. The pool settings below still apply.
	DSN      string
	Host     string
	Port     string
	User     string
	Password string
	Database string

	// MaxOpenConns and MaxIdleConns bound the connection pool; ConnMaxLifetime is how long a
	// connection is reused; DialTimeout bounds one attempt to reach the server.
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	DialTimeout     time.Duration
}

// Defaults of the MySQL settings.
const (
	defaultMySQLHost            = "localhost"
	defaultMySQLPort            = "3306"
	defaultMySQLDatabase        = "luna"
	defaultMySQLMaxOpenConns    = 20
	defaultMySQLMaxIdleConns    = 5
	defaultMySQLConnMaxLifetime = 5 * time.Minute
	defaultMySQLDialTimeout     = 2 * time.Second
)

// loadMySQL reads the MySQL variables. Every problem is returned, none includes a value.
func loadMySQL(getenv func(string) string) (MySQL, []error) {
	m := MySQL{
		DSN:      strings.TrimSpace(getenv("MYSQL_DSN")),
		Host:     valueOr(getenv("MYSQL_HOST"), defaultMySQLHost),
		Port:     valueOr(getenv("MYSQL_PORT"), defaultMySQLPort),
		User:     strings.TrimSpace(getenv("MYSQL_USER")),
		Password: getenv("MYSQL_PASSWORD"), // not trimmed: spaces may be part of a password
		Database: valueOr(getenv("MYSQL_DATABASE"), defaultMySQLDatabase),
	}

	var errs []error
	if m.DSN == "" && m.User == "" {
		errs = append(errs, errors.New("config: MYSQL_USER (or MYSQL_DSN) is required when DB_DRIVER is mysql"))
	}
	if port, err := strconv.Atoi(m.Port); err != nil || port < 1 || port > 65535 {
		errs = append(errs, errors.New("config: MYSQL_PORT must be a number from 1 to 65535"))
	}

	var err error
	if m.MaxOpenConns, err = positiveInt(getenv("MYSQL_MAX_OPEN_CONNS"), defaultMySQLMaxOpenConns); err != nil {
		errs = append(errs, errors.New("config: MYSQL_MAX_OPEN_CONNS must be a whole number of at least 1"))
	}
	if m.MaxIdleConns, err = positiveInt(getenv("MYSQL_MAX_IDLE_CONNS"), defaultMySQLMaxIdleConns); err != nil {
		errs = append(errs, errors.New("config: MYSQL_MAX_IDLE_CONNS must be a whole number of at least 1"))
	}
	if m.ConnMaxLifetime, err = positiveDuration(getenv("MYSQL_CONN_MAX_LIFETIME"), defaultMySQLConnMaxLifetime); err != nil {
		errs = append(errs, errors.New("config: MYSQL_CONN_MAX_LIFETIME must be a duration such as 5m or 30s"))
	}
	if m.DialTimeout, err = positiveDuration(getenv("MYSQL_DIAL_TIMEOUT"), defaultMySQLDialTimeout); err != nil {
		errs = append(errs, errors.New("config: MYSQL_DIAL_TIMEOUT must be a duration such as 2s"))
	}
	return m, errs
}

// positiveInt parses v as an integer of at least 1; an empty v is the fallback.
func positiveInt(v string, fallback int) (int, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return fallback, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 1 {
		return fallback, errors.New("invalid")
	}
	return n, nil
}

// positiveDuration parses v as a Go duration above zero; an empty v is the fallback.
func positiveDuration(v string, fallback time.Duration) (time.Duration, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return fallback, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil || d <= 0 {
		return fallback, errors.New("invalid")
	}
	return d, nil
}

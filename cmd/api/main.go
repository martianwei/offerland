// Package classification OfferLand API.
//
// the purpose of this application is to provide an application
// that is using plain go code to define an API
//
// This should demonstrate all the possible comment annotations
// that are available to turn go code into a fully compliant swagger 2.0 spec
//
// Terms Of Service:
//
// there are no TOS at this moment, use at your own risk we take no responsibility
//
//	Schemes: http, https
//	Host: localhost
//	BasePath: /v2
//	Version: 0.0.1
//	License: MIT http://opensource.org/licenses/MIT
//	Contact: John Doe<john.doe@example.com> http://john.doe.com
//
//	Consumes:
//	- application/json
//	- application/xml
//
//	Produces:
//	- application/json
//	- application/xml
//
//	Security:
//	- api_key:
//
//	SecurityDefinitions:
//	api_key:
//	     type: apiKey
//	     name: KEY
//	     in: header
//	oauth2:
//	    type: oauth2
//	    authorizationUrl: /oauth2/auth
//	    tokenUrl: /oauth2/token
//	    in: header
//	    scopes:
//	      bar: foo
//	    flow: accessCode
//
//	Extensions:
//	x-meta-value: value
//	x-meta-array:
//	  - value1
//	  - value2
//	x-meta-array-obj:
//	  - name: obj
//	    value: field
//
// swagger:meta
package main

import (
	"database/sql"
	"flag"
	"log"
	"os"
	"path"
	"sync"

	_ "github.com/lib/pq"
	"offerland.cc/internal/database"
	"offerland.cc/internal/funcs"
	"offerland.cc/internal/leveledlog"
	"offerland.cc/internal/models"
	"offerland.cc/internal/smtp"
)

// Define a config struct to hold all the configuration settings for our application.
// For now, the only configuration settings will be the network port that we want the
// server to listen on, and the name of the current operating environment for the
// application (development, staging, production, etc.). We will read in these
// configuration settings from command-line flags when the application starts.
type config struct {
	port    int
	baseURL string
	env     string
	db      struct {
		dsn          string
		automigrate  bool
		maxOpenConns int
		maxIdleConns int
		maxIdleTime  string
	}
	jwt struct {
		secretKey string
	}
	smtp struct {
		host     string
		port     int
		username string
		password string
		from     string
	}
}

// Define an application struct to hold the dependencies for our HTTP handlers, helpers,
// and middleware. At the moment this only contains a copy of the config struct and a
// logger, but it will grow to include a lot more as our build progresses.
type application struct {
	config config
	logger *leveledlog.Logger
	models *models.Models
	db     *sql.DB
	mailer *smtp.Mailer
	wg     sync.WaitGroup
}

func main() {
	// Declare an instance of the config struct.
	var cfg config
	// Read the value of the port and env command-line flags into the config struct. We
	// default to using the port number 4000 and the environment "development" if no
	// corresponding flags are provided.
	flag.IntVar(&cfg.port, "port", 8000, "API server port")
	flag.StringVar(&cfg.baseURL, "base-url", funcs.LoadEnv("BASE_URL"), "base URL for the application")
	flag.StringVar(&cfg.env, "env", "development", "Environment (development|staging|production)")
	flag.StringVar(&cfg.db.dsn, "db-dsn", funcs.LoadEnv("OFFERLAND_DB_DSN"), "PostgreSQL DSN")
	// Read the connection pool settings from command-line flags into the config struct.
	// Notice the default values that we're using?
	flag.IntVar(&cfg.db.maxOpenConns, "db-max-open-conns", 25, "PostgreSQL max open connections")
	flag.IntVar(&cfg.db.maxIdleConns, "db-max-idle-conns", 25, "PostgreSQL max idle connections")
	flag.StringVar(&cfg.db.maxIdleTime, "db-max-idle-time", "15m", "PostgreSQL max connection idle time")

	flag.StringVar(&cfg.jwt.secretKey, "jwt-secret-key", funcs.LoadEnv("JWT_SECRET"), "secret key for JWT authentication")

	flag.StringVar(&cfg.smtp.host, "smtp-host", "smtp.gmail.com", "smtp host")
	flag.IntVar(&cfg.smtp.port, "smtp-port", 587, "smtp port")
	flag.StringVar(&cfg.smtp.username, "smtp-username", funcs.LoadEnv("SMTP_USERNAME"), "smtp username")
	flag.StringVar(&cfg.smtp.password, "smtp-password", funcs.LoadEnv("SMTP_PASSWORD"), "smtp password")
	flag.StringVar(&cfg.smtp.from, "smtp-from", "OfferLand <contact@offerland.cc>", "smtp sender")

	flag.Parse()

	var f *os.File
	currDir, _ := os.Getwd()
	logPath := path.Join(currDir, "logger.log")
	f, err := os.OpenFile(logPath, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0666)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	// Initialize a new logger which writes messages to the standard out stream,
	// prefixed with the current date and time.

	// logger := leveledlog.NewJSONLogger(f, leveledlog.LevelAll)
	logger := leveledlog.NewLogger(os.Stdout, leveledlog.LevelAll, true)

	db, err := database.New(cfg.db.dsn, cfg.db.automigrate, cfg.db.maxOpenConns, cfg.db.maxIdleConns, cfg.db.maxIdleTime)
	if err != nil {
		logger.Fatal(err)
	}
	// Defer a call to db.Close() so that the connection pool is closed before the
	// main() function exits.
	defer db.Close()

	mailer := smtp.NewMailer(cfg.smtp.host, cfg.smtp.port, cfg.smtp.username, cfg.smtp.password, cfg.smtp.from)

	// Declare an instance of the application struct, containing the config struct and
	// the logger.
	app := &application{
		config: cfg,
		logger: logger,
		models: models.NewModels(db),
		db:     db,
		mailer: mailer,
	}

	err = app.serve()
	if err != nil {
		logger.Fatal(err)
	}
}

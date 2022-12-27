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
	"context"
	"database/sql"
	"log"
	"os"
	"path"
	"sync"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	_ "github.com/lib/pq"
	"google.golang.org/api/option"
	"offerland.cc/configs"
	"offerland.cc/internal/database"
	"offerland.cc/internal/leveledlog"
	"offerland.cc/internal/models"
	"offerland.cc/internal/smtp"
)

// Define a config struct to hold all the configuration settings for our application.
// For now, the only configuration settings will be the network port that we want the
// server to listen on, and the name of the current operating environment for the
// application (development, staging, production, etc.). We will read in these
// configuration settings from command-line flags when the application starts.

// Define an application struct to hold the dependencies for our HTTP handlers, helpers,
// and middleware. At the moment this only contains a copy of the config struct and a
// logger, but it will grow to include a lot more as our build progresses.
type application struct {
	config         *configs.Config
	logger         *leveledlog.Logger
	models         *models.Models
	firebaseClient *auth.Client
	db             *sql.DB
	mailer         *smtp.Mailer
	wg             sync.WaitGroup
}

func main() {
	// Declare an instance of the config struct.
	cfg, err := configs.LoadConfig(".")
	if err != nil {
		log.Fatal(err)
	}
	// Read the value of the port and env command-line flags into the config struct. We
	// default to using the port number 8080 and the environment "development" if no
	// corresponding flags are provided.

	var f *os.File
	currDir, _ := os.Getwd()
	logPath := path.Join(currDir, "logger.log")
	f, err = os.OpenFile(logPath, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0666)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	// Initialize a new instance of the leveledlog.Logger
	// Write logs to file
	// logger := leveledlog.NewJSONLogger(f, leveledlog.LevelAll)
	// Write logs to stdout
	logger := leveledlog.NewLogger(os.Stdout, leveledlog.LevelAll, true)

	db, err := database.New(cfg.DB_DSN, cfg.DB_AUTOMIGRATE, cfg.DB_MAX_OPEN_CONNS, cfg.DB_MAX_IDLE_CONNS, cfg.DB_MAX_IDLE_TIME)
	if err != nil {
		logger.Fatal(err)
	}
	// Defer a call to db.Close() so that the connection pool is closed before the
	// main() function exits.
	defer db.Close()

	mailer := smtp.NewMailer(cfg.SMTP_HOST, cfg.SMTP_PORT, cfg.SMTP_USERNAME, cfg.SMTP_PASSWORD, cfg.SMTP_FROM)

	opt := option.WithCredentialsFile(cfg.FIREBASE_CONFIG)
	firebase, err := firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		logger.Fatal(err)
	}
	firebaseClient, err := firebase.Auth(context.Background())
	if err != nil {
		log.Fatalf("error getting Auth client: %v\n", err)
	}

	app := &application{
		config:         cfg,
		logger:         logger,
		models:         models.NewModels(db),
		firebaseClient: firebaseClient,
		db:             db,
		mailer:         mailer,
	}

	// Start the HTTP server
	err = app.serve()
	if err != nil {
		logger.Fatal(err)
	}
}

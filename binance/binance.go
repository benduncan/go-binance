/*
   binance.go
       Wrapper for the Binance Exchange API

   Authors:
       Pat DePippo  <patrick.depippo@dcrypt.io>
       Matthew Woop <matthew.woop@dcrypt.io>

   To Do:

*/
package binance

import (
	"database/sql"
	"time"

	gorp "gopkg.in/gorp.v2"
)

//"errors"

// DBI string for logging verbose HTTP headers/response
var dbmap gorp.DbMap
var verboseLogging bool

const (
	BaseUrl = "https://api.binance.com"
)

type Binance struct {
	client *Client
}

type HTTPLog struct {
	TransactionID int64
	QueryURL      string
	QueryHeaders  []byte
	QueryDate     time.Time
	ResponseBody  []byte
	ResponseCode  int
	ResponseDate  time.Time
	ID            int64
}

/*
func handleErr(r jsonResponse) error {

    if !r.Success {
        return errors.New(r.Message)
    }

    return nil
}
*/
func New(key, secret, dbi string) *Binance {
	client := NewClient(key, secret)

	// DB logging enabled? Connect to the specified DBI source
	if dbi != "" {

		db, err := sql.Open("mymysql", dbi)

		// DB error? Panic
		if err != nil {
			panic(err)
		}

		// Connect to the DB and provision the DB table if missing
		dbmap = gorp.DbMap{Db: db, Dialect: gorp.MySQLDialect{"InnoDB", "UTF8"}}
		dbmap.AddTableWithName(HTTPLog{}, "HTTPLog").SetKeys(true, "ID")

		// Log all requests
		verboseLogging = true
	}

	return &Binance{client}
}

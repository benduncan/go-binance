/*
   client.go
       Wrapper for the Binance Exchange API
*/
package binance

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"time"

	_ "github.com/ziutek/mymysql/godrv"
	gorp "gopkg.in/gorp.v2"

	_ "github.com/ziutek/mymysql/native" // Native engine
)

type Client struct {
	key        string
	secret     string
	httpClient *http.Client
}

type BadRequest struct {
	code int64  `json:"code"`
	msg  string `json:"msg,required"`
}

func handleError(resp *http.Response) error {
	if resp.StatusCode != 200 {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		return fmt.Errorf("Bad response Status %s. Response Body: %s", resp.Status, string(body))
	}
	return nil
}

// Creates a new Binance HTTP Client
func NewClient(key, secret string) (c *Client) {
	client := &Client{
		key:        key,
		secret:     secret,
		httpClient: &http.Client{},
	}
	return client
}

func (c *Client) do(method, resource, payload string, auth bool, result interface{}) (resp *http.Response, err error) {

	rawLog := HTTPLog{}

	fullUrl := fmt.Sprintf("%s/%s", BaseUrl, resource)

	req, err := http.NewRequest(method, fullUrl, nil)
	if err != nil {
		return
	}

	req.Header.Add("Accept", "application/json")

	if auth {

		if len(c.key) == 0 || len(c.secret) == 0 {
			err = errors.New("Private endpoints requre you to set an API Key and API Secret")
			return
		}

		req.Header.Add("X-MBX-APIKEY", c.key)

		q := req.URL.Query()

		timestamp := time.Now().Unix() * 1000
		q.Set("timestamp", fmt.Sprintf("%d", timestamp))

		mac := hmac.New(sha256.New, []byte(c.secret))
		_, err := mac.Write([]byte(q.Encode()))
		if err != nil {
			return nil, err
		}

		signature := hex.EncodeToString(mac.Sum(nil))
		req.URL.RawQuery = q.Encode() + "&signature=" + signature
	}

	resp, err = c.httpClient.Do(req)

	if err != nil {
		return
	}

	// Response success? Log it
	if verboseLogging == true {

		db, err := sql.Open("mymysql", DBIuri)

		// DB error? Panic
		if err != nil {
			panic(err)
		}

		// Connect to the DB and provision the DB table if missing
		dbmap := gorp.DbMap{Db: db, Dialect: gorp.MySQLDialect{"InnoDB", "UTF8"}}
		dbmap.AddTableWithName(HTTPLog{}, "HTTPLog").SetKeys(true, "ID")

		rawLog.QueryURL = req.URL.String()
		rawLog.QueryDate = time.Now()

		rawLog.QueryHeaders, _ = httputil.DumpRequest(req, false)

		rawLog.ResponseCode = resp.StatusCode

		// Duplicate the body content, since ioutil.ReadAll is one-time only, and we need to replace it for the json decoder to function
		var bodyBytes []byte

		if resp.Body != nil {
			bodyBytes, _ = ioutil.ReadAll(resp.Body)
		}

		// Restore the io.ReadCloser to its original state
		resp.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))

		// Log the raw body
		rawLog.ResponseBody = bodyBytes

		// Log the response time
		rawLog.ResponseDate = time.Now()

		err2 := dbmap.Insert(&rawLog)

		if err2 != nil {
			fmt.Println("Error inserting into DB =>", err2)
		}

		defer dbmap.Db.Close()

		/*
			// File audit trail
			f, err := os.OpenFile("/tmp/binance-audit.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
			if err != nil {

			}
			defer f.Close()

			log.SetOutput(f)
			log.Println(rawLog.QueryHeaders)
			log.Println("==========")
			log.Println(bodyBytes)
		*/

	}

	// Check for error
	defer resp.Body.Close()
	err = handleError(resp)

	if err != nil {
		return
	}

	// Process response
	if resp != nil {
		decoder := json.NewDecoder(resp.Body)
		err = decoder.Decode(result)
	}

	return
}

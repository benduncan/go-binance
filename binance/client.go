/*
   client.go
       Wrapper for the Binance Exchange API
*/
package binance

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	_ "github.com/ziutek/mymysql/godrv"

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

// formatRequest generates ascii representation of a request
func formatRequest(r *http.Request) string {
	// Create return string
	var request []string
	// Add the request string
	url := fmt.Sprintf("%v %v %v", r.Method, r.URL, r.Proto)
	request = append(request, url)
	// Add the host
	request = append(request, fmt.Sprintf("Host: %v", r.Host))
	// Loop through headers
	for name, headers := range r.Header {
		name = strings.ToLower(name)
		for _, h := range headers {
			request = append(request, fmt.Sprintf("%v: %v", name, h))
		}
	}

	// If this is a POST, add post data
	if r.Method == "POST" {
		r.ParseForm()
		request = append(request, "\n")
		request = append(request, r.Form.Encode())
	}
	// Return the request as a string
	return strings.Join(request, "\n")
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

	// Log the request via the specified DBI, use as an audit trail
	if verboseLogging == true {
		rawLog.QueryURL = req.URL.String()
		rawLog.QueryDate = time.Now()

		rawLog.QueryHeaders = []byte(formatRequest(req))

		err = dbmap.Insert(&rawLog)

		if err != nil {
			return
		}

	}

	//fmt.Println("Request =>", req)
	resp, err = c.httpClient.Do(req)

	if err != nil {
		return
	}

	// Response success? Log it
	if verboseLogging == true {
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

		// Update the log to reflect the body
		dbmap.Update(&rawLog)
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

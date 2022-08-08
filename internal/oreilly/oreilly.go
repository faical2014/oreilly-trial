package oreilly

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/bilalcaliskan/oreilly-trial/internal/logging"
	"github.com/bilalcaliskan/oreilly-trial/internal/options"
	"github.com/bilalcaliskan/oreilly-trial/internal/random"

	"go.uber.org/zap"
)

var (
	logger *zap.Logger
	client *http.Client
)

func init() {
	logger = logging.GetLogger()
	client = &http.Client{}
}

// Generate does the heavy lifting, communicates with the Oreilly API
func Generate(opts *options.OreillyTrialOptions) error {
	var (
		username, password string
		jsonData           []byte
		req                *http.Request
		resp               *http.Response
		respBody           []byte
		err                error
	)

	// generate random email address from usable domains
	emailDomain := random.PickEmail(opts.EmailDomains)
	logger.Info("selected random email domain", zap.String("emailDomain", emailDomain))

	// generate random username and password
	if username, err = random.Generate(opts.UsernameRandomLength, random.TypeUsername); err != nil {
		return err
	}

	if password, err = random.Generate(opts.PasswordRandomLength, random.TypePassword); err != nil {
		return err
	}

	emailAddr := fmt.Sprintf("%s@%s", username, emailDomain)
	logger.Info("random credentials generated", zap.String("email", emailAddr), zap.String("password", password))

	// prepare json data
	values := map[string]string{
		"email":         emailAddr,
		"password":      password,
		"first_name":    "John",
		"last_name":     "Doe",
		"country":       "US",
		"t_c_agreement": "true",
		"contact":       "true",
		"trial_length":  "10",
		"path":          "/register/",
		"source":        "payments-client-register",
	}

	// marshall the json body
	if jsonData, err = json.Marshal(values); err != nil {
		return err
	}

	// prepare and make the request
	if req, err = http.NewRequest("POST", opts.CreateUserUrl, bytes.NewBuffer(jsonData)); err != nil {
		return err
	}

	logger.Debug("trying to set request headers")
	setRequestHeaders(req)

	logger.Debug("sending request with http client", zap.String("url", opts.CreateUserUrl))
	if resp, err = client.Do(req); err != nil {
		return err
	}

	defer func(body io.ReadCloser) {
		err := body.Close()
		if err != nil {
			panic(err)
		}
	}(resp.Body)

	// read the response
	if respBody, err = ioutil.ReadAll(resp.Body); err != nil {
		return err
	}

	if resp.StatusCode == 200 {
		var successResponse successResponse
		if err := json.Unmarshal(respBody, &successResponse); err != nil {
			return err
		}

		logger.Info("trial account successfully created", zap.String("email", emailAddr),
			zap.String("password", password), zap.String("user_id", successResponse.UserID))
	} else {
		return errors.New(string(respBody))
	}

	return nil
}

// setRequestHeaders gets the http.Request as input and add some headers for proper API request
func setRequestHeaders(req *http.Request) {
	req.Header.Set("authority", "learning.oreilly.com")
	req.Header.Set("pragma", "no-cache")
	req.Header.Set("cache-control", "no-cache")
	req.Header.Set("accept", "application/json")
	req.Header.Set("user-agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/88.0.4324.96 Safari/537.36")
	req.Header.Set("content-type", "application/json")
	req.Header.Set("origin", "https://learning.oreilly.com")
	req.Header.Set("sec-fetch-site", "same-origin")
	req.Header.Set("sec-fetch-mode", "cors")
	req.Header.Set("sec-fetch-dest", "empty")
	req.Header.Set("referer", "https://learning.oreilly.com/p/register/")
	req.Header.Set("accept-language", "en-US,en;q=0.9")
}

package chatgpt

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type Auth struct {
	// email represents the user's email address
	email string
	// password represents the user's password
	password string
	// authState is used to store the state of the user's authentication status
	authState string
	// apiKey stores the API key value for authentication
	apiKey string
	// accessToken stores the access token value generated after successful authentication
	accessToken string
	// expires stores the time when the access token will expire
	expires time.Time
	// enableCache is used to enable or disable caching of access tokens
	enableCache bool
	// clientStarted keeps track of whether or not the client has been started
	clientStarted bool
	// sessionName is used to store the name of the session
	sessionName string
}

// GetAccessToken generates and retrieves the OpenAI API access token by performing a series of authentication steps.
func (a *Auth) GetAccessToken() (string, error) {
	if a.enableCache {
		a.loadCachedAccessToken()
	}

	// check if the access token is already present and not expired
	if a.accessToken != "" && a.expires.After(time.Now()) {
		return a.accessToken, nil
	}

	// validate if email and password are set for authentication
	if a.email == "" || a.password == "" {
		return "", fmt.Errorf("email and password must be set to authenticate with OpenAI")
	}

	// get the callback URL after step one of authentication
	callback_url, err := a.stepOne()
	if err != nil {
		return "", err
	}

	// get the URL for step two of authentication using the obtained callback URL along with email and password
	code_url, err := a.stepTwo(callback_url, a.email, a.password)
	if err != nil {
		return "", err
	}

	// complete the final step of authentication and fetch the response containing the access token and its expiry time
	resp, err := a.stepThree(code_url)
	if err != nil {
		return "", err
	}

	// store the generated access token and its expiry in the Auth struct for future use
	a.accessToken = resp.AccessToken
	a.expires = resp.Expires
	if a.enableCache {
		a.cacheAccessToken() // cache the access token in a separate goroutine
	}

	return resp.AccessToken, nil
}

// ExpiresIn returns the remaining duration before the access token expires.
func (a *Auth) ExpiresIn() time.Duration {
	// calculate the remaining time until access token expires using current time and expiry time stored in the Auth struct
	return time.Until(a.expires)
}

type authCache struct {
	AccessToken string    `json:"access_token,omitempty"`
	Expires     time.Time `json:"expires,omitempty"`
}

func (a *Auth) cacheAccessToken() error {
	var previousData map[string]authCache
	if _, err := os.Stat("gpt-cache.json"); err == nil {
		if file, err := os.Open("gpt-cache.json"); err == nil {
			defer file.Close()
			json.NewDecoder(file).Decode(&previousData)
		}
	}
	if previousData == nil {
		previousData = make(map[string]authCache)
	}
	previousData[a.sessionName] = authCache{
		AccessToken: a.accessToken,
		Expires:     a.expires,
	}
	file, err := os.Create("gpt-cache.json")
	if err != nil {
		return err
	}
	defer file.Close()
	return json.NewEncoder(file).Encode(previousData)
}

func (a *Auth) loadCachedAccessToken() {
	if _, err := os.Stat("gpt-cache.json"); err != nil {
		return // no cache file
	}
	file, err := os.Open("gpt-cache.json")
	if err != nil {
		return // error opening file
	}
	defer file.Close()
	var data map[string]authCache
	if err := json.NewDecoder(file).Decode(&data); err != nil {
		return // error decoding file
	}
	if data == nil {
		return // no data in file
	}
	if cache, ok := data[a.sessionName]; ok {
		a.accessToken = cache.AccessToken
		a.expires = cache.Expires
	}
}

// copyCookies copies cookies from the source slice of http.Cookies to the destination http.Request.
func (a *Auth) copyCookies(from []*http.Cookie, to *http.Request) {
	// iterate over each cookie in the source slice
	for _, cookie := range from {
		// add the current cookie to the destination request
		to.AddCookie(cookie)
	}
}

// This function performs StepOne for authentication using the Auth struct provided
func (a *Auth) stepOne() (string, error) {

	// Send a GET request to the authentication endpoint given and retrieve the response
	resp, err := http.Get("https://chat-api.zhile.io/auth/endpoint")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Check if the status of the response is ok, return an error message if not
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("bad status: %s", resp.Status)
	}

	// Decode the response body into a result variable that contains 'state' and 'url'
	var result struct {
		State string `json:"state"`
		Url   string `json:"url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	// Set the Authentication state to the 'state' received in the response
	a.authState = result.State

	// Return the 'url' received in the response
	return result.Url, nil
}

// StepTwo performs authentication using the given url, email, and password.
// It follows redirects, sets appropriate headers and cookies, and returns the final redirect URL,
// or an error if any occurred during the process.
func (a *Auth) stepTwo(auth_url, _email, _password string) (string, error) {
	// create an http client with required cookie settings and redirect policy
	httpx := http.Client{
		Jar: http.DefaultClient.Jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	// prepare GET request for the specified authentication URL
	req, _ := http.NewRequest("GET", auth_url, nil)
	resp, err := httpx.Do(req)
	_ref_cookies := resp.Cookies()
	_url_prefix := "https://auth0.openai.com"
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// check if server responded with a redirect status
	if resp.StatusCode != 302 {
		return "", fmt.Errorf("bad status for url: %s", auth_url)
	}

	// extract next URL from the response header and its associated state value
	next_url := _url_prefix + resp.Header.Get("Location")
	current_state := strings.Split(strings.Split(next_url, "state=")[1], "&")[0]

	// prepare form data for POST request containing username/email as well as current state value obtained from previous step
	form_data := `state=` + current_state + `&username=` + url.QueryEscape(_email) + `&js-available=true&webauthn-available=true&is-brave=false&webauthn-platform-available=false&action=default`

	// prepare a POST request with the extracted form data and headers
	req, _ = http.NewRequest("POST", next_url, strings.NewReader(form_data))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// copy cookies from the previous response to the current request
	a.copyCookies(_ref_cookies, req)
	resp, err = httpx.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// check for correct status code, and handle incorrect email/password combination error if received
	if resp.StatusCode != 302 {
		if resp.StatusCode == 400 {
			return "", &ChatError{"email and password combination is incorrect or you have not verified your email address yet", 400}
		}
		return "", &ChatError{"bad status for url: " + next_url, resp.StatusCode}
	}

	// extract next URL from the response header and update the form data with provided password
	next_url = _url_prefix + resp.Header.Get("Location")
	form_data = `state=` + current_state + `&username=` + url.QueryEscape(_email) + `&password=` + url.QueryEscape(_password) + `&action=default`

	// prepare another POST request with the updated form data and headers
	req, _ = http.NewRequest("POST", next_url, strings.NewReader(form_data))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// copy cookies from the previous response to the current request
	a.copyCookies(_ref_cookies, req)
	resp, err = httpx.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// check for correct status code after performing final redirect
	if resp.StatusCode != 302 {
		if resp.StatusCode == 400 {
			return "", &ChatError{"email and password combination is incorrect or you have not verified your email address yet", 400}
		}
		return "", &ChatError{"bad status for url: " + next_url, resp.StatusCode}
	}

	// extract the final redirect URL and return it
	next_url = _url_prefix + resp.Header.Get("Location")
	req, _ = http.NewRequest("GET", next_url, nil)
	a.copyCookies(_ref_cookies, req)
	resp, err = httpx.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// check for correct status code after visiting the final URL
	if resp.StatusCode != 302 {
		return "", &ChatError{"bad status for url: " + next_url, resp.StatusCode}
	}
	return resp.Header.Get("Location"), nil
}

// AuthResp is a struct that represents the response returned by an authentication request.
type authResp struct {
	// AccessToken contains the access token string returned by the auth server.
	AccessToken string `json:"accessToken"`
	// Expires is a time.Time object representing the time when the access token will expire.
	Expires time.Time `json:"expires"`
	// Detail provides additional information about the authentication response, if any.
	Detail string `json:"detail"`
}

// StepThree completes the third step of the authentication process by exchanging the authorization
// code for an access token, using the provided callback URL.
func (a *Auth) stepThree(code_url string) (*authResp, error) {
	// Compose the data payload for the request.
	var data = strings.NewReader(`state=` + a.authState + `&callbackUrl=` + url.QueryEscape(code_url))

	// Create a new HTTP POST request object with the appropriate endpoint URL and data payload.
	req, _ := http.NewRequest("POST", "https://chat-api.zhile.io/auth/token", data)
	req.Header.Set("content-type", "application/x-www-form-urlencoded")

	// Send the request and obtain the response.
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Parse the response body as an AuthResp object.
	var result authResp
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	// Return the resulting AuthResp object and any error that occurred during the request/response cycle.
	return &result, nil
}

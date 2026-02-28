package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	plexTVBaseURL = "https://plex.tv"
	authAppBaseURL = "https://app.plex.tv/auth#"
)

// Pin represents a PIN code used for OAuth-style authentication with plex.tv.
type Pin struct {
	// ID is the unique identifier for this PIN.
	ID int `json:"id"`

	// Code is the PIN code the user must enter on plex.tv.
	Code string `json:"code"`

	// Product is the name of the application requesting authentication.
	Product string `json:"product"`

	// Trusted indicates if the PIN was marked as trusted.
	Trusted bool `json:"trusted"`

	// ClientIdentifier is the unique identifier of the client requesting the PIN.
	ClientIdentifier string `json:"clientIdentifier"`

	// ExpiresIn is the number of seconds until the PIN expires.
	ExpiresIn int `json:"expiresIn"`

	// AuthToken is populated with the authentication token after the user authorizes.
	AuthToken string `json:"authToken"`
}

// RequestPin requests a new PIN from plex.tv for user authentication.
// The clientIdentifier should be a unique identifier for the application instance.
func RequestPin(clientIdentifier string) (*Pin, error) {
	formValues := url.Values{
		"strong":                    {"true"},
		"X-Plex-Product":            {"Scanline"},
		"X-Plex-Client-Identifier":  {clientIdentifier},
	}

	req, err := http.NewRequest(http.MethodPost, plexTVBaseURL+"/api/v2/pins", strings.NewReader(formValues.Encode()))
	if err != nil {
		return nil, fmt.Errorf("creating pin request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing pin request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("failed to request PIN: %s", resp.Status)
	}

	var pin Pin
	if err := json.NewDecoder(resp.Body).Decode(&pin); err != nil {
		return nil, fmt.Errorf("decoding pin response: %w", err)
	}
	return &pin, nil
}

// CheckPin checks the status of a PIN and returns the updated PIN information.
// If the user has authorized the PIN, the returned Pin will have AuthToken populated.
func CheckPin(pinID int, pinCode, clientIdentifier string) (*Pin, error) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/api/v2/pins/%d", plexTVBaseURL, pinID), nil)
	if err != nil {
		return nil, fmt.Errorf("creating check request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Plex-Client-Identifier", clientIdentifier)

	params := req.URL.Query()
	params.Set("code", pinCode)
	req.URL.RawQuery = params.Encode()

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing check request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to check PIN: %s", resp.Status)
	}

	var pin Pin
	if err := json.NewDecoder(resp.Body).Decode(&pin); err != nil {
		return nil, fmt.Errorf("decoding check response: %w", err)
	}
	return &pin, nil
}

// PollPin polls for PIN authorization until the user authorizes or the context is cancelled.
// It checks the PIN status at the specified interval.
// Returns the authentication token when authorization is complete.
func PollPin(ctx context.Context, pinID int, pinCode, clientIdentifier string, interval time.Duration) (string, error) {
	for {
		select {
		case <-ctx.Done():
			return "", errors.New("PIN linking canceled")
		default:
			pin, err := CheckPin(pinID, pinCode, clientIdentifier)
			if err != nil {
				return "", err
			}
			if pin.AuthToken != "" {
				return pin.AuthToken, nil
			}
			time.Sleep(interval)
		}
	}
}

// AuthAppURL returns the URL where the user should be directed to authorize the PIN.
func AuthAppURL(clientIdentifier, code, product string) string {
	params := url.Values{
		"clientID":                  {clientIdentifier},
		"code":                      {code},
		"context[device][product]": {product},
	}
	return authAppBaseURL + "?" + params.Encode()
}

// Product is the application name used in PIN authentication.
const Product = "Scanline"

// StartPinLinking initiates the PIN-based authentication flow.
//
// The callback function is called with the PIN, authentication URL, and a cancel function.
// The caller should display the auth URL to the user and call cancel if they want to abort.
// This function blocks until the user completes authentication or the context is cancelled.
func StartPinLinking(clientIdentifier string, cb func(*Pin, string, context.CancelFunc)) (string, error) {
	pin, err := RequestPin(clientIdentifier)
	if err != nil {
		return "", err
	}

	authURL := AuthAppURL(clientIdentifier, pin.Code, Product)

	ctx, cancel := context.WithCancel(context.Background())
	cb(pin, authURL, cancel)

	token, err := PollPin(ctx, pin.ID, pin.Code, clientIdentifier, 2*time.Second)
	if err != nil {
		return "", err
	}
	return token, nil
}

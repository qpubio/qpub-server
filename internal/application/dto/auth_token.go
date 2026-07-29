package dto

import "encoding/json"

// IssueTokenRequest represents the request body for issuing a token
type IssueTokenRequest struct {
	Permission json.RawMessage `json:"permission,omitempty"`
	Alias      string          `json:"alias,omitempty"`
	ExpiresIn  *int            `json:"expiresIn,omitempty"` // in seconds
}

// TokenResponse represents the response body for token issuance
type TokenResponse struct {
	Token string `json:"token"`
}

// TokenRequestBody represents the request body for obtaining a token via a pre-signed request object.
// Created by a trusted server with the API secret key, then sent by an untrusted client.
type TokenRequestBody struct {
	AKI        string          `json:"aki" binding:"required"`       // API Key public ID
	Permission json.RawMessage `json:"permission,omitempty"`         // Permission map
	Alias      string          `json:"alias,omitempty"`              // Client identifier
	Timestamp  int64           `json:"timestamp" binding:"required"` // Request timestamp (unix seconds)
	Signature  string          `json:"signature" binding:"required"` // HMAC-SHA256 signature of signable fields
}

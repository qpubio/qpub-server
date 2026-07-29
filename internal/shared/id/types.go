package id

// Int represents the internal auto-incrementing ID
type Int = int

// Hash represents the public-facing ID (length configured via HASHID_LENGTH)
type Hash = string

// ULID represents distributed resource ID (22 chars)
type ULID = string

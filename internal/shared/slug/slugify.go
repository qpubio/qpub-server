package slug

import (
	"regexp"
	"strings"
)

var (
	// specialCharsRegex matches any characters that aren't lowercase letters, numbers, or hyphens
	specialCharsRegex = regexp.MustCompile(`[^a-z0-9-]`)

	// multiHyphenRegex matches multiple consecutive hyphens
	multiHyphenRegex = regexp.MustCompile(`-+`)
)

// SlugifyOptions holds configuration for slug generation
type SlugifyOptions struct {
	MaxLength    int  // Maximum length of the resulting slug
	AllowUnicode bool // Whether to allow Unicode characters
	ForceLower   bool // Whether to force lowercase (default: true)
	TrimSpace    bool // Whether to trim spaces (default: true)
}

// DefaultSlugifyOptions returns the default options for slug generation
func DefaultSlugifyOptions() SlugifyOptions {
	return SlugifyOptions{
		MaxLength:    100,
		ForceLower:   true,
		TrimSpace:    true,
		AllowUnicode: false,
	}
}

// Slugify converts a string to a URL-friendly slug with default options
func Slugify(text string) string {
	return SlugifyWithOptions(text, DefaultSlugifyOptions())
}

// SlugifyWithOptions converts a string to a URL-friendly slug with custom options
func SlugifyWithOptions(text string, opts SlugifyOptions) string {
	if text == "" {
		return ""
	}

	// Trim spaces if enabled
	if opts.TrimSpace {
		text = strings.TrimSpace(text)
	}

	// Convert to lowercase if enabled
	if opts.ForceLower {
		text = strings.ToLower(text)
	}

	// Handle Unicode characters
	if !opts.AllowUnicode {
		// Replace special characters with hyphens
		text = specialCharsRegex.ReplaceAllString(text, "-")
	} else {
		// Replace only common separators with hyphens
		text = strings.ReplaceAll(text, " ", "-")
		text = strings.ReplaceAll(text, "_", "-")
		text = strings.ReplaceAll(text, ".", "-")
	}

	// Replace multiple consecutive hyphens with a single hyphen
	text = multiHyphenRegex.ReplaceAllString(text, "-")

	// Remove leading/trailing hyphens
	text = strings.Trim(text, "-")

	// Enforce maximum length if specified
	if opts.MaxLength > 0 && len(text) > opts.MaxLength {
		// Cut at max length and ensure we don't end with a hyphen
		text = strings.TrimRight(text[:opts.MaxLength], "-")
	}

	return text
}

// MustSlugify is like Slugify but panics if the input is invalid
func MustSlugify(text string) string {
	if text == "" {
		panic("cannot create slug from empty string")
	}
	return Slugify(text)
}

// IsValidSlug checks if a string is a valid slug
func IsValidSlug(slug string) bool {
	if slug == "" {
		return false
	}

	// Check if the slug contains only lowercase letters, numbers, and hyphens
	return !specialCharsRegex.MatchString(slug) &&
		!strings.HasPrefix(slug, "-") &&
		!strings.HasSuffix(slug, "-")
}

package ui

import "embed"

// QuotesJSON is an embedded filesystem containing the quotes.json file.
// This allows the quotes to be bundled directly into the application binary.
//
//go:embed quotes.json
var QuotesJSON embed.FS

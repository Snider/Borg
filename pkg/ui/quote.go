package ui

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/fatih/color"
)

var (
	cachedQuotes *Quotes
	quotesOnce   sync.Once
	quotesErr    error
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// Quotes represents the structure of the quotes.json file.
type Quotes struct {
	InitWorkAssimilate        []string `json:"init_work_assimilate"`
	EncryptionServiceMessages []string `json:"encryption_service_messages"`
	CodeRelatedShort          []string `json:"code_related_short"`
	VCSProcessing             []string `json:"vcs_processing"`
	PWAProcessing             []string `json:"pwa_processing"`
	CodeRelatedLong           []string `json:"code_related_long"`
	ImageRelated              struct {
		PNG  string `json:"png"`
		JPG  string `json:"jpg"`
		SVG  string `json:"svg"`
		WEBP string `json:"webp"`
		HEIC string `json:"heic"`
		RAW  string `json:"raw"`
		ICO  string `json:"ico"`
		AVIF string `json:"avif"`
		TIFF string `json:"tiff"`
		GIF  string `json:"gif"`
	} `json:"image_related"`
}

func loadQuotes() (*Quotes, error) {
	quotesFile, err := QuotesJSON.ReadFile("quotes.json")
	if err != nil {
		return nil, fmt.Errorf("failed to read quotes.json: %w", err)
	}

	var quotes Quotes
	if err := json.Unmarshal(quotesFile, &quotes); err != nil {
		return nil, fmt.Errorf("failed to unmarshal quotes.json: %w", err)
	}
	return &quotes, nil
}

func getQuotes() (*Quotes, error) {
	quotesOnce.Do(func() {
		cachedQuotes, quotesErr = loadQuotes()
	})
	return cachedQuotes, quotesErr
}

// GetRandomQuote selects and returns a random quote from all available quote
// categories.
func GetRandomQuote() (string, error) {
	quotes, err := getQuotes()
	if err != nil {
		return "", err
	}

	allQuotes := []string{}
	allQuotes = append(allQuotes, quotes.InitWorkAssimilate...)
	allQuotes = append(allQuotes, quotes.EncryptionServiceMessages...)
	allQuotes = append(allQuotes, quotes.CodeRelatedShort...)
	allQuotes = append(allQuotes, quotes.VCSProcessing...)
	allQuotes = append(allQuotes, quotes.PWAProcessing...)
	allQuotes = append(allQuotes, quotes.CodeRelatedLong...)

	if len(allQuotes) == 0 {
		return "", fmt.Errorf("no quotes available")
	}

	return allQuotes[rand.Intn(len(allQuotes))], nil
}

// PrintQuote retrieves a random quote and prints it to the console in green.
func PrintQuote() {
	quote, err := GetRandomQuote()
	if err != nil {
		fmt.Println("Error getting quote:", err)
		return
	}
	c := color.New(color.FgGreen)
	c.Println(quote)
}

// GetVCSQuote returns a random quote specifically from the VCS processing category.
func GetVCSQuote() (string, error) {
	quotes, err := getQuotes()
	if err != nil {
		return "", err
	}
	if len(quotes.VCSProcessing) == 0 {
		return "", fmt.Errorf("no VCS quotes available")
	}
	return quotes.VCSProcessing[rand.Intn(len(quotes.VCSProcessing))], nil
}

// GetPWAQuote returns a random quote specifically from the PWA processing category.
func GetPWAQuote() (string, error) {
	quotes, err := getQuotes()
	if err != nil {
		return "", err
	}
	if len(quotes.PWAProcessing) == 0 {
		return "", fmt.Errorf("no PWA quotes available")
	}
	return quotes.PWAProcessing[rand.Intn(len(quotes.PWAProcessing))], nil
}

// GetWebsiteQuote returns a random quote specifically from the "code related long"
// category, which is used for website processing.
func GetWebsiteQuote() (string, error) {
	quotes, err := getQuotes()
	if err != nil {
		return "", err
	}
	if len(quotes.CodeRelatedLong) == 0 {
		return "", fmt.Errorf("no website quotes available")
	}
	return quotes.CodeRelatedLong[rand.Intn(len(quotes.CodeRelatedLong))], nil
}

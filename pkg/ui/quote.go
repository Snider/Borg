
package ui

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"github.com/Snider/Borg/data"
	"github.com/fatih/color"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

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
	quotesFile, err := data.QuotesJSON.ReadFile("quotes.json")
	if err != nil {
		return nil, fmt.Errorf("failed to read quotes.json: %w", err)
	}

	var quotes Quotes
	if err := json.Unmarshal(quotesFile, &quotes); err != nil {
		return nil, fmt.Errorf("failed to unmarshal quotes.json: %w", err)
	}
	return &quotes, nil
}

func GetRandomQuote() (string, error) {
	quotes, err := loadQuotes()
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

	return allQuotes[rand.Intn(len(allQuotes))], nil
}

func PrintQuote() {
	quote, err := GetRandomQuote()
	if err != nil {
		fmt.Println("Error getting quote:", err)
		return
	}
	c := color.New(color.FgGreen)
	c.Println(quote)
}

func GetVCSQuote() (string, error) {
	quotes, err := loadQuotes()
	if err != nil {
		return "", err
	}
	return quotes.VCSProcessing[rand.Intn(len(quotes.VCSProcessing))], nil
}

func GetPWAQuote() (string, error) {
	quotes, err := loadQuotes()
	if err != nil {
		return "", err
	}
	return quotes.PWAProcessing[rand.Intn(len(quotes.PWAProcessing))], nil
}

func GetWebsiteQuote() (string, error) {
	quotes, err := loadQuotes()
	if err != nil {
		return "", err
	}
	return quotes.CodeRelatedLong[rand.Intn(len(quotes.CodeRelatedLong))], nil
}

package changelog

import (
	"encoding/json"
	"fmt"
	"strings"
)

// FormatAsText formats the ChangeReport as plain text.
func FormatAsText(report *ChangeReport) (string, error) {
	var builder strings.Builder

	if len(report.Added) > 0 {
		builder.WriteString(fmt.Sprintf("Added (%d files):\n", len(report.Added)))
		for _, file := range report.Added {
			builder.WriteString(fmt.Sprintf("- %s\n", file))
		}
		builder.WriteString("\n")
	}

	if len(report.Modified) > 0 {
		builder.WriteString(fmt.Sprintf("Modified (%d files):\n", len(report.Modified)))
		for _, file := range report.Modified {
			builder.WriteString(fmt.Sprintf("- %s (+%d -%d lines)\n", file.Path, file.Additions, file.Deletions))
			for _, commit := range file.Commits {
				builder.WriteString(fmt.Sprintf("  - %s\n", strings.Split(commit, "\n")[0]))
			}
		}
		builder.WriteString("\n")
	}

	if len(report.Removed) > 0 {
		builder.WriteString(fmt.Sprintf("Removed (%d files):\n", len(report.Removed)))
		for _, file := range report.Removed {
			builder.WriteString(fmt.Sprintf("- %s\n", file))
		}
		builder.WriteString("\n")
	}

	return builder.String(), nil
}

// FormatAsMarkdown formats the ChangeReport as Markdown.
func FormatAsMarkdown(report *ChangeReport) (string, error) {
	var builder strings.Builder

	builder.WriteString("# Changes\n\n")

	if len(report.Added) > 0 {
		builder.WriteString(fmt.Sprintf("## Added (%d files)\n", len(report.Added)))
		for _, file := range report.Added {
			builder.WriteString(fmt.Sprintf("- `%s`\n", file))
		}
		builder.WriteString("\n")
	}

	if len(report.Modified) > 0 {
		builder.WriteString(fmt.Sprintf("## Modified (%d files)\n", len(report.Modified)))
		for _, file := range report.Modified {
			builder.WriteString(fmt.Sprintf("- `%s` (+%d -%d lines)\n", file.Path, file.Additions, file.Deletions))
			for _, commit := range file.Commits {
				builder.WriteString(fmt.Sprintf("  - *%s*\n", strings.Split(commit, "\n")[0]))
			}
		}
		builder.WriteString("\n")
	}

	if len(report.Removed) > 0 {
		builder.WriteString(fmt.Sprintf("## Removed (%d files)\n", len(report.Removed)))
		for _, file := range report.Removed {
			builder.WriteString(fmt.Sprintf("- `%s`\n", file))
		}
		builder.WriteString("\n")
	}

	return builder.String(), nil
}

// FormatAsJSON formats the ChangeReport as JSON.
func FormatAsJSON(report *ChangeReport) (string, error) {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

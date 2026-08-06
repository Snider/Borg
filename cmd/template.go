package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Snider/Borg/pkg/templates"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func init() {
	RootCmd.AddCommand(NewTemplateCmd())
}

func NewTemplateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "template",
		Short: "Manage and run collection templates",
		Long:  `Manage and run collection templates.`,
	}

	cmd.AddCommand(NewTemplateListCmd())
	cmd.AddCommand(NewTemplateRunCmd())
	cmd.AddCommand(NewTemplateSaveCmd())

	return cmd
}

func NewTemplateListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available templates",
		Long:  `List available templates.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			userTemplateDir, err := templates.GetUserTemplateDir()
			if err != nil {
				return err
			}

			userTemplates, err := templates.ListUserTemplates(userTemplateDir)
			if err != nil {
				return err
			}

			builtinTemplates, err := templates.ListBuiltinTemplates()
			if err != nil {
				return err
			}

			if len(userTemplates) == 0 && len(builtinTemplates) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No templates found.")
				return nil
			}

			if len(userTemplates) > 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "Custom templates:")
				for _, t := range userTemplates {
					fmt.Fprintf(cmd.OutOrStdout(), "  - %s\n", t)
				}
			}

			if len(builtinTemplates) > 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "Built-in templates:")
				for _, t := range builtinTemplates {
					fmt.Fprintf(cmd.OutOrStdout(), "  - %s\n", t)
				}
			}

			return nil
		},
	}
}

func NewTemplateRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run [template-name] [flags]",
		Short: "Run a collection template",
		Long:  `Run a collection template.`,
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			templateName := args[0]

			// Find and load the template
			_, templateData, err := templates.FindTemplate(templateName)
			if err != nil {
				return err
			}
			tmpl, err := templates.LoadTemplate(templateData)
			if err != nil {
				return err
			}

			// Parse variable flags
			vars := make(map[string]string)
			for i := 1; i < len(args); i++ {
				arg := args[i]
				if strings.HasPrefix(arg, "--") {
					key := strings.TrimPrefix(arg, "--")
					if i+1 < len(args) && !strings.HasPrefix(args[i+1], "--") {
						vars[key] = args[i+1]
						i++
					} else {
						// Handle boolean flags if necessary
						vars[key] = "true"
					}
				}
			}

			// Validate required variables
			for key, value := range tmpl.Variables {
				if value == "required" {
					if _, ok := vars[key]; !ok {
						return fmt.Errorf("missing required variable: %s", key)
					}
				}
			}

			// Execute steps
			for _, step := range tmpl.Steps {
				collectCmdStr := templates.Substitute(step.Collect, vars)
				output := templates.Substitute(step.Output, vars)

				cmdArgs := []string{"collect"}
				cmdArgs = append(cmdArgs, strings.Fields(collectCmdStr)...)

				if output != "" {
					cmdArgs = append(cmdArgs, "--output", output)
				}
				if step.Encrypt {
					// This assumes a --password flag exists on the target command
					// A more robust implementation might be needed
					if password, ok := vars["password"]; ok {
						cmdArgs = append(cmdArgs, "--password", password)
					} else {
						// It might be better to prompt for a password
						return fmt.Errorf("encryption requested but no password provided")
					}
				}
				if step.Depth > 0 {
					cmdArgs = append(cmdArgs, "--depth", fmt.Sprintf("%d", step.Depth))
				}

				rootCmd := cmd.Root()
				subCmd, remainingArgs, err := rootCmd.Find(cmdArgs)
				if err != nil {
					return fmt.Errorf("could not find command for step '%s': %w", collectCmdStr, err)
				}

				subCmd.SetArgs(remainingArgs)
				var runErr error
				if subCmd.RunE != nil {
					runErr = subCmd.RunE(subCmd, remainingArgs)
				} else if subCmd.Run != nil {
					subCmd.Run(subCmd, remainingArgs)
				}
				if runErr != nil {
					return fmt.Errorf("error executing step '%s': %w", collectCmdStr, runErr)
				}
			}

			return nil
		},
	}
	// This allows the command to accept arbitrary flags for variables
	cmd.Flags().SetInterspersed(false)
	return cmd
}

func NewTemplateSaveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "save [template-name]",
		Short: "Save a new template from history",
		Long:  `Save a new template from history.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			templateName := args[0]
			from, _ := cmd.Flags().GetString("from")

			var commandToSave string
			var err error

			if from != "" {
				commandToSave = from
			} else {
				commandToSave, err = templates.ReadLastHistoryEntry()
				if err != nil {
					return err
				}
			}

			// Basic parsing
			parts := strings.Fields(commandToSave)
			if len(parts) < 2 || parts[0] != "collect" {
				return fmt.Errorf("can only save 'collect' commands, but got: %s", commandToSave)
			}

			step := templates.Step{}
			vars := make(map[string]string)
			var collectArgs []string

			// This is still a simplified parser, but better.
			i := 1 // Skip "collect"
			for i < len(parts) {
				part := parts[i]
				if strings.HasPrefix(part, "--") {
					flagName := strings.TrimPrefix(part, "--")
					if i+1 >= len(parts) {
						return fmt.Errorf("flag '%s' has no value", part)
					}
					value := parts[i+1]
					i += 2

					switch flagName {
					case "output":
						step.Output = value
					case "password":
						step.Encrypt = true
						vars["password"] = "required"
					case "depth":
						step.Depth, _ = strconv.Atoi(value)
					default:
						// Assume other flags are variables for the collect command
						varName := flagName
						collectArgs = append(collectArgs, fmt.Sprintf("--%s", flagName), fmt.Sprintf("{{%s}}", varName))
						vars[varName] = "required"
					}
				} else {
					collectArgs = append(collectArgs, part)
					i++
				}
			}
			step.Collect = strings.Join(collectArgs, " ")

			tmpl := templates.Template{
				Name:      templateName,
				Steps:     []templates.Step{step},
				Variables: vars,
			}

			data, err := yaml.Marshal(&tmpl)
			if err != nil {
				return fmt.Errorf("could not marshal template: %w", err)
			}

			userTemplateDir, err := templates.GetUserTemplateDir()
			if err != nil {
				return err
			}
			if err := os.MkdirAll(userTemplateDir, 0755); err != nil {
				return fmt.Errorf("could not create template directory: %w", err)
			}

			templatePath := filepath.Join(userTemplateDir, templateName+".yaml")
			if err := os.WriteFile(templatePath, data, 0644); err != nil {
				return fmt.Errorf("could not write template file: %w", err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Template saved to %s\n", templatePath)
			return nil
		},
	}
	cmd.Flags().String("from", "", "Specify a command from history to save")
	return cmd
}

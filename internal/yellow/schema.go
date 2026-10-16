package yellow

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// terminalNote marks commands that need a person at a terminal. Everything
// else runs unattended.
const terminalNote = "yellow/terminal"

type schemaFlag struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	Default    string `json:"default,omitempty"`
	Usage      string `json:"usage"`
	Required   bool   `json:"required,omitempty"`
	Repeatable bool   `json:"repeatable,omitempty"`
}
type schemaCommand struct {
	Command  string       `json:"command"`
	Usage    string       `json:"usage"`
	Summary  string       `json:"summary"`
	Terminal string       `json:"terminal,omitempty"`
	Flags    []schemaFlag `json:"flags,omitempty"`
}

func flagsOf(set *pflag.FlagSet) []schemaFlag {
	flags := []schemaFlag{}
	set.VisitAll(func(f *pflag.Flag) {
		if f.Hidden || f.Name == "help" {
			return
		}
		required := len(f.Annotations[cobra.BashCompOneRequiredFlag]) > 0
		def := f.DefValue
		if def == "[]" || def == "false" || def == "0s" || (def == "0" && f.Value.Type() == "int") {
			def = ""
		}
		flags = append(flags, schemaFlag{Name: "--" + f.Name, Type: strings.TrimSuffix(f.Value.Type(), "Array"), Default: def, Usage: f.Usage, Required: required, Repeatable: strings.HasSuffix(f.Value.Type(), "Array")})
	})
	sort.Slice(flags, func(i, j int) bool { return flags[i].Name < flags[j].Name })
	return flags
}

func (a *app) schemaCommand() *cobra.Command {
	return &cobra.Command{Use: "schema", Short: "Describe every command, option and exit code as JSON for agents", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		root := cmd.Root()
		commands := []schemaCommand{}
		var walk func(*cobra.Command)
		walk = func(c *cobra.Command) {
			for _, child := range c.Commands() {
				if child.Hidden || child.Name() == "help" || child.Name() == "completion" {
					continue
				}
				if child.HasSubCommands() {
					walk(child)
					continue
				}
				commands = append(commands, schemaCommand{Command: child.CommandPath(), Usage: child.UseLine(), Summary: child.Short, Terminal: child.Annotations[terminalNote], Flags: flagsOf(child.LocalNonPersistentFlags())})
			}
		}
		walk(root)
		schema := map[string]any{
			"name":        "yellow",
			"version":     a.version,
			"globalFlags": flagsOf(root.PersistentFlags()),
			"commands":    commands,
			"output":      "With --json, stdout is exactly one object: {\"version\":1,\"result\":…} or {\"version\":1,\"error\":\"…\",\"exitCode\":n}. Prompts and diagnostics go to stderr.",
			"exitCodes": map[string]string{
				"0": "success, including a sealed transfer", "2": "input or local storage error", "3": "transfer pending",
				"4": "outcome unknown or unresolved; check it before sending again", "5": "transfer rejected",
				"6": "network, trust, capability or index failure", "130": "cancelled",
			},
			"agentAccounts": "account create <name> --agent creates an account that signs without prompts. Its balance is the limit of what it can spend.",
		}
		return a.emit(schema, func() {
			raw, _ := json.MarshalIndent(schema, "", "  ")
			fmt.Fprintln(a.out, string(raw))
		})
	}}
}

// annotateTerminal records which commands need a person at a terminal.
func annotateTerminal(root *cobra.Command) {
	notes := map[string]string{
		"yellow account create":       "personal accounts only; --agent runs unattended",
		"yellow account import":       "always",
		"yellow account backup":       "always",
		"yellow account export-key":   "always",
		"yellow network add":          "unless --fingerprint is given",
		"yellow network trust-update": "unless --fingerprint is given",
		"yellow transfer send":        "personal accounts only; agent accounts and --dry-run run unattended",
		"yellow operation retry":      "personal accounts only; agent accounts run unattended",
		"yellow shell":                "always",
	}
	var walk func(*cobra.Command)
	walk = func(c *cobra.Command) {
		if note, ok := notes[c.CommandPath()]; ok {
			if c.Annotations == nil {
				c.Annotations = map[string]string{}
			}
			c.Annotations[terminalNote] = note
		}
		for _, child := range c.Commands() {
			walk(child)
		}
	}
	walk(root)
}

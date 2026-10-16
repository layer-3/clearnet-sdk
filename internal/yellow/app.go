// Package yellow implements the local user CLI using only the public gateway client.
package yellow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"
	"time"
	"unicode"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

const (
	ExitInput     = 2
	ExitPending   = 3
	ExitUnknown   = 4
	ExitRejected  = 5
	ExitNetwork   = 6
	ExitCancelled = 130
)

type failure struct {
	code     int
	message  string
	reported bool
}

func (e *failure) Error() string { return e.message }
func fail(code int, format string, args ...any) error {
	return &failure{code: code, message: fmt.Sprintf(format, args...)}
}

type prompter interface {
	secret(string) ([]byte, error)
	line(string) (string, error)
	confirm(string) error
	recovery(string) error
	privateKey(string, string, []byte) error
}
type terminalPrompt struct {
	in  *os.File
	out io.Writer
	ctx context.Context
}

func (p terminalPrompt) tty() error {
	if p.in == nil || !term.IsTerminal(int(p.in.Fd())) {
		return fail(ExitInput, "this action needs an interactive terminal; secrets and approvals cannot be supplied through arguments, environment variables or pipes")
	}
	return nil
}
func (p terminalPrompt) secret(label string) ([]byte, error) {
	if err := p.tty(); err != nil {
		return nil, err
	}
	state, err := term.MakeRaw(int(p.in.Fd()))
	if err != nil {
		return nil, err
	}
	defer term.Restore(int(p.in.Fd()), state)
	type result struct {
		raw []byte
		err error
	}
	done := make(chan result)
	go func() {
		terminal := term.NewTerminal(struct {
			io.Reader
			io.Writer
		}{p.in, p.out}, "")
		value, err := terminal.ReadPassword(label + ": ")
		raw := []byte(value)
		select {
		case done <- result{raw, err}:
		case <-p.ctx.Done():
			clear(raw)
		}
	}()
	select {
	case <-p.ctx.Done():
		return nil, fail(ExitCancelled, "input cancelled")
	case r := <-done:
		if r.err != nil {
			clear(r.raw)
			return nil, fail(ExitCancelled, "input cancelled")
		}
		return r.raw, nil
	}
}
func (p terminalPrompt) line(label string) (string, error) {
	if err := p.tty(); err != nil {
		return "", err
	}
	fmt.Fprint(p.out, label+": ")
	type result struct {
		value string
		err   error
	}
	done := make(chan result)
	go func() {
		value, err := readTerminalLine(p.in)
		select {
		case done <- result{value, err}:
		case <-p.ctx.Done():
		}
	}()
	select {
	case <-p.ctx.Done():
		return "", fail(ExitCancelled, "input cancelled")
	case r := <-done:
		return r.value, r.err
	}
}
func readTerminalLine(in *os.File) (string, error) {
	// Avoid prefetching a following hidden secret.
	var b []byte
	defer func() { clear(b) }()
	one := make([]byte, 1)
	for len(b) < 1<<20 {
		n, err := in.Read(one)
		if err != nil {
			return "", fail(ExitCancelled, "input cancelled")
		}
		if n > 0 {
			if one[0] == '\n' {
				return strings.TrimSpace(string(b)), nil
			}
			b = append(b, one[0])
		}
	}
	return "", fail(ExitInput, "input is too long")
}
func (p terminalPrompt) confirm(label string) error {
	answer, err := p.line(label + " [yes/no]")
	if err != nil {
		return err
	}
	if !strings.EqualFold(answer, "yes") {
		return fail(ExitCancelled, "cancelled; no authorization submitted")
	}
	return nil
}
func (p terminalPrompt) recovery(words string) error {
	if err := p.tty(); err != nil {
		return err
	}
	// Secrets go directly to the terminal, never to redirected diagnostic output.
	tty, err := os.OpenFile(terminalDevice(), os.O_WRONLY, 0)
	if err != nil {
		return fmt.Errorf("cannot display recovery words on the terminal")
	}
	defer tty.Close()
	list := strings.Fields(words)
	fmt.Fprint(tty, "\nRecovery words · shown once\nWrite these down in order and keep them somewhere private.\n\n")
	for i := 0; i < len(list); i += 4 {
		for j := i; j < i+4 && j < len(list); j++ {
			fmt.Fprintf(tty, "%2d. %-12s ", j+1, list[j])
		}
		fmt.Fprintln(tty)
	}
	fmt.Fprintln(tty, "\nAnyone with these words can control this account.")
	return p.confirm("I have saved all 24 recovery words safely")
}

func (p terminalPrompt) privateKey(name, address string, key []byte) error {
	if err := p.tty(); err != nil {
		return err
	}
	// Keep the revealed key out of redirected stdout and stderr.
	tty, err := os.OpenFile(terminalDevice(), os.O_WRONLY, 0)
	if err != nil {
		return fmt.Errorf("cannot display the private key on the terminal")
	}
	defer tty.Close()
	if _, err := fmt.Fprintf(tty, "\nPrivate key export · %s\nAddress: %s\n\nAnyone with this key can control this account.\nRevealing it leaves it in your terminal scrollback.\n", name, address); err != nil {
		return fmt.Errorf("cannot display the private key export review on the terminal")
	}
	answer, err := p.line("Reveal this private key [yes/no]")
	if err != nil {
		return err
	}
	if !strings.EqualFold(answer, "yes") {
		return fail(ExitCancelled, "cancelled; private key was not revealed")
	}
	if err := p.ctx.Err(); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(tty, "\nPrivate key · copy only into your wallet's private-key import field"); err != nil {
		return fmt.Errorf("cannot display the private key on the terminal")
	}
	if _, err := tty.Write(key); err != nil {
		return fmt.Errorf("cannot display the private key on the terminal")
	}
	_, err = fmt.Fprintln(tty, "\n\nIn MetaMask, import this as an Ethereum account and verify the address above.\nKeep your encrypted Yellow backup; it also contains your operation journal.")
	return err
}

type app struct {
	store            store
	cfg              config
	in               *os.File
	out, diag        io.Writer
	prompt           prompter
	network, account string
	json, noColor    bool
	timeout          time.Duration
	version          string
	completions      *completionData
	inShell          bool
}

// Execute runs one command. JSON output is one versioned, nonsecret object.
func Execute(ctx context.Context, args []string, in *os.File, out, diag io.Writer, dir, version string) int {
	if dir == "" {
		base, err := os.UserConfigDir()
		if err != nil {
			fmt.Fprintln(diag, "Cannot locate local configuration:", err)
			return ExitInput
		}
		dir = filepath.Join(base, "yellow")
	}
	a := &app{store: store{dir}, in: in, out: out, diag: diag, version: version}
	a.prompt = terminalPrompt{in: in, out: diag, ctx: ctx}
	return a.execute(ctx, args)
}
func (a *app) execute(ctx context.Context, args []string) int {
	return a.executeCommand(ctx, args, a.command())
}
func (a *app) executeCommand(ctx context.Context, args []string, root *cobra.Command) int {
	// Find errors can precede flag parsing; honor the JSON request for diagnostics.
	for _, arg := range args {
		if arg == "--" {
			break
		}
		switch arg {
		case "--json", "--json=true":
			a.json = true
		case "--json=false":
			a.json = false
		}
	}
	root.SetArgs(args)
	root.SetOut(a.out)
	root.SetErr(a.diag)
	err := root.ExecuteContext(ctx)
	if err == nil {
		return 0
	}
	code := ExitInput
	var f *failure
	if errors.As(err, &f) {
		code = f.code
	}
	if errors.Is(err, context.Canceled) || ctx.Err() != nil {
		code = ExitCancelled
	}
	if f != nil && f.reported {
		return code
	}
	if a.json {
		_ = json.NewEncoder(a.out).Encode(map[string]any{"version": 1, "error": clean(err.Error()), "exitCode": code})
	} else {
		fmt.Fprintf(a.diag, "\n%s %s\n", a.colorFor(a.diag, "31", "Error"), cleanDiagnostic(err.Error()))
	}
	return code
}
func (a *app) color(code, text string) string { return a.colorFor(a.out, code, text) }
func (a *app) colorFor(w io.Writer, code, text string) string {
	f, ok := w.(*os.File)
	if a.noColor || os.Getenv("NO_COLOR") != "" || !ok || !term.IsTerminal(int(f.Fd())) {
		return text
	}
	return "\x1b[" + code + "m" + text + "\x1b[0m"
}
func clean(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, s)
}
func (a *app) title(text string) { fmt.Fprintln(a.out, "\n"+a.color("1;33", clean(text))) }
func (a *app) table(headers []string, rows [][]string) {
	w := tabwriter.NewWriter(a.out, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, strings.Join(headers, "\t"))
	for _, row := range rows {
		for i := range row {
			row[i] = clean(row[i])
		}
		fmt.Fprintln(w, strings.Join(row, "\t"))
	}
	_ = w.Flush()
}
func (a *app) emit(data any, human func()) error {
	if a.json {
		enc := json.NewEncoder(a.out)
		enc.SetEscapeHTML(false)
		return enc.Encode(struct {
			Version int `json:"version"`
			Result  any `json:"result"`
		}{1, data})
	}
	human()
	return nil
}
func (a *app) action(fn func(context.Context, *cobra.Command, []string) error) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		if a.timeout <= 0 {
			return fail(ExitInput, "--timeout must be positive")
		}
		// The lock wait is a local budget, independent of the gateway request
		// timeout: a short --timeout must not turn a busy journal into an
		// unrelated storage error.
		lockctx, cancel := context.WithTimeout(cmd.Context(), lockWait)
		defer cancel()
		unlock, err := a.store.lock(lockctx)
		if err != nil {
			return err
		}
		defer unlock()
		a.cfg, err = a.store.config()
		if err != nil {
			return err
		}
		if err := cmd.Context().Err(); err != nil {
			return err
		}
		return fn(cmd.Context(), cmd, args)
	}
}
func (a *app) cmd(use, short string, args cobra.PositionalArgs, fn func(context.Context, *cobra.Command, []string) error) *cobra.Command {
	c := &cobra.Command{Use: use, Short: short, Args: args, RunE: a.action(fn)}
	c.ValidArgsFunction = func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return c
}
func (a *app) command() *cobra.Command {
	root := &cobra.Command{Use: "yellow", Short: "Your Clearnet account, at your fingertips.", Long: "YELLOW · Clearnet in your terminal\n\nManage local accounts, view live balances, and send transfers with safe recovery.\n\nStart here:\n  yellow network add testnet --gateway https://gateway.example.org --network-id YTES\n  yellow account create personal\n  yellow balance\n\nPress Tab to discover commands, saved accounts and networks.\nEnable it with: yellow completion --help\nOr use the interactive prompt: yellow shell", SilenceUsage: true, SilenceErrors: true, SuggestionsMinimumDistance: 2}
	root.PersistentFlags().StringVar(&a.network, "network", "", "Use a saved network for this command")
	root.PersistentFlags().StringVar(&a.account, "account", "", "Use a saved account for this command")
	root.PersistentFlags().BoolVar(&a.json, "json", false, "Return one versioned JSON object")
	root.PersistentFlags().DurationVar(&a.timeout, "timeout", 15*time.Second, "Time limit for each gateway request")
	root.PersistentFlags().BoolVar(&a.noColor, "no-color", false, "Disable terminal colors")
	root.Args = cobra.ArbitraryArgs
	root.RunE = groupRun
	root.AddGroup(&cobra.Group{ID: "setup", Title: "Set up:"}, &cobra.Group{ID: "daily", Title: "Use Clearnet:"}, &cobra.Group{ID: "tools", Title: "Tools:"})
	network := a.networkCommands()
	network.GroupID = "setup"
	account := a.accountCommands()
	account.GroupID = "setup"
	balance := a.balanceCommand()
	balance.GroupID = "daily"
	transfer := a.transferCommands()
	transfer.GroupID = "daily"
	operation := a.operationCommands()
	operation.GroupID = "daily"
	version := &cobra.Command{Use: "version", Short: "Show the installed version", GroupID: "tools", Args: cobra.NoArgs, RunE: func(*cobra.Command, []string) error {
		return a.emit(map[string]string{"version": a.version}, func() { fmt.Fprintln(a.out, "yellow", a.version) })
	}}
	root.SetHelpCommandGroupID("tools")
	root.SetCompletionCommandGroupID("tools")
	shell := a.shellCommand()
	shell.GroupID = "tools"
	schema := a.schemaCommand()
	schema.GroupID = "tools"
	root.AddCommand(network, account, balance, transfer, operation, version, schema, shell)
	annotateTerminal(root)
	_ = root.RegisterFlagCompletionFunc("network", a.completeNetworks)
	_ = root.RegisterFlagCompletionFunc("account", a.completeAccounts)
	return root
}
func (a *app) completeNetworks(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	cfg, err := a.completionConfig()
	var names []string
	if err == nil {
		for n, p := range cfg.Networks {
			names = append(names, n+"\t"+p.NetworkID)
		}
	}
	sort.Strings(names)
	return names, cobra.ShellCompDirectiveNoFileComp
}
func (a *app) completeAccounts(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	accounts, _ := a.completionAccounts()
	var names []string
	for _, ac := range accounts {
		names = append(names, ac.name+"\t"+ac.address)
	}
	return names, cobra.ShellCompDirectiveNoFileComp
}
func (a *app) selectedAccount(explicit string) (string, *accountFile, error) {
	name := explicit
	if name == "" {
		name = a.account
	}
	if name == "" {
		name = a.cfg.Account
	}
	if name == "" {
		return "", nil, fail(ExitInput, "choose an account with --account, or run yellow account create <name>")
	}
	ac, err := a.store.account(name)
	return name, ac, err
}

func (a *app) completeAddresses(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	accounts, _ := a.completionAccounts()
	var addresses []string
	for _, ac := range accounts {
		addresses = append(addresses, ac.address+"\t"+ac.name)
	}
	return addresses, cobra.ShellCompDirectiveNoFileComp
}

func cleanDiagnostic(s string) string {
	return strings.Map(func(r rune) rune {
		if r != '\n' && r != '\t' && unicode.IsControl(r) {
			return ' '
		}
		return r
	}, s)
}

func groupRun(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return cmd.Help()
	}
	message := fmt.Sprintf("unknown command %q for %q", clean(args[0]), cmd.CommandPath())
	if suggestions := cmd.SuggestionsFor(args[0]); len(suggestions) > 0 {
		message += "\n\nDid you mean?\n  " + strings.Join(suggestions, "\n  ")
	}
	return fail(ExitInput, "%s", message)
}
func group(use, short string) *cobra.Command {
	return &cobra.Command{Use: use, Short: short, Args: cobra.ArbitraryArgs, RunE: groupRun, SuggestionsMinimumDistance: 2}
}

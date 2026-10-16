package yellow

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"golang.org/x/term"
)

type suggestion struct {
	value, description string
	noSpace            bool
}
type shellModel struct {
	runner, completer     *app
	runTree, completeTree *cobra.Command
	app                   *app
	ctx                   context.Context
	input                 textinput.Model
	data                  *completionData
	suggestions           []suggestion
	start, end            int
	selected              int
	history               []string
	historyIndex          int
	draft                 string
	hint, notice          string
	width, height         int
	busy, quitting        bool
	exitCode              int
}
type shellDone struct {
	command *shellExecution
	err     error
}
type shellExecution struct {
	runner *app
	tree   *cobra.Command
	parent *app
	ctx    context.Context
	args   []string
	line   string
	code   int
	path   string
}

// Bubble Tea releases its input reader and restores the terminal before Run,
// so only the existing command's secure prompter can read a key or approval.
func (e *shellExecution) Run() error {
	fmt.Fprintln(e.parent.out, e.parent.color("2", "› "+clean(e.line)))
	child, root := e.runner, e.tree
	if err := resetCommandFlags(root); err != nil {
		return err
	}
	cmd, _, _ := root.Find(e.args)
	if cmd != nil {
		e.path = cmd.CommandPath()
	}
	e.code = child.executeCommand(e.ctx, append(e.parent.shellOptions(), e.args...), root)
	return nil
}
func (*shellExecution) SetStdin(io.Reader)  {}
func (*shellExecution) SetStdout(io.Writer) {}
func (*shellExecution) SetStderr(io.Writer) {}
func (a *app) shellOptions() []string {
	args := []string{"--timeout=" + a.timeout.String()}
	if a.network != "" {
		args = append(args, "--network="+a.network)
	}
	if a.account != "" {
		args = append(args, "--account="+a.account)
	}
	if a.noColor {
		args = append(args, "--no-color")
	}
	return args
}
func (a *app) shellChild(ctx context.Context, out io.Writer) *app {
	child := &app{store: a.store, in: a.in, out: out, diag: a.diag, version: a.version, inShell: true, completions: a.completions}
	child.prompt = terminalPrompt{in: a.in, out: a.diag, ctx: ctx}
	return child
}
func (a *app) shellCommand() *cobra.Command {
	return &cobra.Command{Use: "shell", Short: "Open an interactive prompt with live suggestions", Hidden: a.inShell, Args: cobra.NoArgs, Long: "Explore Yellow interactively, with live command and argument suggestions, descriptions and session history. Commands use the same validation, encrypted storage and explicit approval as direct CLI calls. For scripts and agents, use direct commands with --json.", RunE: func(cmd *cobra.Command, _ []string) error {
		if a.inShell {
			return fail(ExitInput, "already in yellow shell; enter a command or exit")
		}
		if a.json {
			return fail(ExitInput, "yellow shell is interactive; use yellow <command> --json for structured output")
		}
		out, ok := a.out.(*os.File)
		if a.in == nil || !term.IsTerminal(int(a.in.Fd())) || !ok || !term.IsTerminal(int(out.Fd())) {
			return fail(ExitInput, "yellow shell needs an interactive input and output terminal; use direct commands for scripts")
		}
		if a.timeout <= 0 {
			return fail(ExitInput, "--timeout must be positive")
		}
		model := newShellModel(a, cmd.Context())
		if err := model.refresh(); err != nil {
			return err
		}
		fmt.Fprintln(a.out, a.color("1;33", "YELLOW · interactive shell"))
		fmt.Fprintln(a.out, "Type a command below. Suggestions are local; transfers always require approval.")
		program := tea.NewProgram(model, tea.WithInput(a.in), tea.WithOutput(a.out), tea.WithContext(cmd.Context()), tea.WithoutSignalHandler())
		final, err := program.Run()
		if cmd.Context().Err() != nil {
			return fail(ExitCancelled, "shell cancelled")
		}
		if err != nil {
			return fmt.Errorf("interactive shell: %w", err)
		}
		if final.(*shellModel).exitCode != 0 {
			return &failure{code: final.(*shellModel).exitCode, message: "shell cancelled"}
		}
		fmt.Fprintln(a.out, "Session ended.")
		return nil
	}}
}
func newShellModel(a *app, ctx context.Context) *shellModel {
	input := textinput.New()
	input.Prompt = "› "
	input.Placeholder = "Try balance, account, transfer or help"
	input.CharLimit = 4096
	input.TextStyle = lipgloss.NewStyle()
	input.PromptStyle = lipgloss.NewStyle()
	input.PlaceholderStyle = lipgloss.NewStyle()
	input.Cursor.Style = lipgloss.NewStyle().Reverse(true)
	input.KeyMap.Paste.SetEnabled(false)
	input.Cursor.SetMode(cursor.CursorStatic)
	input.Focus()
	model := &shellModel{app: a, ctx: ctx, input: input, selected: -1, historyIndex: -1, width: 80, height: 24}
	model.runner = a.shellChild(ctx, a.out)
	model.runTree = model.runner.command()
	model.completer = a.shellChild(ctx, new(bytes.Buffer))
	model.completer.diag = io.Discard
	model.completeTree = model.completer.command()
	return model
}
func (m *shellModel) Init() tea.Cmd { return nil }
func (m *shellModel) refresh() error {
	data, err := m.app.store.completionData()
	if err != nil {
		return err
	}
	if m.app.network != "" {
		data.config.Network = m.app.network
	}
	if m.app.account != "" {
		data.config.Account = m.app.account
	}
	m.data = data
	m.app.completions = data
	m.completer.completions = data
	m.updateSuggestions()
	return nil
}
func (m *shellModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		if msg.Width > 0 {
			m.width = msg.Width
		}
		if msg.Height > 0 {
			m.height = msg.Height
		}
		m.input.Width = max(1, m.width-4)
		return m, nil
	case shellDone:
		m.busy = false
		if msg.err != nil {
			m.notice = "Could not resume terminal: " + clean(msg.err.Error())
		} else {
			switch msg.command.code {
			case 0:
				m.notice = "Ready"
			case ExitInput:
				m.notice = "Check the command arguments"
			case ExitPending:
				m.notice = "Pending · execution is not confirmed"
			case ExitUnknown:
				m.notice = "Unresolved · check the saved operation before sending again"
			case ExitRejected:
				m.notice = "Rejected · review the operation"
			case ExitNetwork:
				m.notice = "Gateway or local index unavailable"
			case ExitCancelled:
				m.notice = "Cancelled"
			}
			if msg.command.code != ExitInput && msg.command.line != "" {
				m.remember(msg.command.line)
			}
			if msg.command.code == 0 {
				switch msg.command.path {
				case "yellow network use":
					m.app.network = ""
				case "yellow account use":
					m.app.account = ""
				}
			}
		}
		if err := m.refresh(); err != nil {
			m.notice = "Local metadata unavailable: " + clean(err.Error())
		}
		return m, nil
	case tea.KeyMsg:
		if m.busy {
			return m, nil
		}
		if msg.Paste {
			// Pasting multiple lines must never execute a command or feed an approval.
			for i, r := range msg.Runes {
				if unicode.IsControl(r) {
					msg.Runes[i] = ' '
				}
			}
		} else {
			switch msg.String() {
			case "ctrl+d":
				if m.input.Value() != "" {
					var cmd tea.Cmd
					m.input, cmd = m.input.Update(msg)
					m.selected = -1
					m.historyIndex = -1
					m.updateSuggestions()
					return m, cmd
				}
				m.quitting = true
				return m, tea.Quit
			case "ctrl+c":
				if m.input.Value() == "" {
					m.quitting = true
					m.exitCode = ExitCancelled
					return m, tea.Quit
				}
				m.input.Reset()
				m.selected = -1
				m.historyIndex = -1
				m.notice = "Input cleared"
				m.updateSuggestions()
				return m, nil
			case "esc":
				m.selected = -1
				return m, nil
			case "tab":
				m.accept()
				return m, nil
			case "shift+tab":
				if len(m.suggestions) > 0 {
					m.selected = (m.selected - 1 + len(m.suggestions)) % len(m.suggestions)
				}
				return m, nil
			case "down":
				if m.historyIndex >= 0 {
					m.recall(1)
				} else if len(m.suggestions) > 0 {
					m.selected = (m.selected + 1) % len(m.suggestions)
				}
				return m, nil
			case "up":
				if m.selected >= 0 {
					m.selected = (m.selected - 1 + len(m.suggestions)) % len(m.suggestions)
				} else {
					m.recall(-1)
				}
				return m, nil
			case "enter":
				if m.selected >= 0 {
					m.accept()
					return m, nil
				}
				return m.runLine()
			}
		}
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		m.selected = -1
		m.historyIndex = -1
		m.updateSuggestions()
		return m, cmd
	}
	return m, nil
}
func (m *shellModel) remember(line string) {
	if len(m.history) == 0 || m.history[len(m.history)-1] != line {
		m.history = append(m.history, line)
		if len(m.history) > 100 {
			m.history = m.history[1:]
		}
	}
	m.historyIndex = -1
}
func (m *shellModel) recall(direction int) {
	if len(m.history) == 0 {
		return
	}
	if m.historyIndex < 0 {
		if direction > 0 {
			return
		}
		m.draft = m.input.Value()
		m.historyIndex = len(m.history)
	}
	m.historyIndex += direction
	m.historyIndex = max(0, min(len(m.history), m.historyIndex))
	if m.historyIndex == len(m.history) {
		m.input.SetValue(m.draft)
		m.historyIndex = -1
	} else {
		m.input.SetValue(m.history[m.historyIndex])
	}
	m.input.CursorEnd()
	m.updateSuggestions()
}
func (m *shellModel) runLine() (tea.Model, tea.Cmd) {
	line := strings.TrimSpace(m.input.Value())
	if line == "" {
		return m, nil
	}
	words, err := shellWords(line, false)
	if err != nil {
		m.notice = err.Error()
		return m, nil
	}
	args := wordValues(words)
	if len(args) > 0 && args[0] == "yellow" {
		args = args[1:]
	}
	if len(args) == 0 {
		return m, nil
	}
	if len(args) == 1 && (args[0] == "exit" || args[0] == "quit") {
		m.quitting = true
		return m, tea.Quit
	}
	if args[0] == "shell" {
		m.notice = "Already in yellow shell; enter a command or exit"
		return m, nil
	}
	m.input.Reset()
	m.selected = -1
	m.historyIndex = -1
	m.busy = true
	m.suggestions = nil
	execution := &shellExecution{parent: m.app, ctx: m.ctx, args: args, line: line, runner: m.runner, tree: m.runTree}
	return m, tea.Exec(execution, func(err error) tea.Msg { return shellDone{execution, err} })
}
func (m *shellModel) accept() {
	if len(m.suggestions) == 0 {
		return
	}
	index := m.selected
	if index < 0 {
		index = 0
	}
	item := m.suggestions[index]
	line := m.input.Value()
	replacement := quoteWord(item.value)
	if !item.noSpace {
		replacement += " "
	}
	m.input.SetValue(line[:m.start] + replacement + line[m.end:])
	m.input.SetCursor(utf8.RuneCountInString(line[:m.start] + replacement))
	m.selected = -1
	m.historyIndex = -1
	m.updateSuggestions()
}
func (m *shellModel) updateSuggestions() {
	line := m.input.Value()
	pos := len(string([]rune(line)[:m.input.Position()]))
	left, _ := shellWords(line[:pos], true)
	full, _ := shellWords(line, true)
	values := wordValues(left)
	m.start = pos
	m.end = pos
	if len(left) > 0 && left[len(left)-1].end == pos && (left[len(left)-1].open || pos == 0 || !unicode.IsSpace([]rune(line[:pos])[len([]rune(line[:pos]))-1]) || strings.HasSuffix(line[:pos], "\\ ")) {
		m.start = left[len(left)-1].start
		for _, w := range full {
			if w.start == m.start {
				m.end = w.end
				break
			}
		}
	} else {
		values = append(values, "")
	}
	if len(values) == 0 {
		values = []string{""}
	}
	if len(values) > 1 && values[0] == "yellow" {
		values = values[1:]
	}
	prefix := values[len(values)-1]
	child := m.completer
	out := child.out.(*bytes.Buffer)
	out.Reset()
	if err := resetCommandFlags(m.completeTree); err != nil {
		m.notice = err.Error()
		return
	}
	_ = child.executeCommand(m.ctx, append([]string{cobra.ShellCompRequestCmd}, values...), m.completeTree)
	m.suggestions = nil
	directive := cobra.ShellCompDirectiveNoFileComp
	for _, line := range strings.Split(strings.TrimSpace(out.String()), "\n") {
		if strings.HasPrefix(line, ":") {
			n, _ := strconv.Atoi(strings.TrimPrefix(line, ":"))
			directive = cobra.ShellCompDirective(n)
			continue
		}
		value, desc, _ := strings.Cut(line, "\t")
		matchPrefix := prefix
		if flag, p, ok := strings.Cut(prefix, "="); ok && strings.HasPrefix(flag, "--") {
			matchPrefix = p
			value = flag + "=" + value
		}
		matchedValue := value
		if _, p, ok := strings.Cut(value, "="); ok && strings.HasPrefix(value, "--") {
			matchedValue = p
		}
		if value != "" && strings.HasPrefix(matchedValue, matchPrefix) {
			m.suggestions = append(m.suggestions, suggestion{value, clean(desc), false})
		}
	}
	for i := range m.suggestions {
		m.suggestions[i].noSpace = directive&cobra.ShellCompDirectiveNoSpace != 0 || strings.HasSuffix(m.suggestions[i].value, "=")
	}
	if len(values) == 1 && strings.HasPrefix("exit", prefix) {
		m.suggestions = append(m.suggestions, suggestion{"exit", "Close the interactive shell", false})
	}
	if directive&(cobra.ShellCompDirectiveNoFileComp|cobra.ShellCompDirectiveError) == 0 && len(values) > 1 && (values[len(values)-2] == "--out" || strings.HasPrefix(prefix, "--out=")) {
		m.suggestions = fileSuggestions(prefix)
	}
	sort.Slice(m.suggestions, func(i, j int) bool { return m.suggestions[i].value < m.suggestions[j].value })
	m.hint = "Tab complete · ↓ choose · ↑ history · Enter run · Ctrl+D exit"
	if len(m.suggestions) == 0 {
		tree := m.completeTree
		tree.InitDefaultHelpCmd()
		cmd, _, err := tree.Find(values[:len(values)-1])
		if err == nil && cmd != nil {
			m.hint = clean(cmd.UseLine())
			if len(prefix) >= 2 && !strings.HasPrefix(prefix, "-") {
				for _, name := range cmd.SuggestionsFor(prefix) {
					for _, candidate := range cmd.Commands() {
						if candidate.Name() == name {
							m.suggestions = append(m.suggestions, suggestion{name, "Suggested: " + clean(candidate.Short), false})
						}
					}
				}
			}
		}
	}
}
func fileSuggestions(prefix string) []suggestion {
	flag := ""
	if strings.HasPrefix(prefix, "--out=") {
		flag = "--out="
		prefix = strings.TrimPrefix(prefix, flag)
	}
	dir, base := filepath.Split(prefix)
	search := dir
	if search == "" {
		search = "."
	}
	entries, err := os.ReadDir(search)
	if err != nil {
		return nil
	}
	var items []suggestion
	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), base) {
			continue
		}
		value := dir + entry.Name()
		desc := "Existing file · backup needs a new name"
		noSpace := false
		if entry.IsDir() {
			value += string(os.PathSeparator)
			desc = "Directory"
			noSpace = true
		}
		items = append(items, suggestion{flag + value, desc, noSpace})
		if len(items) == 100 {
			break
		}
	}
	return items
}
func (m *shellModel) View() string {
	if m.quitting || m.busy {
		return ""
	}
	network, account := m.app.network, m.app.account
	if m.data != nil {
		if network == "" {
			network = m.data.config.Network
		}
		if account == "" {
			account = m.data.config.Account
		}
	}
	if network == "" {
		network = "no network"
	}
	if account == "" {
		account = "no account"
	}
	width := max(1, m.width)
	if m.height < 3 {
		return ansi.Truncate(m.input.View(), width, "…")
	}
	var view strings.Builder
	view.WriteString(ansi.Truncate(m.app.color("1;33", "YELLOW")+"  "+clean(network)+" / "+clean(account), width, "…") + "\n")
	view.WriteString(ansi.Truncate(m.input.View(), width, "…") + "\n")
	rows := max(0, min(6, m.height-5))
	offset := 0
	if m.selected >= rows && rows > 0 {
		offset = m.selected - rows + 1
	}
	for i := 0; i < rows; i++ {
		n := offset + i
		row := ""
		if n < len(m.suggestions) {
			item := m.suggestions[n]
			mark := "  "
			if n == m.selected {
				mark = "› "
			}
			label := fmt.Sprintf("%s%-18s %s", mark, clean(item.value), item.description)
			if n == m.selected {
				row = m.app.color("1;36", label)
			} else {
				row = fmt.Sprintf("%s%-18s %s", mark, clean(item.value), m.app.color("2", item.description))
			}
		}
		view.WriteString(ansi.Truncate(row, width, "…") + "\n")
	}
	hint := m.hint
	if m.selected >= 0 {
		hint = "Enter or Tab choose · Esc back · ↑↓ navigate"
	}
	view.WriteString(ansi.Truncate(m.app.color("2", hint), width, "…"))
	if m.notice != "" {
		view.WriteString("\n" + ansi.Truncate(clean(m.notice), width, "…"))
	}
	return view.String()
}

var _ tea.ExecCommand = (*shellExecution)(nil)

// Reuse the registered flag pointers: Cobra keeps completion callbacks in a
// global map, so rebuilding trees for every keypress would retain every tree.
func resetCommandFlags(root *cobra.Command) error {
	seen := map[*pflag.Flag]bool{}
	var reset func(*cobra.Command) error
	reset = func(cmd *cobra.Command) error {
		var local, persistent []*pflag.Flag
		cmd.LocalNonPersistentFlags().VisitAll(func(flag *pflag.Flag) { local = append(local, flag) })
		cmd.PersistentFlags().VisitAll(func(flag *pflag.Flag) { persistent = append(persistent, flag) })
		for _, flag := range append(append([]*pflag.Flag{}, local...), persistent...) {
			if seen[flag] {
				continue
			}
			seen[flag] = true
			var err error
			if slice, ok := flag.Value.(pflag.SliceValue); ok && flag.DefValue == "[]" {
				err = slice.Replace(nil)
			} else {
				err = flag.Value.Set(flag.DefValue)
			}
			if err != nil {
				return fmt.Errorf("reset --%s: %w", flag.Name, err)
			}
			flag.Changed = false
		}
		cmd.ResetFlags()
		for _, flag := range local {
			cmd.Flags().AddFlag(flag)
		}
		for _, flag := range persistent {
			cmd.PersistentFlags().AddFlag(flag)
		}
		for _, child := range cmd.Commands() {
			if err := reset(child); err != nil {
				return err
			}
		}
		return nil
	}
	return reset(root)
}

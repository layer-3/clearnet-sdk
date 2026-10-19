package yellow

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

func TestShellWords(t *testing.T) {
	for _, test := range []struct {
		line string
		want []string
	}{
		{`account   backup alice --out 'my wallet backup.json'`, []string{"account", "backup", "alice", "--out", "my wallet backup.json"}},
		{`account backup alice --out "my wallet backup.json"`, []string{"account", "backup", "alice", "--out", "my wallet backup.json"}},
		{`account backup alice --out my\ backup.json`, []string{"account", "backup", "alice", "--out", "my backup.json"}},
		{`account backup alice --out 'C:\Users\Alice\backup.json'`, []string{"account", "backup", "alice", "--out", `C:\Users\Alice\backup.json`}},
		{`account backup alice --out "C:\Users\Alice\backup.json"`, []string{"account", "backup", "alice", "--out", `C:\Users\Alice\backup.json`}},
		{`version ; exit`, []string{"version", ";", "exit"}},
		{`balance "$ACCOUNT"`, []string{"balance", "$ACCOUNT"}},
		{`account address ""`, []string{"account", "address", ""}},
		{`account backup alice --out 'a'\''b.json'`, []string{"account", "backup", "alice", "--out", "a'b.json"}},
	} {
		words, err := shellWords(test.line, false)
		if err != nil {
			t.Fatal(test.line, err)
		}
		if got := wordValues(words); !reflect.DeepEqual(got, test.want) {
			t.Fatalf("%s: got %q want %q", test.line, got, test.want)
		}
	}
	for _, line := range []string{`account backup alice --out 'unfinished`, "account backup alice --out x\\"} {
		if _, err := shellWords(line, false); err == nil {
			t.Fatal("accepted unfinished argument", line)
		}
		if _, err := shellWords(line, true); err != nil {
			t.Fatal("partial completion rejected", err)
		}
	}
	for _, word := range []string{"plain", "with spaces", "a'b", `C:\file name.json`, "£账本"} {
		words, err := shellWords(quoteWord(word), false)
		if err != nil || len(words) != 1 || words[0].value != word {
			t.Fatalf("quote roundtrip %q %v", word, err)
		}
	}
}
func shellForTest(t *testing.T, dir string) *shellModel {
	t.Helper()
	a, _, _ := newTestApp(dir, &scriptedPrompt{approve: true})
	a.timeout = time.Second
	m := newShellModel(a, context.Background())
	if err := m.refresh(); err != nil {
		t.Fatal(err)
	}
	return m
}
func setShellLine(m *shellModel, line string) {
	m.input.SetValue(line)
	m.input.CursorEnd()
	m.selected = -1
	m.updateSuggestions()
}
func hasSuggestion(m *shellModel, value string) bool {
	for _, item := range m.suggestions {
		if item.value == value {
			return true
		}
	}
	return false
}
func TestShellCompletionReusesCommandsWithoutIO(t *testing.T) {
	g := newGateway(t)
	dir, _ := setup(t, g)
	m := shellForTest(t, dir)
	for _, test := range []struct{ line, want string }{
		{"", "balance"}, {"acc", "account"}, {"balanc", "balance"}, {"account creat", "create"}, {"account ", "backup"}, {"account use ", "alice"},
		{"network use ", "dev"}, {"balance ", m.data.accounts[0].address},
		{"transfer send ", m.data.accounts[0].address},
		{"transfer send " + recipient + " ", "--asset"},
		{"transfer send " + recipient + " --asset Y", "YELLOW="},
		{"transfer send " + recipient + " --asset=Y", "--asset=YELLOW="},
		{"transfer send " + recipient + " --from ", "alice"},
		{"--network ", "dev"}, {"--network=d", "--network=dev"},
		{"yellow account use a", "alice"}, {"ex", "exit"},
		{"account export", "export-key"}, {"account export-key a", "alice"},
	} {
		setShellLine(m, test.line)
		if !hasSuggestion(m, test.want) {
			t.Fatalf("%q: want %q, got %+v", test.line, test.want, m.suggestions)
		}
	}
	setShellLine(m, "account use alice ")
	if hasSuggestion(m, "alice") {
		t.Fatal("positional argument was suggested twice")
	}
	// Delete the source files: suggestions keep using the public snapshot.
	if err := os.RemoveAll(dir); err != nil {
		t.Fatal(err)
	}
	setShellLine(m, "account use a")
	if !hasSuggestion(m, "alice") {
		t.Fatal("completion re-read account files")
	}
	if g.infoCalls != 0 || len(g.posted) != 0 {
		t.Fatal("typing made gateway calls")
	}
	for _, line := range []string{testKey, testPass} {
		if strings.Contains(m.View(), line) {
			t.Fatal("secret in shell suggestions")
		}
	}
}
func TestShellSelectionDoesNotExecute(t *testing.T) {
	m := shellForTest(t, t.TempDir())
	setShellLine(m, "acc")
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if cmd != nil || m.selected < 0 {
		t.Fatal("could not select suggestion")
	}
	_, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil || m.input.Value() != "account " || m.busy {
		t.Fatal("choosing suggestion executed a command")
	}
	setShellLine(m, "version")
	_, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil || !m.busy {
		t.Fatal("Enter did not dispatch command")
	}
}
func TestShellCompleteAtCursorAndQuotedPaths(t *testing.T) {
	g := newGateway(t)
	dir, _ := setup(t, g)
	m := shellForTest(t, dir)
	setShellLine(m, "account use al --no-color")
	m.input.SetCursor(len("account use al"))
	m.updateSuggestions()
	m.accept()
	if got := m.input.Value(); got != "account use alice  --no-color" {
		t.Fatal("completion changed following arguments", got)
	}
	temp := t.TempDir()
	path := filepath.Join(temp, "my wallet backup.json")
	if err := os.WriteFile(path, nil, 0600); err != nil {
		t.Fatal(err)
	}
	line := "account backup alice --out '" + filepath.Join(temp, "my wallet ")
	setShellLine(m, line)
	if !hasSuggestion(m, path) {
		t.Fatalf("unfinished quoted path completion: %+v", m.suggestions)
	}
	m.accept()
	words, err := shellWords(m.input.Value(), false)
	if err != nil || words[len(words)-1].value != path {
		t.Fatal("quoted path was broken", m.input.Value(), err)
	}
	setShellLine(m, "transfer send "+recipient+" --asset Y")
	m.accept()
	if !strings.HasSuffix(m.input.Value(), "YELLOW=") {
		t.Fatal("asset completion appended a space", m.input.Value())
	}
}
func TestShellHistoryPasteAndCancellation(t *testing.T) {
	m := shellForTest(t, t.TempDir())
	m.remember("version")
	m.remember("account list")
	setShellLine(m, "bal")
	m.recall(-1)
	if m.input.Value() != "account list" {
		t.Fatal("history missing")
	}
	m.recall(-1)
	if m.input.Value() != "version" {
		t.Fatal("older history missing")
	}
	m.recall(1)
	m.recall(1)
	if m.input.Value() != "bal" {
		t.Fatal("history lost draft")
	}
	m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if m.input.Value() != "" || m.quitting {
		t.Fatal("Ctrl+C did not clear draft")
	}
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("version\nexit\r"), Paste: true})
	if cmd != nil || m.busy || m.quitting {
		t.Fatal("paste executed commands")
	}
	if strings.ContainsAny(m.input.Value(), "\r\n") {
		t.Fatal("paste retained command separators")
	}
	setShellLine(m, `account backup alice --out 'unfinished`)
	_, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil || m.busy || !strings.Contains(m.notice, "quoted") {
		t.Fatal("unfinished quote executed")
	}
	setShellLine(m, "versionx")
	m.input.SetCursor(len("version"))
	m.updateSuggestions()
	m.selected = 0
	m.Update(tea.KeyMsg{Type: tea.KeyCtrlD})
	if m.quitting || m.input.Value() != "version" || m.selected != -1 {
		t.Fatal("Ctrl+D did not edit the current command safely")
	}
	setShellLine(m, "exit")
	_, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil || !m.quitting {
		t.Fatal("exit did not close shell")
	}
}
func TestShellCallsExistingCommandsAndRefreshesContext(t *testing.T) {
	g := newGateway(t)
	dir, _ := setup(t, g)
	a, out, _ := newTestApp(dir, &scriptedPrompt{})
	a.timeout = time.Second
	m := newShellModel(a, context.Background())
	if err := m.refresh(); err != nil {
		t.Fatal(err)
	}
	run := &shellExecution{parent: a, ctx: context.Background(), args: []string{"--json", "account", "list"}, line: "--json account list", runner: m.runner, tree: m.runTree}
	if err := run.Run(); err != nil {
		t.Fatal(err)
	}
	if run.code != 0 {
		t.Fatal("direct command failed", out)
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	assertOneJSON(t, lines[len(lines)-1])
	m.Update(shellDone{command: run})
	if len(m.history) != 1 || !strings.Contains(m.View(), "dev / alice") {
		t.Fatal("context/history not refreshed")
	}
	a.network = "override"
	run.path = "yellow network use"
	m.Update(shellDone{command: run})
	if a.network != "" {
		t.Fatal("network use did not replace startup selector")
	}
	a.json = true
	out.Reset()
	a.diag = io.Discard
	code := a.execute(context.Background(), []string{"--json", "shell"})
	if code != ExitInput {
		t.Fatal("JSON shell should fail predictably", code)
	}
	assertOneJSON(t, out.String())
	a, out, _ = newTestApp(dir, &scriptedPrompt{})
	if code := a.execute(context.Background(), []string{"shell"}); code != ExitInput {
		t.Fatal("non-TTY shell accepted", code, out)
	}
}
func TestShellViewFitsSmallTerminalAndNoColor(t *testing.T) {
	m := shellForTest(t, t.TempDir())
	m.app.noColor = true
	m.Update(tea.WindowSizeMsg{Width: 28, Height: 8})
	m.notice = "A deliberately long status explanation for a narrow window"
	view := m.View()
	for _, line := range strings.Split(view, "\n") {
		if ansi.StringWidth(line) > 28 {
			t.Fatal("shell exceeds terminal width", line)
		}
	}
	if strings.Contains(view, "\x1b[1;33m") || strings.Contains(view, "\x1b[1;36m") {
		t.Fatal("--no-color ignored")
	}
	if bytes.Contains([]byte(view), []byte("\x1b[?1049")) {
		t.Fatal("shell uses alternate screen")
	}
	m.Update(tea.WindowSizeMsg{Width: 16, Height: 1})
	if len(strings.Split(m.View(), "\n")) > 1 {
		t.Fatal("shell exceeds a one-line terminal")
	}
}

func TestShellReusesTreesAndResetsEveryFlag(t *testing.T) {
	g := newGateway(t)
	dir, _ := setup(t, g)
	m := shellForTest(t, dir)
	tree := m.completeTree
	setShellLine(m, "transfer send "+recipient+" --asset ETH=1 --asset ")
	if !hasSuggestion(m, "YELLOW=") {
		t.Fatal("repeat asset completion missing")
	}
	setShellLine(m, "transfer send "+recipient+" ")
	if !hasSuggestion(m, "--asset") {
		t.Fatal("completion retained changed flags")
	}
	for _, line := range []string{"--network wrong account use a", "account use a", "network use d", "transfer send " + recipient + " --asset=Y"} {
		setShellLine(m, line)
		if m.completeTree != tree {
			t.Fatal("typing rebuilt the command tree")
		}
	}
	out := m.app.out.(*bytes.Buffer)
	for i, args := range [][]string{{"--json", "account", "list"}, {"version"}, {"account", "address"}, {"version"}} {
		out.Reset()
		run := &shellExecution{parent: m.app, ctx: context.Background(), args: args, line: strings.Join(args, " "), runner: m.runner, tree: m.runTree}
		if err := run.Run(); err != nil || run.code != 0 {
			t.Fatal(i, run.code, err, out)
		}
		if i == 0 {
			lines := strings.Split(strings.TrimSpace(out.String()), "\n")
			assertOneJSON(t, lines[len(lines)-1])
		} else if strings.Contains(out.String(), `"version":1`) {
			t.Fatal("JSON flag leaked to next command", out)
		}
	}
	send, _, err := m.runTree.Find([]string{"transfer", "send"})
	if err != nil {
		t.Fatal(err)
	}
	if err := send.ParseFlags([]string{"--asset", "YELLOW=1", "--max-fee", "0"}); err != nil {
		t.Fatal(err)
	}
	if err := resetCommandFlags(m.runTree); err != nil {
		t.Fatal(err)
	}
	values, _ := send.Flags().GetStringArray("asset")
	if len(values) != 0 || send.Flags().Changed("max-fee") {
		t.Fatal("transfer flags or asset array survived command reset")
	}
}

func TestShellTransfersKeepApprovalFeeAndRecoveryGuards(t *testing.T) {
	g := newGateway(t)
	g.postCode = 202
	dir, _ := setup(t, g)
	m := shellForTest(t, dir)
	m.runner.prompt = &scriptedPrompt{approve: true}
	execute := func(args []string) int {
		run := &shellExecution{parent: m.app, ctx: context.Background(), args: args, line: strings.Join(args, " "), runner: m.runner, tree: m.runTree}
		if err := run.Run(); err != nil {
			t.Fatal(err)
		}
		m.Update(shellDone{command: run})
		return run.code
	}
	if got := execute([]string{"transfer", "send", recipient, "--asset", "YELLOW=1", "--max-fee", "0"}); got != ExitPending {
		t.Fatal("shell announced pending transfer as success", got)
	}
	if !strings.Contains(m.notice, "not confirmed") {
		t.Fatal("pending status missing from prompt")
	}
	if got := execute([]string{"transfer", "send", recipient, "--asset", "ETH=1"}); got != ExitInput {
		t.Fatal("fee cap leaked from previous command", got)
	}
	if got := execute([]string{"transfer", "send", recipient, "--asset", "YELLOW=1", "--max-fee", "0"}); got != ExitUnknown {
		t.Fatal("shell bypassed unresolved operation guard", got)
	}
	if len(g.posted) != 1 {
		t.Fatal("shell submitted another transfer while pending")
	}
}

func TestAgentReadsRemainNoninteractiveJSON(t *testing.T) {
	g := newGateway(t)
	dir, _ := setup(t, g)
	for _, args := range [][]string{{"--json", "network", "list"}, {"--json", "account", "list"}, {"--json", "balance", recipient}, {"--json", "operation", "list"}, {"--json", "version"}} {
		a, out, diag := newTestApp(dir, &scriptedPrompt{approve: false})
		if code := a.execute(context.Background(), args); code != 0 {
			t.Fatal(args, code, out, diag)
		}
		assertOneJSON(t, out.String())
		if diag.Len() != 0 || strings.Contains(out.String(), "interactive shell") || strings.Contains(out.String(), "\x1b") {
			t.Fatal("agent read emitted interactive UI", args, out, diag)
		}
	}
}

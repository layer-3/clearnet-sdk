package yellow

import (
	"sort"

	"github.com/spf13/cobra"
)

type publicAccount struct{ name, address string }
type completionData struct {
	config     config
	accounts   []publicAccount
	operations []string
}

func (s store) completionData() (*completionData, error) {
	cfg, err := s.config()
	if err != nil {
		return nil, err
	}
	data := &completionData{config: cfg}
	names, err := s.accounts()
	if err != nil {
		return nil, err
	}
	for _, name := range names {
		ac, err := s.account(name)
		if err != nil {
			return nil, err
		}
		data.accounts = append(data.accounts, publicAccount{name, ac.Address})
		for _, op := range ac.Operations {
			data.operations = append(data.operations, op.ID+"\t"+op.State+" · "+op.Profile.Name)
		}
	}
	sort.Strings(data.operations)
	return data, nil
}
func (a *app) completionConfig() (config, error) {
	if a.completions != nil {
		return a.completions.config, nil
	}
	return a.store.config()
}
func (a *app) completionAccounts() ([]publicAccount, error) {
	if a.completions != nil {
		return a.completions.accounts, nil
	}
	names, err := a.store.accounts()
	if err != nil {
		return nil, err
	}
	var accounts []publicAccount
	for _, name := range names {
		ac, err := a.store.account(name)
		if err != nil {
			return nil, err
		}
		accounts = append(accounts, publicAccount{name, ac.Address})
	}
	return accounts, nil
}

// Flag completion remains valid anywhere; positional completion stops once
// the command's single optional/required argument has already been supplied.
func oneArgument(fn func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective)) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, prefix string) ([]string, cobra.ShellCompDirective) {
		if len(args) > 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return fn(cmd, args, prefix)
	}
}

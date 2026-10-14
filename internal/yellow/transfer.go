package yellow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"sort"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/layer-3/clearnet-sdk/pkg/client"
	"github.com/layer-3/clearnet-sdk/pkg/core"
	"github.com/spf13/cobra"
)

func userAddress(raw string) (string, error) {
	if strings.Contains(raw, "://") {
		if err := core.ValidateURI(raw); err != nil {
			return "", fail(ExitInput, "use a canonical yellow://ynet/user/<address> URI")
		}
		_, kind, path, err := core.ParseURI(raw)
		if err != nil || kind != "user" {
			return "", fail(ExitInput, "recipient must be a user account")
		}
		raw = path
	}
	if !common.IsHexAddress(raw) || common.HexToAddress(raw) == (common.Address{}) {
		return "", fail(ExitInput, "use a full, nonzero Ethereum user address")
	}
	return common.HexToAddress(raw).Hex(), nil
}
func displayUnits(raw string, decimals uint8) string {
	v, ok := new(big.Int).SetString(raw, 10)
	if !ok || v.Sign() < 0 {
		return "invalid balance"
	}
	digits := v.String()
	d := int(decimals)
	if d == 0 {
		return digits
	}
	if len(digits) <= d {
		digits = strings.Repeat("0", d-len(digits)+1) + digits
	}
	whole, fraction := digits[:len(digits)-d], strings.TrimRight(digits[len(digits)-d:], "0")
	if fraction == "" {
		return whole
	}
	return whole + "." + fraction
}
func (a *app) readAddress(args []string) (string, error) {
	if len(args) > 0 {
		return userAddress(args[0])
	}
	_, ac, err := a.selectedAccount("")
	if err != nil {
		return "", err
	}
	return userAddress(ac.Address)
}
func (a *app) balanceCommand() *cobra.Command {
	command := a.cmd("balance [address]", "View live, spendable Clearnet balances", cobra.MaximumNArgs(1), func(ctx context.Context, _ *cobra.Command, args []string) error {
		address, err := a.readAddress(args)
		if err != nil {
			return err
		}
		p, err := a.selectedNetwork()
		if err != nil {
			return err
		}
		c, err := a.gateway(p, true)
		if err != nil {
			return err
		}
		balance, err := c.Balance(ctx, address)
		if err != nil {
			return netError(err)
		}
		decimals := map[string]uint8{}
		for _, asset := range p.Identity.Assets {
			decimals[asset.Symbol] = asset.Decimals
		}
		rows := [][]string{}
		entries := []map[string]string{}
		for _, entry := range balance.Balances {
			symbol := entry.Symbol
			if symbol == "" {
				symbol = entry.Asset
			}
			d, ok := decimals[symbol]
			if !ok {
				return fail(ExitNetwork, "balance contains an asset outside the reviewed catalogue")
			}
			amount := displayUnits(entry.ClearnetBalance, d)
			if amount == "invalid balance" {
				return fail(ExitNetwork, "gateway returned an invalid balance")
			}
			rows = append(rows, []string{symbol, amount, entry.ClearnetBalance})
			entries = append(entries, map[string]string{"asset": symbol, "amount": amount, "baseUnits": entry.ClearnetBalance})
		}
		sort.Slice(rows, func(i, j int) bool { return rows[i][0] < rows[j][0] })
		sort.Slice(entries, func(i, j int) bool { return entries[i]["asset"] < entries[j]["asset"] })
		return a.emit(map[string]any{"network": p.Name, "address": address, "nonce": balance.Nonce, "balances": entries}, func() {
			a.title("Live balance · " + p.Name)
			fmt.Fprintln(a.out, address)
			fmt.Fprintf(a.out, "Nonce %d\n\n", balance.Nonce)
			a.table([]string{"ASSET", "AVAILABLE", "BASE UNITS"}, rows)
			if len(rows) == 0 {
				fmt.Fprintln(a.out, "No Clearnet balances.")
			}
		})
	})
	command.ValidArgsFunction = oneArgument(a.completeAddresses)
	return command
}
func parseAssets(raw []string, metadata []client.AssetInfo) ([]client.TransferAmount, error) {
	known := map[string]uint8{}
	var names []string
	for _, asset := range metadata {
		known[asset.Symbol] = asset.Decimals
		names = append(names, asset.Symbol)
	}
	seen := map[string]bool{}
	var assets []client.TransferAmount
	if len(raw) == 0 {
		return nil, fail(ExitInput, "add at least one --asset SYMBOL=AMOUNT")
	}
	for _, leg := range raw {
		symbol, amount, ok := strings.Cut(leg, "=")
		d, exists := known[symbol]
		if !ok || !exists {
			return nil, fail(ExitInput, "use --asset SYMBOL=AMOUNT; available assets: %s", strings.Join(names, ", "))
		}
		if seen[symbol] {
			return nil, fail(ExitInput, "asset %s appears twice; combine it into one amount", symbol)
		}
		seen[symbol] = true
		units, err := client.AmountInBaseUnits(amount, d)
		if err != nil {
			return nil, fail(ExitInput, "%s: %s", symbol, err)
		}
		if units.Sign() == 0 {
			return nil, fail(ExitInput, "%s transfer amount must be greater than zero", symbol)
		}
		assets = append(assets, client.TransferAmount{Asset: symbol, Amount: amount})
	}
	return assets, nil
}
func (a *app) reviewTransfer(p profile, name string, intent *client.TransferIntent) {
	fmt.Fprintf(a.diag, "\nReview transfer\n  Network    %s (%s)\n  Gateway    %s\n  Sender     %s · %s\n  Recipient  %s\n  Nonce      %d\n", p.Name, p.NetworkID, clean(p.Gateway), name, intent.From, intent.To, intent.Nonce)
	decimals := map[string]uint8{}
	for _, asset := range p.Identity.Assets {
		decimals[asset.Symbol] = asset.Decimals
	}
	for _, asset := range intent.Assets {
		fmt.Fprintf(a.diag, "  %-10s %s · %s base units\n", asset.Asset, displayUnits(asset.Amount, decimals[asset.Asset]), asset.Amount)
	}
	fmt.Fprintf(a.diag, "  Fee cap    %s YELLOW · %s base units\n", displayUnits(intent.MaxFee, decimals["YELLOW"]), intent.MaxFee)
}
func opAAD(op operation) string {
	raw, _ := json.Marshal(struct {
		ID, Sender string
		Nonce      uint64
		Profile    profile
		Created    time.Time
	}{op.ID, op.Sender, op.Nonce, op.Profile, op.Created})
	return "yellow:operation:v1:" + string(raw)
}
func (a *app) transferCommands() *cobra.Command {
	parent := group("transfer", "Send a reviewed transfer or view recent activity")
	var legs []string
	var maxFee, from string
	var wait time.Duration
	send := a.cmd("send <recipient>", "Review, sign and submit one transfer", cobra.ExactArgs(1), func(ctx context.Context, _ *cobra.Command, args []string) error {
		recipient, err := userAddress(args[0])
		if err != nil {
			return err
		}
		p, err := a.selectedNetwork()
		if err != nil {
			return err
		}
		if wait < 0 {
			return fail(ExitInput, "--wait cannot be negative")
		}
		if from != "" && a.account != "" && from != a.account {
			return fail(ExitInput, "--from and --account must select the same account")
		}
		name, ac, err := a.selectedAccount(from)
		if err != nil {
			return err
		}
		for _, op := range ac.Operations {
			if op.Profile.NetworkID == p.NetworkID && op.State != "sealed" && op.State != "rejected" {
				return fail(ExitUnknown, "operation %s is %s; reconcile it with yellow operation show %s before sending again", op.ID, op.State, op.ID)
			}
		}
		assets, err := parseAssets(legs, p.Identity.Assets)
		if err != nil {
			return err
		}
		yellowDecimals, found := uint8(0), false
		for _, asset := range p.Identity.Assets {
			if asset.Symbol == "YELLOW" {
				yellowDecimals = asset.Decimals
				found = true
			}
		}
		if !found {
			return fail(ExitNetwork, "reviewed catalogue has no YELLOW asset for the fee cap")
		}
		if _, err := client.AmountInBaseUnits(maxFee, yellowDecimals); err != nil {
			return fail(ExitInput, "--max-fee: %s", err)
		}
		c, err := a.gateway(p, true)
		if err != nil {
			return err
		}
		if _, err := c.Info(ctx); err != nil {
			return netError(err)
		}
		signer, key, close, err := a.unlock(ac)
		if err != nil {
			return err
		}
		defer close()
		intent, err := c.PrepareTransfer(ctx, signer, recipient, assets, maxFee)
		if err != nil {
			return netError(err)
		}
		for _, existing := range ac.Operations {
			if existing.Profile.NetworkID == p.NetworkID && existing.State == "sealed" && existing.Nonce >= intent.Nonce {
				return fail(ExitNetwork, "gateway nonce %d has not advanced beyond sealed operation %s; wait for the live account index to catch up", intent.Nonce, existing.ID)
			}
		}
		a.reviewTransfer(p, name, intent)
		if err := a.prompt.confirm("Sign and send this transfer"); err != nil {
			return err
		}
		signed, err := c.SignTransfer(ctx, signer, intent)
		if err != nil {
			return netError(err)
		}
		now := time.Now().UTC()
		op := operation{ID: signed.OperationID, Account: name, Sender: intent.From, Nonce: intent.Nonce, Profile: p, Created: now, Updated: now, State: "signed_unsubmitted"}
		raw, err := json.Marshal(signed)
		if err != nil {
			return err
		}
		defer clear(raw)
		op.Payload, err = seal(key, raw, opAAD(op))
		if err != nil {
			return err
		}
		ac.Operations = append(ac.Operations, op)
		// This durable commit is also the nonce reservation. The OS lock survives
		// for the entire command and is released automatically on process death.
		if err := a.store.saveAccount(name, ac); err != nil {
			return fmt.Errorf("could not save signed authorization; nothing submitted: %w", err)
		}
		return a.submit(ctx, c, name, ac, len(ac.Operations)-1, signed, wait)
	})
	send.Example = "  yellow transfer send 0x00000000000000000000000000000000000000bb --asset YELLOW=1.25 --max-fee 0.01\n  yellow transfer send <address> --asset ETH=0.1 --asset YELLOW=5 --max-fee 0 --wait 20s"
	send.Flags().StringArrayVar(&legs, "asset", nil, "Exact amount, SYMBOL=AMOUNT (repeat for different assets)")
	send.Flags().StringVar(&maxFee, "max-fee", "", "Required YELLOW fee cap; 0 authorizes zero fees")
	send.Flags().StringVar(&from, "from", "", "Local account to sign with")
	send.Flags().DurationVar(&wait, "wait", 0, "Optionally poll for a seal for this long")
	_ = send.MarkFlagRequired("max-fee")
	_ = send.MarkFlagRequired("asset")
	_ = send.RegisterFlagCompletionFunc("from", a.completeAccounts)
	_ = send.RegisterFlagCompletionFunc("asset", func(_ *cobra.Command, _ []string, prefix string) ([]string, cobra.ShellCompDirective) {
		cfg, err := a.completionConfig()
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		name := a.network
		if name == "" {
			name = cfg.Network
		}
		var candidates []string
		for _, v := range cfg.Networks[name].Identity.Assets {
			candidates = append(candidates, v.Symbol+"=\t"+fmt.Sprint(v.Decimals)+" decimal places")
		}
		return candidates, cobra.ShellCompDirectiveNoSpace | cobra.ShellCompDirectiveNoFileComp
	})
	var limit int
	var cursor string
	history := a.cmd("history [address]", "View the gateway's recent transfer cache", cobra.MaximumNArgs(1), func(ctx context.Context, _ *cobra.Command, args []string) error {
		if limit < 1 || limit > 100 {
			return fail(ExitInput, "--limit must be between 1 and 100")
		}
		address, err := a.readAddress(args)
		if err != nil {
			return err
		}
		p, err := a.selectedNetwork()
		if err != nil {
			return err
		}
		c, err := a.gateway(p, true)
		if err != nil {
			return err
		}
		page, err := c.Transfers(ctx, address, client.TransferPageOptions{Limit: limit, Cursor: cursor})
		if err != nil {
			return fail(ExitNetwork, "recent history is unavailable or backfilling: %s", err)
		}
		for i := range page.Items {
			if incoming(page.Items[i].Direction) {
				page.Items[i].Delivery = "sender sealed; recipient credit not verified"
			}
		}
		return a.emit(map[string]any{"network": p.Name, "address": address, "history": page, "spendability": "live balance only"}, func() {
			a.title("Recent transfers · " + p.Name)
			fmt.Fprintf(a.out, "%d-day gateway cache · not an archive\nLive balance determines spendability.\n\n", page.RetentionDays)
			if len(page.Items) == 0 {
				fmt.Fprintln(a.out, "No transfers in this page of the recent cache.")
			}
			for _, item := range page.Items {
				fmt.Fprintf(a.out, "%s  %s\n  %s → %s\n  Block %s · entry %d · nonce %d\n", clean(item.Direction), clean(item.OperationID), clean(item.From), clean(item.To), clean(item.BlockHash), item.EntryIndex, item.Nonce)
				for _, v := range item.Assets {
					amount := ""
					for _, asset := range p.Identity.Assets {
						if asset.Symbol == v.Asset {
							amount = displayUnits(v.Amount, asset.Decimals) + " " + asset.Symbol + " · "
						}
					}
					fmt.Fprintf(a.out, "  %s%s base units\n", clean(amount), clean(v.Amount))
				}
				if incoming(item.Direction) {
					fmt.Fprintln(a.out, "  Sender sealed; recipient credit not verified.")
				}
				fmt.Fprintln(a.out)
			}
			if page.NextCursor != nil {
				fmt.Fprintf(a.out, "Next page: yellow transfer history %s --limit %d --cursor %q\n", address, limit, *page.NextCursor)
			}
		})
	})
	history.Flags().IntVar(&limit, "limit", 20, "Transfers per page (1–100)")
	history.Flags().StringVar(&cursor, "cursor", "", "Opaque cursor from the previous page")
	send.ValidArgsFunction = oneArgument(a.completeAddresses)
	history.ValidArgsFunction = oneArgument(a.completeAddresses)
	parent.AddCommand(send, history)
	return parent
}
func incoming(direction string) bool {
	return direction == "incoming" || direction == "inbound" || direction == "in"
}
func (a *app) submit(ctx context.Context, c *client.Client, name string, ac *accountFile, index int, signed *client.SignedTransfer, wait time.Duration) error {
	op := &ac.Operations[index]
	// Commit uncertainty BEFORE POST: a crash after a send cannot leave an
	// apparently unsubmitted authorization that may be retried immediately.
	priorState, priorSubmitted := op.State, op.Submitted
	now := time.Now().UTC()
	op.State = "unresolved"
	op.Submitted = &now
	op.Updated = now
	if err := a.store.saveAccount(name, ac); err != nil {
		return fail(ExitUnknown, "operation %s saved; could not commit submission marker, so no POST was made: %s", op.ID, err)
	}
	result, err := c.SubmitTransfer(ctx, signed)
	if err != nil {
		var se *client.SubmissionError
		var api *client.APIError
		// Local preflight failures are not evidence of gateway rejection.
		if errors.As(err, &se) && !se.Submitted {
			op.State = priorState
			op.Submitted = priorSubmitted
		} else if errors.As(err, &se) && !se.OutcomeUnknown && errors.As(err, &api) && api.Code != 0 && api.Message != "unrecognized gateway error" {
			op.State = "rejected"
		}
	} else {
		op.State = result.Status
		op.Receipt = result.Receipt
	}
	op.Updated = time.Now().UTC()
	if saveErr := a.store.saveAccount(name, ac); saveErr != nil {
		return fail(ExitUnknown, "operation %s may have executed; failed to save the observed outcome: %s; run yellow operation show %s", op.ID, saveErr, op.ID)
	}
	if op.State == "pending" && wait > 0 {
		pollctx, cancel := context.WithTimeout(ctx, wait)
		defer cancel()
		for op.State == "pending" {
			timer := time.NewTimer(time.Second)
			select {
			case <-pollctx.Done():
				timer.Stop()
				return a.operationResult(*op)
			case <-timer.C:
			}
			status, queryErr := c.Operation(pollctx, op.ID)
			if queryErr != nil {
				break
			}
			if err := observe(op, status); err != nil {
				break
			}
			if err := a.store.saveAccount(name, ac); err != nil {
				return fail(ExitUnknown, "operation %s may have executed; journal update failed: %s", op.ID, err)
			}
		}
	}
	return a.operationResult(*op)
}

package yellow

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"regexp"
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
	known := map[string]bool{}
	var names []string
	for _, asset := range metadata {
		known[asset.Symbol] = true
		names = append(names, asset.Symbol)
	}
	seen := map[string]bool{}
	var assets []client.TransferAmount
	if len(raw) == 0 {
		return nil, fail(ExitInput, "add at least one --asset SYMBOL=AMOUNT")
	}
	for _, leg := range raw {
		symbol, amount, ok := strings.Cut(leg, "=")
		if !ok || !known[symbol] {
			return nil, fail(ExitInput, "use --asset SYMBOL=AMOUNT; available assets: %s", strings.Join(names, ", "))
		}
		if seen[symbol] {
			return nil, fail(ExitInput, "asset %s appears twice; combine it into one amount", symbol)
		}
		seen[symbol] = true
		assets = append(assets, client.TransferAmount{Asset: symbol, Amount: amount})
	}
	return assets, nil
}
func (a *app) reviewTransfer(p profile, name string, intent *client.TransferIntent) {
	writeReview(a.diag, "Review transfer", p, name, intent)
}
func writeReview(w io.Writer, title string, p profile, name string, intent *client.TransferIntent) {
	fmt.Fprintf(w, "\n%s\n  Network    %s (%s)\n  Gateway    %s\n  Sender     %s · %s\n  Recipient  %s\n  Nonce      %d\n", title, clean(p.Name), clean(p.NetworkID), clean(p.Gateway), clean(name), clean(intent.From), clean(intent.To), intent.Nonce)
	decimals := assetDecimals(p)
	for _, asset := range intent.Assets {
		fmt.Fprintf(w, "  %-10s %s · %s base units\n", asset.Asset, displayUnits(asset.Amount, decimals[asset.Asset]), asset.Amount)
	}
	fmt.Fprintf(w, "  Fee cap    %s YELLOW · %s base units\n", displayUnits(intent.MaxFee, decimals["YELLOW"]), intent.MaxFee)
}
func assetDecimals(p profile) map[string]uint8 {
	decimals := map[string]uint8{}
	for _, asset := range p.Identity.Assets {
		decimals[asset.Symbol] = asset.Decimals
	}
	return decimals
}

// baseUnitLegs converts validated amounts to sorted base-unit legs and fee.
func baseUnitLegs(p profile, assets []client.TransferAmount, maxFee string) ([]client.TransferAsset, string, error) {
	decimals := assetDecimals(p)
	legs := make([]client.TransferAsset, 0, len(assets))
	for _, v := range assets {
		units, err := client.AmountInBaseUnits(v.Amount, decimals[v.Asset])
		if err != nil {
			return nil, "", fail(ExitInput, "%s: %s", v.Asset, err)
		}
		if units.Sign() == 0 {
			return nil, "", fail(ExitInput, "%s transfer amount must be greater than zero", v.Asset)
		}
		legs = append(legs, client.TransferAsset{Asset: v.Asset, Amount: units.String()})
	}
	sort.Slice(legs, func(i, j int) bool { return legs[i].Asset < legs[j].Asset })
	fee, err := client.AmountInBaseUnits(maxFee, decimals["YELLOW"])
	if err != nil {
		return nil, "", fail(ExitInput, "--max-fee: %s", err)
	}
	return legs, fee.String(), nil
}

var idempotencyPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`)

// requestHash identifies what a transfer asks for, so an idempotency key can
// only ever refer to one transfer.
func requestHash(p profile, to string, legs []client.TransferAsset, fee string) string {
	raw, _ := json.Marshal(struct {
		NetworkID, To string
		Assets        []client.TransferAsset
		MaxFee        string
	}{p.NetworkID, to, legs, fee})
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

// preview reports exactly what a send would sign, without unlocking,
// signing, saving or submitting anything.
func (a *app) preview(ctx context.Context, c *client.Client, p profile, name string, ac *accountFile, to string, legs []client.TransferAsset, fee string) error {
	balance, err := c.Balance(ctx, ac.Address)
	if err != nil {
		return netError(err)
	}
	decimals := assetDecimals(p)
	assets := []map[string]string{}
	for _, leg := range legs {
		assets = append(assets, map[string]string{"asset": leg.Asset, "amount": displayUnits(leg.Amount, decimals[leg.Asset]), "baseUnits": leg.Amount})
	}
	intent := &client.TransferIntent{NetworkID: p.NetworkID, ChainID: p.Identity.ChainID, From: ac.Address, To: to, Assets: legs, MaxFee: fee, Nonce: balance.Nonce}
	return a.emit(map[string]any{
		"dryRun": true, "network": p.Name, "networkId": p.NetworkID, "gateway": p.Gateway,
		"account": name, "agent": ac.Agent, "from": ac.Address, "to": to, "nonce": balance.Nonce,
		"assets": assets, "maxFee": map[string]string{"asset": "YELLOW", "amount": displayUnits(fee, decimals["YELLOW"]), "baseUnits": fee},
	}, func() {
		writeReview(a.out, a.color("1;33", "Transfer preview · nothing sent"), p, name, intent)
	})
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
	var maxFee, from, idempotencyKey string
	var wait time.Duration
	var dryRun bool
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
		if idempotencyKey != "" && !idempotencyPattern.MatchString(idempotencyKey) {
			return fail(ExitInput, "--idempotency-key must be 1–128 letters, digits, dots, colons, dashes or underscores, starting with a letter or digit")
		}
		if from != "" && a.account != "" && from != a.account {
			return fail(ExitInput, "--from and --account must select the same account")
		}
		name, ac, err := a.selectedAccount(from)
		if err != nil {
			return err
		}
		assets, err := parseAssets(legs, p.Identity.Assets)
		if err != nil {
			return err
		}
		if _, found := assetDecimals(p)["YELLOW"]; !found {
			return fail(ExitNetwork, "reviewed catalogue has no YELLOW asset for the fee cap")
		}
		baseLegs, fee, err := baseUnitLegs(p, assets, maxFee)
		if err != nil {
			return err
		}
		hash := requestHash(p, recipient, baseLegs, fee)
		// Dry-run describes the live request; it never returns an earlier saved
		// operation just because its idempotency key matches.
		if !dryRun && idempotencyKey != "" {
			for _, op := range ac.Operations {
				if op.IdempotencyKey != idempotencyKey || op.Profile.NetworkID != p.NetworkID {
					continue
				}
				if op.RequestHash != hash {
					return fail(ExitInput, "idempotency key %s was already used for a different transfer (operation %s)", idempotencyKey, op.ID)
				}
				fmt.Fprintf(a.diag, "Idempotency key %s matches saved operation %s; nothing new was signed or sent\n", idempotencyKey, op.ID)
				return a.operationResult(op)
			}
		}
		c, err := a.gateway(p, true)
		if err != nil {
			return err
		}
		if _, err := c.Info(ctx); err != nil {
			return netError(err)
		}
		// A preview signs nothing, so it must not be blocked by an unrelated
		// unresolved operation on this account and network.
		if dryRun {
			return a.preview(ctx, c, p, name, ac, recipient, baseLegs, fee)
		}
		for _, op := range ac.Operations {
			if op.Profile.NetworkID == p.NetworkID && op.State != "sealed" && op.State != "rejected" {
				return fail(ExitUnknown, "operation %s is %s; reconcile it with yellow operation show %s before sending again", op.ID, op.State, op.ID)
			}
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
		if err := a.approve(ac, "Sign and send this transfer"); err != nil {
			return err
		}
		signed, err := c.SignTransfer(ctx, signer, intent)
		if err != nil {
			return netError(err)
		}
		now := time.Now().UTC()
		op := operation{ID: signed.OperationID, Account: name, Sender: intent.From, Nonce: intent.Nonce, Profile: p, Created: now, Updated: now, State: "signed_unsubmitted", IdempotencyKey: idempotencyKey, RequestHash: hash}
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
	send.Example = "  yellow transfer send 0x00000000000000000000000000000000000000bb --asset YELLOW=1.25 --max-fee 0.01\n  yellow transfer send <address> --asset ETH=0.1 --asset YELLOW=5 --max-fee 0 --wait 20s\n  yellow --json --account assistant transfer send <address> --asset USDC=50 --max-fee 1 --idempotency-key invoice-42 --wait 20s"
	send.Long = "Review, sign and submit one transfer. Personal accounts ask for the passphrase and a confirmation at the terminal; agent accounts (account create --agent) sign without prompts.\n\n--dry-run returns the exact transfer that would be signed, with the live nonce, and sends nothing. --idempotency-key makes a repeated command safe: if this account already saved a transfer with the same key on this network, Yellow returns that operation instead of signing again."
	send.Flags().StringArrayVar(&legs, "asset", nil, "Exact amount, SYMBOL=AMOUNT (repeat for different assets)")
	send.Flags().StringVar(&maxFee, "max-fee", "", "Required YELLOW fee cap; 0 authorizes zero fees")
	send.Flags().StringVar(&from, "from", "", "Local account to sign with")
	send.Flags().DurationVar(&wait, "wait", 0, "Optionally poll for a seal for this long")
	send.Flags().BoolVar(&dryRun, "dry-run", false, "Show the exact transfer without unlocking, signing or sending")
	send.Flags().StringVar(&idempotencyKey, "idempotency-key", "", "Your unique key for this payment; repeating it never sends twice")
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
		} else if priorSubmitted == nil && errors.As(err, &se) && !se.OutcomeUnknown && errors.As(err, &api) && api.Code != 0 && api.Message != "unrecognized gateway error" {
			// Only a first submission can be a definite rejection. A refused
			// replay of an authorization that was already sent proves nothing
			// about the original outcome, so it must stay unresolved.
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

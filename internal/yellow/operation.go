package yellow

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/layer-3/clearnet-sdk/pkg/client"
	"github.com/layer-3/clearnet-sdk/pkg/eip712"
	"github.com/spf13/cobra"
)

type operationView struct {
	ID          string          `json:"operationId"`
	Account     string          `json:"account"`
	Network     string          `json:"network"`
	NetworkID   string          `json:"networkId"`
	Gateway     string          `json:"gateway"`
	Fingerprint string          `json:"profileFingerprint"`
	Sender      string          `json:"sender"`
	Nonce       uint64          `json:"nonce"`
	State       string          `json:"state"`
	Created     time.Time       `json:"createdAt"`
	Updated     time.Time       `json:"updatedAt"`
	Submitted   *time.Time      `json:"submittedAt,omitempty"`
	Receipt     *client.Receipt `json:"receipt,omitempty"`
	Message     string          `json:"message,omitempty"`
}

func viewOperation(op operation) operationView {
	v := operationView{ID: op.ID, Account: op.Account, Network: op.Profile.Name, NetworkID: op.Profile.NetworkID, Gateway: op.Profile.Gateway, Fingerprint: op.Profile.Fingerprint, Sender: op.Sender, Nonce: op.Nonce, State: op.State, Created: op.Created, Updated: op.Updated, Submitted: op.Submitted, Receipt: op.Receipt}
	switch op.State {
	case "signed_unsubmitted":
		v.Message = "Saved safely; not submitted. Use operation retry to submit the saved authorization."
	case "pending":
		v.Message = "Accepted by the gateway; awaiting a seal. Execution is not confirmed."
	case "unresolved":
		v.Message = "This transfer may have executed. Keep this operation ID; do not send the transfer again at a new nonce."
	case "rejected":
		v.Message = "The original gateway definitively rejected this authorization."
	case "sealed":
		v.Message = "Sealed by the network."
	}
	return v
}
func (a *app) operationResult(op operation) error {
	v := viewOperation(op)
	if err := a.emit(v, func() {
		label := map[string]string{"sealed": "Transfer sealed", "pending": "Awaiting a seal", "unresolved": "Outcome unresolved", "signed_unsubmitted": "Saved · not submitted", "rejected": "Transfer rejected"}[op.State]
		if label == "" {
			label = op.State
		}
		a.title(label)
		fmt.Fprintln(a.out, "Operation", op.ID)
		fmt.Fprintf(a.out, "Network   %s (%s)\nGateway   %s\nSender    %s\nNonce     %d\n", op.Profile.Name, op.Profile.NetworkID, clean(op.Profile.Gateway), op.Sender, op.Nonce)
		if op.Receipt != nil {
			fmt.Fprintf(a.out, "Block     %s\nEntry     %d\nSealed at %s\n", clean(op.Receipt.BlockHash), op.Receipt.EntryIndex, time.Unix(op.Receipt.SealedAt, 0).UTC().Format(time.RFC3339))
		}
		fmt.Fprintln(a.out, v.Message)
		if op.State != "sealed" && op.State != "rejected" {
			fmt.Fprintf(a.out, "Check: yellow operation show %s\n", op.ID)
		}
	}); err != nil {
		return err
	}
	code := 0
	switch op.State {
	case "pending":
		code = ExitPending
	case "signed_unsubmitted", "unresolved":
		code = ExitUnknown
	case "rejected":
		code = ExitRejected
	}
	if code != 0 {
		return &failure{code: code, reported: true}
	}
	return nil
}
func observe(op *operation, status *client.OperationStatus) error {
	switch status.Status {
	case "sealed":
		if status.Receipt == nil || status.Receipt.BlockHash == "" || status.Receipt.OperationID != op.ID {
			return fail(ExitNetwork, "gateway reported a seal without a matching verified receipt; outcome remains unresolved")
		}
		op.State = "sealed"
		op.Receipt = status.Receipt
	case "rejected":
		if status.Reason == "" {
			return fail(ExitNetwork, "gateway rejection has no reason; outcome remains unresolved")
		}
		if op.State != "sealed" {
			op.State = "rejected"
		}
	case "pending":
		if op.State != "sealed" && op.State != "rejected" {
			op.State = "pending"
		}
	case "unknown":
		if op.Submitted != nil && op.State != "sealed" && op.State != "rejected" {
			op.State = "unresolved"
		}
	}
	op.Updated = time.Now().UTC()
	return nil
}
func (a *app) findOperation(id string) (string, *accountFile, int, error) {
	if !eip712.ValidTransferOperationID(id) {
		return "", nil, 0, fail(ExitInput, "use the full lowercase 0x-prefixed operation ID printed when you sent the transfer")
	}
	names, err := a.store.accounts()
	if err != nil {
		return "", nil, 0, err
	}
	for _, name := range names {
		ac, err := a.store.account(name)
		if err != nil {
			return "", nil, 0, err
		}
		for i, op := range ac.Operations {
			if op.ID == id {
				return name, ac, i, nil
			}
		}
	}
	return "", nil, 0, fail(ExitInput, "operation was not found in the local journal; run yellow operation list")
}
func decodeOperation(key []byte, op operation) (*client.SignedTransfer, error) {
	raw, err := openCipher(key, op.Payload, opAAD(op))
	if err != nil {
		return nil, err
	}
	defer clear(raw)
	var signed client.SignedTransfer
	if err := json.Unmarshal(raw, &signed); err != nil {
		return nil, fmt.Errorf("damaged encrypted authorization")
	}
	if signed.OperationID != op.ID || signed.Intent.Nonce != op.Nonce || signed.Intent.From != op.Sender || signed.Intent.NetworkID != op.Profile.NetworkID || signed.Intent.ChainID != op.Profile.Identity.ChainID {
		return nil, fmt.Errorf("authorization does not match its journal record")
	}
	return &signed, nil
}
func (a *app) completeOperations(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	if a.completions != nil {
		return a.completions.operations, cobra.ShellCompDirectiveNoFileComp
	}
	names, _ := a.store.accounts()
	var ids []string
	for _, name := range names {
		ac, err := a.store.account(name)
		if err != nil {
			continue
		}
		for _, op := range ac.Operations {
			ids = append(ids, op.ID+"\t"+op.State+" · "+op.Profile.Name)
		}
	}
	return ids, cobra.ShellCompDirectiveNoFileComp
}
func (a *app) operationCommands() *cobra.Command {
	parent := group("operation", "Inspect saved transfers and recover uncertain submissions")
	list := a.cmd("list", "View your persistent operation journal", cobra.NoArgs, func(context.Context, *cobra.Command, []string) error {
		names, err := a.store.accounts()
		if err != nil {
			return err
		}
		views := []operationView{}
		rows := [][]string{}
		for _, name := range names {
			ac, err := a.store.account(name)
			if err != nil {
				return err
			}
			for _, op := range ac.Operations {
				views = append(views, viewOperation(op))
				rows = append(rows, []string{op.State, op.Profile.Name, name, fmt.Sprint(op.Nonce), op.ID})
			}
		}
		return a.emit(views, func() {
			a.title("Saved operations")
			if len(rows) == 0 {
				fmt.Fprintln(a.out, "No signed transfers yet.")
			} else {
				a.table([]string{"STATE", "NETWORK", "ACCOUNT", "NONCE", "OPERATION"}, rows)
				fmt.Fprintln(a.out, "Inspect a transfer: yellow operation show <operation-id>")
			}
		})
	})
	show := a.cmd("show <operation-id>", "Check an operation against its original gateway", cobra.ExactArgs(1), func(ctx context.Context, _ *cobra.Command, args []string) error {
		name, ac, index, err := a.findOperation(args[0])
		if err != nil {
			return err
		}
		op := &ac.Operations[index]
		c, err := a.gateway(a.recoveryProfile(op.Profile), true)
		if err != nil {
			return err
		}
		status, err := c.Operation(ctx, op.ID)
		if err != nil {
			return fail(ExitNetwork, "could not query original gateway; saved state is %s for %s: %s", op.State, op.ID, err)
		}
		if err := observe(op, status); err != nil {
			return err
		}
		if err := a.store.saveAccount(name, ac); err != nil {
			return fail(ExitUnknown, "could not save status for operation %s: %s", op.ID, err)
		}
		return a.operationResult(*op)
	})
	retry := a.cmd("retry <operation-id>", "Explicitly replay the identical encrypted authorization", cobra.ExactArgs(1), func(ctx context.Context, _ *cobra.Command, args []string) error {
		name, ac, index, err := a.findOperation(args[0])
		if err != nil {
			return err
		}
		op := &ac.Operations[index]
		if op.State == "sealed" || op.State == "rejected" {
			return fail(ExitInput, "operation is already %s; inspect it with yellow operation show %s", op.State, op.ID)
		}
		c, err := a.gateway(a.recoveryProfile(op.Profile), true)
		if err != nil {
			return err
		}
		status, err := c.Operation(ctx, op.ID)
		if err != nil {
			return netError(err)
		}
		if err := observe(op, status); err != nil {
			return err
		}
		if err := a.store.saveAccount(name, ac); err != nil {
			return err
		}
		if op.State == "sealed" || op.State == "rejected" {
			return a.operationResult(*op)
		}
		neverSubmitted := op.Submitted == nil && op.State == "signed_unsubmitted" && status.Status == "unknown"
		ended := status.Status == "unknown" && status.PendingUntil > 0 && status.PendingUntil <= time.Now().Unix() && op.Submitted != nil && status.PendingUntil >= op.Submitted.Unix()
		if !neverSubmitted && !ended {
			return fail(ExitUnknown, "operation %s is unresolved or still pending; the original gateway has not established an ended pending window without a seal. No replay sent", op.ID)
		}
		signer, key, close, err := a.unlock(ac)
		if err != nil {
			return err
		}
		defer close()
		_ = signer
		signed, err := decodeOperation(key, *op)
		if err != nil {
			return err
		}
		if signed.Intent.From != ac.Address {
			return fmt.Errorf("saved authorization belongs to a different signer")
		}
		current := a.recoveryProfile(op.Profile)
		if current.Fingerprint != op.Profile.Fingerprint {
			fmt.Fprintln(a.diag, "\nCatalogue changed since signing. Saved base units and signature are unchanged; review their current meaning:")
			for _, change := range identityDiff(op.Profile.Identity, current.Identity) {
				fmt.Fprintln(a.diag, "  "+clean(change))
			}
		}
		known := map[string]bool{}
		for _, asset := range current.Identity.Assets {
			known[asset.Symbol] = true
		}
		for _, leg := range signed.Intent.Assets {
			if !known[leg.Asset] {
				return fail(ExitNetwork, "saved asset %s is absent from the current reviewed catalogue; no replay sent", leg.Asset)
			}
		}
		if !known["YELLOW"] {
			return fail(ExitNetwork, "current reviewed catalogue has no YELLOW fee asset; no replay sent")
		}
		a.reviewTransfer(current, name, &signed.Intent)
		fmt.Fprintf(a.diag, "  Operation  %s\nReplaying the original signed bytes; this transfer may already have executed.\n", op.ID)
		if err := a.prompt.confirm("Submit this exact saved authorization"); err != nil {
			return err
		}
		return a.submit(ctx, c, name, ac, index, signed, 0)
	})
	retry.Long = "Replay an authorization only after explicit review and unlock. A previously submitted operation requires its original gateway to report unknown with a past pendingUntil (Unix seconds). Older gateways without this evidence fail closed. The original nonce, recipient, assets, fee and signature are preserved."
	show.ValidArgsFunction = oneArgument(a.completeOperations)
	retry.ValidArgsFunction = oneArgument(a.completeOperations)
	parent.AddCommand(list, show, retry)
	return parent
}

// A reviewed catalogue update can unblock recovery at the same original
// gateway and signing domain, without changing the retained authorization.
func (a *app) recoveryProfile(original profile) profile {
	current, ok := a.cfg.Networks[original.Name]
	if ok && current.Gateway == original.Gateway && current.NetworkID == original.NetworkID && current.Identity.ChainID == original.Identity.ChainID && current.Identity.Router == original.Identity.Router {
		return current
	}
	return original
}

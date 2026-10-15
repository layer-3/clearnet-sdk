package yellow

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"

	"github.com/layer-3/clearnet-sdk/pkg/client"
	"github.com/spf13/cobra"
)

type identity struct {
	NetworkID string             `json:"networkId"`
	ChainID   uint64             `json:"chainId"`
	Router    string             `json:"router"`
	Assets    []client.AssetInfo `json:"assets"`
}

func reviewed(info *client.NetworkInfo) (identity, string, error) {
	id := identity{info.NetworkID, info.ChainID, strings.ToLower(info.VirtualRouter), append([]client.AssetInfo(nil), info.Assets...)}
	for i := range id.Assets {
		if !namePattern.MatchString(id.Assets[i].Symbol) {
			return id, "", fmt.Errorf("gateway catalogue has an invalid asset symbol")
		}
		id.Assets[i].Address = strings.ToLower(id.Assets[i].Address)
	}
	sort.Slice(id.Assets, func(i, j int) bool { return id.Assets[i].Symbol < id.Assets[j].Symbol })
	raw, err := json.Marshal(id)
	if err != nil {
		return id, "", err
	}
	sum := sha256.Sum256(raw)
	return id, hex.EncodeToString(sum[:]), nil
}

// Every SDK Info request is checked, including preparation/signing/submission
// preflights. Redirects are disabled so an HTTP 307 cannot replay a POST.
type trustTransport struct {
	base    http.RoundTripper
	profile profile
}

func (t trustTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	resp, err := t.base.RoundTrip(r)
	if err != nil {
		return nil, err
	}
	if r.Method != http.MethodGet || !strings.HasSuffix(r.URL.Path, "/v1/gateway/info") || resp.StatusCode != 200 {
		return resp, nil
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, (2<<20)+1))
	resp.Body.Close()
	if err != nil {
		return nil, err
	}
	if len(raw) > 2<<20 {
		return nil, fmt.Errorf("gateway metadata is too large")
	}
	var info client.NetworkInfo
	if err = json.Unmarshal(raw, &info); err != nil {
		return nil, fmt.Errorf("invalid gateway metadata")
	}
	_, fp, err := reviewed(&info)
	if err != nil {
		return nil, err
	}
	if fp != t.profile.Fingerprint {
		return nil, fmt.Errorf("gateway identity or asset catalogue changed; review yellow network trust-update %s before continuing", t.profile.Name)
	}
	resp.Body = io.NopCloser(bytes.NewReader(raw))
	return resp, nil
}
func (a *app) gateway(p profile, check bool) (*client.Client, error) {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.DisableKeepAlives = true
	var rt http.RoundTripper = transport
	if check {
		rt = trustTransport{transport, p}
	}
	h := &http.Client{Timeout: a.timeout, Transport: rt, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	c, err := client.New(p.Gateway, p.NetworkID, client.WithHTTPClient(h))
	if err != nil {
		return nil, fail(ExitNetwork, "%s", err)
	}
	return c, nil
}
func (a *app) selectedNetwork() (profile, error) {
	name := a.network
	if name == "" {
		name = a.cfg.Network
	}
	if name == "" {
		return profile{}, fail(ExitInput, "choose a network with --network, or run yellow network add <name> --gateway <url> --network-id <id>")
	}
	p, ok := a.cfg.Networks[name]
	if !ok {
		return p, fail(ExitInput, "network %q does not exist; run yellow network list", name)
	}
	return p, nil
}
func netError(err error) error {
	if err == nil {
		return nil
	}
	return fail(ExitNetwork, "%s", err)
}
func (a *app) profileReview(p profile) {
	fmt.Fprintf(a.diag, "\nReview network %s\n  Gateway      %s\n  Network ID   %s\n  EIP-712      %d\n  Router       %s\n  Fingerprint  %s\n", clean(p.Name), clean(p.Gateway), p.NetworkID, p.Identity.ChainID, p.Identity.Router, p.Fingerprint)
	for _, asset := range p.Identity.Assets {
		fmt.Fprintf(a.diag, "  %-12s %d decimals · chain %d · %s\n", asset.Symbol, asset.Decimals, asset.ChainID, clean(asset.Address))
	}
	fmt.Fprintln(a.diag, "Trust this HTTPS gateway's self-described signing domain and catalogue only after reviewing them.")
}
func identityDiff(old, new identity) []string {
	var changes []string
	if old.NetworkID != new.NetworkID || old.ChainID != new.ChainID || old.Router != new.Router {
		changes = append(changes, fmt.Sprintf("Signing identity: %s / %d / %s → %s / %d / %s", old.NetworkID, old.ChainID, old.Router, new.NetworkID, new.ChainID, new.Router))
	}
	before, after := map[string]client.AssetInfo{}, map[string]client.AssetInfo{}
	for _, v := range old.Assets {
		before[v.Symbol] = v
	}
	for _, v := range new.Assets {
		after[v.Symbol] = v
	}
	keys := map[string]bool{}
	for k := range before {
		keys[k] = true
	}
	for k := range after {
		keys[k] = true
	}
	var names []string
	for k := range keys {
		names = append(names, k)
	}
	sort.Strings(names)
	for _, k := range names {
		b, bok := before[k]
		n, nok := after[k]
		switch {
		case !bok:
			changes = append(changes, fmt.Sprintf("Added %s (%d decimals)", k, n.Decimals))
		case !nok:
			changes = append(changes, fmt.Sprintf("Removed %s (%d decimals)", k, b.Decimals))
		case b != n:
			changes = append(changes, fmt.Sprintf("Changed %s: decimals %d → %d; chain %d → %d; address %s → %s; native %t → %t", k, b.Decimals, n.Decimals, b.ChainID, n.ChainID, b.Address, n.Address, b.Native, n.Native))
		}
	}
	return changes
}

// trust accepts reviewed gateway metadata. A pinned fingerprint, obtained from
// the operator, replaces the interactive confirmation; a mismatch never saves.
func (a *app) trust(pinned, actual, label string) error {
	if pinned != "" {
		if strings.TrimPrefix(strings.ToLower(pinned), "0x") != actual {
			return fail(ExitNetwork, "gateway fingerprint is %s, not the expected %s; nothing was saved", actual, pinned)
		}
		fmt.Fprintln(a.diag, "Fingerprint matches the pinned value.")
		return nil
	}
	err := a.prompt.confirm(label)
	var f *failure
	if errors.As(err, &f) && f.code == ExitInput {
		return fail(ExitInput, "reviewing a gateway needs an interactive terminal, or --fingerprint %s once you have confirmed this fingerprint with the network operator", actual)
	}
	return err
}
func (a *app) networkCommands() *cobra.Command {
	parent := group("network", "Set up and review trusted gateway profiles")
	var gatewayURL, networkID, fingerprint string
	add := a.cmd("add <name>", "Review and save a gateway", cobra.ExactArgs(1), func(ctx context.Context, _ *cobra.Command, args []string) error {
		name := args[0]
		if err := validName(name); err != nil {
			return err
		}
		if _, exists := a.cfg.Networks[name]; exists {
			return fail(ExitInput, "network %q already exists", name)
		}
		p := profile{Name: name, Gateway: strings.TrimRight(gatewayURL, "/"), NetworkID: networkID}
		c, err := a.gateway(p, false)
		if err != nil {
			return err
		}
		info, err := c.Info(ctx)
		if err != nil {
			return netError(err)
		}
		p.Identity, p.Fingerprint, err = reviewed(info)
		if err != nil {
			return netError(err)
		}
		a.profileReview(p)
		if err := a.trust(fingerprint, p.Fingerprint, "Trust and save this network"); err != nil {
			return err
		}
		a.cfg.Networks[name] = p
		if a.cfg.Network == "" {
			a.cfg.Network = name
		}
		if err := a.store.saveConfig(a.cfg); err != nil {
			return err
		}
		return a.emit(p, func() {
			a.title("Network saved")
			fmt.Fprintf(a.out, "%s · %s\nNext: yellow account create personal\n", name, p.NetworkID)
		})
	})
	add.Flags().StringVar(&gatewayURL, "gateway", "", "HTTPS gateway URL; HTTP is allowed only for a loopback development profile")
	add.Flags().StringVar(&networkID, "network-id", "", "Expected four-byte Clearnet network ID (for example YTES)")
	add.Flags().StringVar(&fingerprint, "fingerprint", "", "Expected gateway fingerprint from the operator; accepts it without a prompt")
	_ = add.MarkFlagRequired("gateway")
	_ = add.MarkFlagRequired("network-id")
	add.Example = "  yellow network add testnet --gateway https://gateway.example.org --network-id YTES\n  yellow --json network add testnet --gateway https://gateway.example.org --network-id YTES --fingerprint <operator-fingerprint>"
	list := a.cmd("list", "View saved networks", cobra.NoArgs, func(context.Context, *cobra.Command, []string) error {
		var names []string
		for n := range a.cfg.Networks {
			names = append(names, n)
		}
		sort.Strings(names)
		profiles := []profile{}
		rows := [][]string{}
		for _, n := range names {
			p := a.cfg.Networks[n]
			profiles = append(profiles, p)
			mark := ""
			if n == a.cfg.Network {
				mark = "active"
			}
			rows = append(rows, []string{n, mark, p.NetworkID, p.Gateway})
		}
		return a.emit(profiles, func() {
			a.title("Your networks")
			if len(rows) == 0 {
				fmt.Fprintln(a.out, "No networks yet. Start with yellow network add --help.")
			} else {
				a.table([]string{"NAME", "", "NETWORK", "GATEWAY"}, rows)
			}
		})
	})
	show := a.cmd("show [name]", "View a network's reviewed identity and assets", cobra.MaximumNArgs(1), func(_ context.Context, _ *cobra.Command, args []string) error {
		p, err := a.namedProfile(args)
		if err != nil {
			return err
		}
		return a.emit(p, func() {
			a.title("Network · " + p.Name)
			fmt.Fprintf(a.out, "Gateway      %s\nNetwork ID   %s\nEIP-712      %d\nRouter       %s\nFingerprint  %s\n", clean(p.Gateway), p.NetworkID, p.Identity.ChainID, p.Identity.Router, p.Fingerprint)
			rows := [][]string{}
			for _, v := range p.Identity.Assets {
				rows = append(rows, []string{v.Symbol, fmt.Sprint(v.Decimals), fmt.Sprint(v.ChainID), v.Address})
			}
			a.table([]string{"ASSET", "DECIMALS", "CHAIN", "ADDRESS"}, rows)
		})
	})
	use := a.cmd("use <name>", "Select the default network", cobra.ExactArgs(1), func(_ context.Context, _ *cobra.Command, args []string) error {
		p, err := a.namedProfile(args)
		if err != nil {
			return err
		}
		a.cfg.Network = p.Name
		if err := a.store.saveConfig(a.cfg); err != nil {
			return err
		}
		return a.emit(map[string]string{"network": p.Name}, func() { fmt.Fprintln(a.out, "Active network:", p.Name) })
	})
	update := a.cmd("trust-update <name>", "Review catalogue differences before accepting them", cobra.ExactArgs(1), func(ctx context.Context, _ *cobra.Command, args []string) error {
		p, err := a.namedProfile(args)
		if err != nil {
			return err
		}
		c, err := a.gateway(p, false)
		if err != nil {
			return err
		}
		info, err := c.Info(ctx)
		if err != nil {
			return netError(err)
		}
		id, fp, err := reviewed(info)
		if err != nil {
			return netError(err)
		}
		changes := identityDiff(p.Identity, id)
		if len(changes) == 0 {
			if fingerprint != "" && strings.TrimPrefix(strings.ToLower(fingerprint), "0x") != fp {
				return fail(ExitNetwork, "gateway fingerprint is %s, not the expected %s", fp, fingerprint)
			}
			return a.emit(p, func() { fmt.Fprintln(a.out, "Trust profile is current:", p.Name) })
		}
		fmt.Fprintln(a.diag, "\nGateway changes:")
		for _, change := range changes {
			fmt.Fprintln(a.diag, "  "+clean(change))
		}
		p.Identity = id
		p.Fingerprint = fp
		a.profileReview(p)
		if err := a.trust(fingerprint, fp, "Accept these changes"); err != nil {
			return err
		}
		a.cfg.Networks[p.Name] = p
		if err := a.store.saveConfig(a.cfg); err != nil {
			return err
		}
		return a.emit(p, func() { fmt.Fprintln(a.out, "Trust updated:", p.Name) })
	})
	update.Flags().StringVar(&fingerprint, "fingerprint", "", "Expected new fingerprint from the operator; accepts it without a prompt")
	for _, c := range []*cobra.Command{show, use, update} {
		c.ValidArgsFunction = oneArgument(a.completeNetworks)
	}
	parent.AddCommand(add, list, show, use, update)
	return parent
}
func (a *app) namedProfile(args []string) (profile, error) {
	if len(args) == 0 {
		return a.selectedNetwork()
	}
	p, ok := a.cfg.Networks[args[0]]
	if !ok {
		return p, fail(ExitInput, "network %q does not exist; run yellow network list", args[0])
	}
	return p, nil
}

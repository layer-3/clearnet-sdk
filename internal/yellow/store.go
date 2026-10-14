package yellow

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/gofrs/flock"
	"github.com/layer-3/clearnet-sdk/pkg/client"
	"golang.org/x/crypto/scrypt"
)

const formatVersion = 1

var namePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,47}$`)

type profile struct {
	Name        string   `json:"name"`
	Gateway     string   `json:"gateway"`
	NetworkID   string   `json:"networkId"`
	Fingerprint string   `json:"fingerprint"`
	Identity    identity `json:"identity"`
}
type config struct {
	Version  int                `json:"version"`
	Network  string             `json:"network,omitempty"`
	Account  string             `json:"account,omitempty"`
	Networks map[string]profile `json:"networks"`
}
type ciphertext struct {
	Nonce []byte `json:"nonce"`
	Data  []byte `json:"data"`
}
type envelope struct {
	Version int    `json:"version"`
	KDF     string `json:"kdf"`
	Salt    []byte `json:"salt"`
	ciphertext
}
type secret struct {
	Kind  string `json:"kind"`
	Value string `json:"value"`
}
type operation struct {
	ID        string          `json:"operationId"`
	Account   string          `json:"account"`
	Sender    string          `json:"sender"`
	Nonce     uint64          `json:"nonce"`
	Profile   profile         `json:"profile"`
	Created   time.Time       `json:"createdAt"`
	Updated   time.Time       `json:"updatedAt"`
	Submitted *time.Time      `json:"submittedAt,omitempty"`
	State     string          `json:"state"`
	Receipt   *client.Receipt `json:"receipt,omitempty"`
	Payload   ciphertext      `json:"payload"`
}
type accountFile struct {
	Version    int         `json:"version"`
	Address    string      `json:"address"`
	Secret     envelope    `json:"secret"`
	Operations []operation `json:"operations"`
}
type store struct{ dir string }

func validName(name string) error {
	if !namePattern.MatchString(name) {
		return fmt.Errorf("use a name of 1–48 letters, digits, dashes or underscores, starting with a letter or digit")
	}
	return nil
}
func (s store) accountPath(name string) (string, error) {
	if err := validName(name); err != nil {
		return "", err
	}
	return filepath.Join(s.dir, "accounts", name+".json"), nil
}
func (s store) init() error {
	for _, dir := range []string{s.dir, filepath.Join(s.dir, "accounts")} {
		if err := os.MkdirAll(dir, 0700); err != nil {
			return err
		}
		st, err := os.Lstat(dir)
		if err != nil {
			return err
		}
		if !st.IsDir() || st.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("storage must be a real directory")
		}
		if err := os.Chmod(dir, 0700); err != nil {
			return err
		}
	}
	return nil
}
func (s store) lock(ctx context.Context) (func(), error) {
	if err := s.init(); err != nil {
		return nil, err
	}
	path := filepath.Join(s.dir, ".lock")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	if err = f.Chmod(0600); err != nil {
		f.Close()
		return nil, err
	}
	f.Close()
	l := flock.New(path)
	ok, err := l.TryLockContext(ctx, 50*time.Millisecond)
	if err != nil || !ok {
		return nil, fmt.Errorf("local storage is busy; another yellow command may be waiting for approval")
	}
	return func() { _ = l.Close() }, nil
}
func readJSON(path string, out any) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return err
	}
	if st.Size() > 16<<20 {
		return fmt.Errorf("local file exceeds 16 MiB")
	}
	return json.NewDecoder(f).Decode(out)
}
func (s store) config() (config, error) {
	c := config{Version: formatVersion, Networks: map[string]profile{}}
	err := readJSON(filepath.Join(s.dir, "config.json"), &c)
	if errors.Is(err, os.ErrNotExist) {
		return c, nil
	}
	if err != nil {
		return c, err
	}
	if c.Version != formatVersion || c.Networks == nil {
		return c, fmt.Errorf("unsupported configuration format")
	}
	return c, nil
}
func (s store) saveConfig(c config) error { return atomicJSON(filepath.Join(s.dir, "config.json"), c) }
func (s store) account(name string) (*accountFile, error) {
	path, err := s.accountPath(name)
	if err != nil {
		return nil, err
	}
	var a accountFile
	if err := readJSON(path, &a); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("account %q does not exist; run yellow account list", name)
		}
		return nil, err
	}
	if a.Version != formatVersion {
		return nil, fmt.Errorf("unsupported account format")
	}
	return &a, nil
}
func (s store) saveAccount(name string, a *accountFile) error {
	path, err := s.accountPath(name)
	if err != nil {
		return err
	}
	return atomicJSON(path, a)
}
func (s store) accounts() ([]string, error) {
	entries, err := os.ReadDir(filepath.Join(s.dir, "accounts"))
	if errors.Is(err, os.ErrNotExist) {
		return []string{}, nil
	}
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".json" {
			n := e.Name()[:len(e.Name())-5]
			if validName(n) == nil {
				names = append(names, n)
			}
		}
	}
	return names, nil
}
func atomicJSON(path string, value any) error {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	defer clear(raw)
	if len(raw) > 16<<20 {
		return fmt.Errorf("local journal exceeds 16 MiB; preserve it and reconcile operations before continuing")
	}
	return atomicWrite(path, raw)
}
func atomicWrite(path string, raw []byte) error {
	f, err := os.CreateTemp(filepath.Dir(path), ".yellow-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	defer f.Close()
	if err := f.Chmod(0600); err != nil {
		return err
	}
	if _, err := f.Write(raw); err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := replaceFile(f.Name(), path); err != nil {
		return err
	}
	return syncDir(filepath.Dir(path))
}
func deriveKey(pass []byte, salt []byte) ([]byte, error) {
	if len(salt) != 32 {
		return nil, fmt.Errorf("invalid encrypted file salt")
	}
	return scrypt.Key(pass, salt, 1<<17, 8, 1, 32)
}
func seal(key, raw []byte, aad string) (ciphertext, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return ciphertext{}, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return ciphertext{}, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return ciphertext{}, err
	}
	return ciphertext{Nonce: nonce, Data: gcm.Seal(nil, nonce, raw, []byte(aad))}, nil
}
func openCipher(key []byte, c ciphertext, aad string) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(c.Nonce) != gcm.NonceSize() {
		return nil, fmt.Errorf("invalid encrypted file nonce")
	}
	raw, err := gcm.Open(nil, c.Nonce, c.Data, []byte(aad))
	if err != nil {
		return nil, fmt.Errorf("wrong passphrase or damaged encrypted file")
	}
	return raw, nil
}
func encryptSecret(pass []byte, address string, sec secret) (envelope, []byte, error) {
	e := envelope{Version: 1, KDF: "scrypt-n131072-r8-p1-aes256gcm", Salt: make([]byte, 32)}
	if _, err := rand.Read(e.Salt); err != nil {
		return e, nil, err
	}
	key, err := deriveKey(pass, e.Salt)
	if err != nil {
		return e, nil, err
	}
	raw, err := json.Marshal(sec)
	if err != nil {
		clear(key)
		return e, nil, err
	}
	defer clear(raw)
	e.ciphertext, err = seal(key, raw, "yellow:secret:v1:"+address)
	if err != nil {
		clear(key)
		return e, nil, err
	}
	return e, key, nil
}
func decryptSecret(pass []byte, a *accountFile) (secret, []byte, error) {
	var sec secret
	if a.Secret.Version != 1 || a.Secret.KDF != "scrypt-n131072-r8-p1-aes256gcm" {
		return sec, nil, fmt.Errorf("unsupported encrypted account format")
	}
	key, err := deriveKey(pass, a.Secret.Salt)
	if err != nil {
		return sec, nil, err
	}
	raw, err := openCipher(key, a.Secret.ciphertext, "yellow:secret:v1:"+a.Address)
	if err != nil {
		clear(key)
		return sec, nil, err
	}
	defer clear(raw)
	if err = json.Unmarshal(raw, &sec); err != nil {
		clear(key)
		return sec, nil, fmt.Errorf("damaged encrypted account")
	}
	return sec, key, nil
}

// atomicWriteNew makes a fully synced backup visible without replacing a file.
func atomicWriteNew(path string, raw []byte) error {
	f, err := os.CreateTemp(filepath.Dir(path), ".yellow-backup-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	defer f.Close()
	if err := f.Chmod(0600); err != nil {
		return err
	}
	if _, err := f.Write(raw); err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Link(f.Name(), path); err != nil {
		return err
	}
	return syncDir(filepath.Dir(path))
}

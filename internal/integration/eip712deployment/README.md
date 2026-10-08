# EIP-712 deployment test

This test deploys the committed EVM bytecode into go-ethereum's in-process
simulated chain and exercises the SDK's signing helpers against it. It does
not require Docker or a running devnet.

The separate Go module keeps the simulated node's database and server
dependencies out of the root SDK module. Its local `replace` directive ensures
the test uses this checkout's SDK code.

Run it from the repository root:

```sh
cd internal/integration/eip712deployment
go test -tags integration -v ./...
```

`make integration` also runs this test. Root commands such as `go test ./...`
and `go vet ./...` do not traverse the nested module.

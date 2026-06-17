// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package evm

import (
	"errors"
	"math/big"
	"strings"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = errors.New
	_ = big.NewInt
	_ = strings.NewReader
	_ = ethereum.NotFound
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
	_ = abi.ConvertType
)

// MockERC20MetaData contains all meta data concerning the MockERC20 contract.
var MockERC20MetaData = &bind.MetaData{
	ABI: "[{\"type\":\"constructor\",\"inputs\":[{\"name\":\"_name\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"_symbol\",\"type\":\"string\",\"internalType\":\"string\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"allowance\",\"inputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"approve\",\"inputs\":[{\"name\":\"spender\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"value\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"balanceOf\",\"inputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"decimals\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint8\",\"internalType\":\"uint8\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"mint\",\"inputs\":[{\"name\":\"to\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"amount\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"name\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"string\",\"internalType\":\"string\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"symbol\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"string\",\"internalType\":\"string\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"totalSupply\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"transfer\",\"inputs\":[{\"name\":\"to\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"value\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"transferFrom\",\"inputs\":[{\"name\":\"from\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"to\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"value\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"event\",\"name\":\"Approval\",\"inputs\":[{\"name\":\"owner\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"spender\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"value\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"Transfer\",\"inputs\":[{\"name\":\"from\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"to\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"value\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false}]",
	Bin: "0x60806040523461032e576109b48038038061001981610332565b92833981019060408183031261032e5780516001600160401b03811161032e5782610045918301610357565b60208201519092906001600160401b03811161032e576100659201610357565b6002805460ff1916601217905581516001600160401b038111610239575f54600181811c91168015610324575b602082101461021b57601f81116102b7575b50602092601f821160011461025857928192935f9261024d575b50508160011b915f199060031b1c1916175f555b80516001600160401b03811161023957600154600181811c9116801561022f575b602082101461021b57601f81116101ad575b50602091601f821160011461014d579181925f92610142575b50508160011b915f199060031b1c1916176001555b60405161060b90816103a98239f35b015190505f8061011e565b601f1982169260015f52805f20915f5b8581106101955750836001951061017d575b505050811b01600155610133565b01515f1960f88460031b161c191690555f808061016f565b9192602060018192868501518155019401920161015d565b818111156101055760015f52601f820160051c7fb10e2d527612073b26eecdfd717e6a320cf44b4afac2b0732d9fcbe2b7fa0cf660208410610213575b81601f9101920160051c03905f5b828110610206575050610105565b5f828201556001016101f8565b5f91506101ea565b634e487b7160e01b5f52602260045260245ffd5b90607f16906100f3565b634e487b7160e01b5f52604160045260245ffd5b015190505f806100be565b601f198216935f8052805f20915f5b86811061029f5750836001959610610287575b505050811b015f556100d2565b01515f1960f88460031b161c191690555f808061027a565b91926020600181928685015181550194019201610267565b818111156100a4575f8052601f820160051c7f290decd9548b62a8d60345a988386fc84ba6bc95484008f6362f93160ef3e5636020841061031c575b81601f9101920160051c03905f5b82811061030f5750506100a4565b5f82820155600101610301565b5f91506102f3565b90607f1690610092565b5f80fd5b6040519190601f01601f191682016001600160401b0381118382101761023957604052565b81601f8201121561032e578051906001600160401b03821161023957610386601f8301601f1916602001610332565b928284526020838301011161032e57815f9260208093018386015e830101529056fe60806040526004361015610011575f80fd5b5f3560e01c806306fdde03146104b5578063095ea7b31461043c57806318160ddd1461041f57806323b872dd1461035a578063313ce5671461033a57806340c10f19146102c057806370a082311461028857806395d89b411461016a578063a9059cbb146100db5763dd62ed3e14610087575f80fd5b346100d75760403660031901126100d7576100a06105b1565b6100a86105c7565b6001600160a01b039182165f908152600560209081526040808320949093168252928352819020549051908152f35b5f80fd5b346100d75760403660031901126100d7576100f46105b1565b60243590335f52600460205260405f2061010f8382546105dd565b905560018060a01b031690815f52600460205260405f206101318282546105fe565b90556040519081527fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef60203392a3602060405160018152f35b346100d7575f3660031901126100d7576040515f6001548060011c9060018116801561027e575b60208310811461026a5782855290811561024e57506001146101f8575b50819003601f01601f191681019067ffffffffffffffff8211818310176101e457604082905281906101e09082610587565b0390f35b634e487b7160e01b5f52604160045260245ffd5b60015f9081529091507fb10e2d527612073b26eecdfd717e6a320cf44b4afac2b0732d9fcbe2b7fa0cf65b828210610238575060209150820101826101ae565b6001816020925483858801015201910190610223565b90506020925060ff191682840152151560051b820101826101ae565b634e487b7160e01b5f52602260045260245ffd5b91607f1691610191565b346100d75760203660031901126100d7576001600160a01b036102a96105b1565b165f526004602052602060405f2054604051908152f35b346100d75760403660031901126100d7576102d96105b1565b5f7fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef60206024359360018060a01b03169384845260048252604084206103208282546105fe565b905561032e816003546105fe565b600355604051908152a3005b346100d7575f3660031901126100d757602060ff60025416604051908152f35b346100d75760603660031901126100d7576103736105b1565b61037b6105c7565b7fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef60206044359360018060a01b031692835f526005825260405f2060018060a01b0333165f52825260405f206103d28682546105dd565b9055835f526004825260405f206103ea8682546105dd565b905560018060a01b031693845f526004825260405f2061040b8282546105fe565b9055604051908152a3602060405160018152f35b346100d7575f3660031901126100d7576020600354604051908152f35b346100d75760403660031901126100d7576104556105b1565b335f8181526005602090815260408083206001600160a01b03909516808452948252918290206024359081905591519182527f8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b92591a3602060405160018152f35b346100d7575f3660031901126100d7576040515f5f548060011c9060018116801561057d575b60208310811461026a5782855290811561024e57506001146105295750819003601f01601f191681019067ffffffffffffffff8211818310176101e457604082905281906101e09082610587565b5f8080529091507f290decd9548b62a8d60345a988386fc84ba6bc95484008f6362f93160ef3e5635b828210610567575060209150820101826101ae565b6001816020925483858801015201910190610552565b91607f16916104db565b602060409281835280519182918282860152018484015e5f828201840152601f01601f1916010190565b600435906001600160a01b03821682036100d757565b602435906001600160a01b03821682036100d757565b919082039182116105ea57565b634e487b7160e01b5f52601160045260245ffd5b919082018092116105ea5756",
}

// MockERC20ABI is the input ABI used to generate the binding from.
// Deprecated: Use MockERC20MetaData.ABI instead.
var MockERC20ABI = MockERC20MetaData.ABI

// MockERC20Bin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use MockERC20MetaData.Bin instead.
var MockERC20Bin = MockERC20MetaData.Bin

// DeployMockERC20 deploys a new Ethereum contract, binding an instance of MockERC20 to it.
func DeployMockERC20(auth *bind.TransactOpts, backend bind.ContractBackend, _name string, _symbol string) (common.Address, *types.Transaction, *MockERC20, error) {
	parsed, err := MockERC20MetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(MockERC20Bin), backend, _name, _symbol)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &MockERC20{MockERC20Caller: MockERC20Caller{contract: contract}, MockERC20Transactor: MockERC20Transactor{contract: contract}, MockERC20Filterer: MockERC20Filterer{contract: contract}}, nil
}

// MockERC20 is an auto generated Go binding around an Ethereum contract.
type MockERC20 struct {
	MockERC20Caller     // Read-only binding to the contract
	MockERC20Transactor // Write-only binding to the contract
	MockERC20Filterer   // Log filterer for contract events
}

// MockERC20Caller is an auto generated read-only Go binding around an Ethereum contract.
type MockERC20Caller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MockERC20Transactor is an auto generated write-only Go binding around an Ethereum contract.
type MockERC20Transactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MockERC20Filterer is an auto generated log filtering Go binding around an Ethereum contract events.
type MockERC20Filterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MockERC20Session is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type MockERC20Session struct {
	Contract     *MockERC20        // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// MockERC20CallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type MockERC20CallerSession struct {
	Contract *MockERC20Caller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts    // Call options to use throughout this session
}

// MockERC20TransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type MockERC20TransactorSession struct {
	Contract     *MockERC20Transactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts    // Transaction auth options to use throughout this session
}

// MockERC20Raw is an auto generated low-level Go binding around an Ethereum contract.
type MockERC20Raw struct {
	Contract *MockERC20 // Generic contract binding to access the raw methods on
}

// MockERC20CallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type MockERC20CallerRaw struct {
	Contract *MockERC20Caller // Generic read-only contract binding to access the raw methods on
}

// MockERC20TransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type MockERC20TransactorRaw struct {
	Contract *MockERC20Transactor // Generic write-only contract binding to access the raw methods on
}

// NewMockERC20 creates a new instance of MockERC20, bound to a specific deployed contract.
func NewMockERC20(address common.Address, backend bind.ContractBackend) (*MockERC20, error) {
	contract, err := bindMockERC20(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &MockERC20{MockERC20Caller: MockERC20Caller{contract: contract}, MockERC20Transactor: MockERC20Transactor{contract: contract}, MockERC20Filterer: MockERC20Filterer{contract: contract}}, nil
}

// NewMockERC20Caller creates a new read-only instance of MockERC20, bound to a specific deployed contract.
func NewMockERC20Caller(address common.Address, caller bind.ContractCaller) (*MockERC20Caller, error) {
	contract, err := bindMockERC20(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &MockERC20Caller{contract: contract}, nil
}

// NewMockERC20Transactor creates a new write-only instance of MockERC20, bound to a specific deployed contract.
func NewMockERC20Transactor(address common.Address, transactor bind.ContractTransactor) (*MockERC20Transactor, error) {
	contract, err := bindMockERC20(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &MockERC20Transactor{contract: contract}, nil
}

// NewMockERC20Filterer creates a new log filterer instance of MockERC20, bound to a specific deployed contract.
func NewMockERC20Filterer(address common.Address, filterer bind.ContractFilterer) (*MockERC20Filterer, error) {
	contract, err := bindMockERC20(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &MockERC20Filterer{contract: contract}, nil
}

// bindMockERC20 binds a generic wrapper to an already deployed contract.
func bindMockERC20(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := MockERC20MetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_MockERC20 *MockERC20Raw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _MockERC20.Contract.MockERC20Caller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_MockERC20 *MockERC20Raw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _MockERC20.Contract.MockERC20Transactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_MockERC20 *MockERC20Raw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _MockERC20.Contract.MockERC20Transactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_MockERC20 *MockERC20CallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _MockERC20.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_MockERC20 *MockERC20TransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _MockERC20.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_MockERC20 *MockERC20TransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _MockERC20.Contract.contract.Transact(opts, method, params...)
}

// Allowance is a free data retrieval call binding the contract method 0xdd62ed3e.
//
// Solidity: function allowance(address , address ) view returns(uint256)
func (_MockERC20 *MockERC20Caller) Allowance(opts *bind.CallOpts, arg0 common.Address, arg1 common.Address) (*big.Int, error) {
	var out []interface{}
	err := _MockERC20.contract.Call(opts, &out, "allowance", arg0, arg1)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Allowance is a free data retrieval call binding the contract method 0xdd62ed3e.
//
// Solidity: function allowance(address , address ) view returns(uint256)
func (_MockERC20 *MockERC20Session) Allowance(arg0 common.Address, arg1 common.Address) (*big.Int, error) {
	return _MockERC20.Contract.Allowance(&_MockERC20.CallOpts, arg0, arg1)
}

// Allowance is a free data retrieval call binding the contract method 0xdd62ed3e.
//
// Solidity: function allowance(address , address ) view returns(uint256)
func (_MockERC20 *MockERC20CallerSession) Allowance(arg0 common.Address, arg1 common.Address) (*big.Int, error) {
	return _MockERC20.Contract.Allowance(&_MockERC20.CallOpts, arg0, arg1)
}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address ) view returns(uint256)
func (_MockERC20 *MockERC20Caller) BalanceOf(opts *bind.CallOpts, arg0 common.Address) (*big.Int, error) {
	var out []interface{}
	err := _MockERC20.contract.Call(opts, &out, "balanceOf", arg0)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address ) view returns(uint256)
func (_MockERC20 *MockERC20Session) BalanceOf(arg0 common.Address) (*big.Int, error) {
	return _MockERC20.Contract.BalanceOf(&_MockERC20.CallOpts, arg0)
}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address ) view returns(uint256)
func (_MockERC20 *MockERC20CallerSession) BalanceOf(arg0 common.Address) (*big.Int, error) {
	return _MockERC20.Contract.BalanceOf(&_MockERC20.CallOpts, arg0)
}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() view returns(uint8)
func (_MockERC20 *MockERC20Caller) Decimals(opts *bind.CallOpts) (uint8, error) {
	var out []interface{}
	err := _MockERC20.contract.Call(opts, &out, "decimals")

	if err != nil {
		return *new(uint8), err
	}

	out0 := *abi.ConvertType(out[0], new(uint8)).(*uint8)

	return out0, err

}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() view returns(uint8)
func (_MockERC20 *MockERC20Session) Decimals() (uint8, error) {
	return _MockERC20.Contract.Decimals(&_MockERC20.CallOpts)
}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() view returns(uint8)
func (_MockERC20 *MockERC20CallerSession) Decimals() (uint8, error) {
	return _MockERC20.Contract.Decimals(&_MockERC20.CallOpts)
}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_MockERC20 *MockERC20Caller) Name(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _MockERC20.contract.Call(opts, &out, "name")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_MockERC20 *MockERC20Session) Name() (string, error) {
	return _MockERC20.Contract.Name(&_MockERC20.CallOpts)
}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_MockERC20 *MockERC20CallerSession) Name() (string, error) {
	return _MockERC20.Contract.Name(&_MockERC20.CallOpts)
}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() view returns(string)
func (_MockERC20 *MockERC20Caller) Symbol(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _MockERC20.contract.Call(opts, &out, "symbol")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() view returns(string)
func (_MockERC20 *MockERC20Session) Symbol() (string, error) {
	return _MockERC20.Contract.Symbol(&_MockERC20.CallOpts)
}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() view returns(string)
func (_MockERC20 *MockERC20CallerSession) Symbol() (string, error) {
	return _MockERC20.Contract.Symbol(&_MockERC20.CallOpts)
}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() view returns(uint256)
func (_MockERC20 *MockERC20Caller) TotalSupply(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _MockERC20.contract.Call(opts, &out, "totalSupply")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() view returns(uint256)
func (_MockERC20 *MockERC20Session) TotalSupply() (*big.Int, error) {
	return _MockERC20.Contract.TotalSupply(&_MockERC20.CallOpts)
}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() view returns(uint256)
func (_MockERC20 *MockERC20CallerSession) TotalSupply() (*big.Int, error) {
	return _MockERC20.Contract.TotalSupply(&_MockERC20.CallOpts)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address spender, uint256 value) returns(bool)
func (_MockERC20 *MockERC20Transactor) Approve(opts *bind.TransactOpts, spender common.Address, value *big.Int) (*types.Transaction, error) {
	return _MockERC20.contract.Transact(opts, "approve", spender, value)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address spender, uint256 value) returns(bool)
func (_MockERC20 *MockERC20Session) Approve(spender common.Address, value *big.Int) (*types.Transaction, error) {
	return _MockERC20.Contract.Approve(&_MockERC20.TransactOpts, spender, value)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address spender, uint256 value) returns(bool)
func (_MockERC20 *MockERC20TransactorSession) Approve(spender common.Address, value *big.Int) (*types.Transaction, error) {
	return _MockERC20.Contract.Approve(&_MockERC20.TransactOpts, spender, value)
}

// Mint is a paid mutator transaction binding the contract method 0x40c10f19.
//
// Solidity: function mint(address to, uint256 amount) returns()
func (_MockERC20 *MockERC20Transactor) Mint(opts *bind.TransactOpts, to common.Address, amount *big.Int) (*types.Transaction, error) {
	return _MockERC20.contract.Transact(opts, "mint", to, amount)
}

// Mint is a paid mutator transaction binding the contract method 0x40c10f19.
//
// Solidity: function mint(address to, uint256 amount) returns()
func (_MockERC20 *MockERC20Session) Mint(to common.Address, amount *big.Int) (*types.Transaction, error) {
	return _MockERC20.Contract.Mint(&_MockERC20.TransactOpts, to, amount)
}

// Mint is a paid mutator transaction binding the contract method 0x40c10f19.
//
// Solidity: function mint(address to, uint256 amount) returns()
func (_MockERC20 *MockERC20TransactorSession) Mint(to common.Address, amount *big.Int) (*types.Transaction, error) {
	return _MockERC20.Contract.Mint(&_MockERC20.TransactOpts, to, amount)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(address to, uint256 value) returns(bool)
func (_MockERC20 *MockERC20Transactor) Transfer(opts *bind.TransactOpts, to common.Address, value *big.Int) (*types.Transaction, error) {
	return _MockERC20.contract.Transact(opts, "transfer", to, value)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(address to, uint256 value) returns(bool)
func (_MockERC20 *MockERC20Session) Transfer(to common.Address, value *big.Int) (*types.Transaction, error) {
	return _MockERC20.Contract.Transfer(&_MockERC20.TransactOpts, to, value)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(address to, uint256 value) returns(bool)
func (_MockERC20 *MockERC20TransactorSession) Transfer(to common.Address, value *big.Int) (*types.Transaction, error) {
	return _MockERC20.Contract.Transfer(&_MockERC20.TransactOpts, to, value)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address from, address to, uint256 value) returns(bool)
func (_MockERC20 *MockERC20Transactor) TransferFrom(opts *bind.TransactOpts, from common.Address, to common.Address, value *big.Int) (*types.Transaction, error) {
	return _MockERC20.contract.Transact(opts, "transferFrom", from, to, value)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address from, address to, uint256 value) returns(bool)
func (_MockERC20 *MockERC20Session) TransferFrom(from common.Address, to common.Address, value *big.Int) (*types.Transaction, error) {
	return _MockERC20.Contract.TransferFrom(&_MockERC20.TransactOpts, from, to, value)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address from, address to, uint256 value) returns(bool)
func (_MockERC20 *MockERC20TransactorSession) TransferFrom(from common.Address, to common.Address, value *big.Int) (*types.Transaction, error) {
	return _MockERC20.Contract.TransferFrom(&_MockERC20.TransactOpts, from, to, value)
}

// MockERC20ApprovalIterator is returned from FilterApproval and is used to iterate over the raw logs and unpacked data for Approval events raised by the MockERC20 contract.
type MockERC20ApprovalIterator struct {
	Event *MockERC20Approval // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *MockERC20ApprovalIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(MockERC20Approval)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(MockERC20Approval)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *MockERC20ApprovalIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *MockERC20ApprovalIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// MockERC20Approval represents a Approval event raised by the MockERC20 contract.
type MockERC20Approval struct {
	Owner   common.Address
	Spender common.Address
	Value   *big.Int
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterApproval is a free log retrieval operation binding the contract event 0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925.
//
// Solidity: event Approval(address indexed owner, address indexed spender, uint256 value)
func (_MockERC20 *MockERC20Filterer) FilterApproval(opts *bind.FilterOpts, owner []common.Address, spender []common.Address) (*MockERC20ApprovalIterator, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var spenderRule []interface{}
	for _, spenderItem := range spender {
		spenderRule = append(spenderRule, spenderItem)
	}

	logs, sub, err := _MockERC20.contract.FilterLogs(opts, "Approval", ownerRule, spenderRule)
	if err != nil {
		return nil, err
	}
	return &MockERC20ApprovalIterator{contract: _MockERC20.contract, event: "Approval", logs: logs, sub: sub}, nil
}

// WatchApproval is a free log subscription operation binding the contract event 0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925.
//
// Solidity: event Approval(address indexed owner, address indexed spender, uint256 value)
func (_MockERC20 *MockERC20Filterer) WatchApproval(opts *bind.WatchOpts, sink chan<- *MockERC20Approval, owner []common.Address, spender []common.Address) (event.Subscription, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var spenderRule []interface{}
	for _, spenderItem := range spender {
		spenderRule = append(spenderRule, spenderItem)
	}

	logs, sub, err := _MockERC20.contract.WatchLogs(opts, "Approval", ownerRule, spenderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(MockERC20Approval)
				if err := _MockERC20.contract.UnpackLog(event, "Approval", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseApproval is a log parse operation binding the contract event 0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925.
//
// Solidity: event Approval(address indexed owner, address indexed spender, uint256 value)
func (_MockERC20 *MockERC20Filterer) ParseApproval(log types.Log) (*MockERC20Approval, error) {
	event := new(MockERC20Approval)
	if err := _MockERC20.contract.UnpackLog(event, "Approval", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// MockERC20TransferIterator is returned from FilterTransfer and is used to iterate over the raw logs and unpacked data for Transfer events raised by the MockERC20 contract.
type MockERC20TransferIterator struct {
	Event *MockERC20Transfer // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *MockERC20TransferIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(MockERC20Transfer)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(MockERC20Transfer)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *MockERC20TransferIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *MockERC20TransferIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// MockERC20Transfer represents a Transfer event raised by the MockERC20 contract.
type MockERC20Transfer struct {
	From  common.Address
	To    common.Address
	Value *big.Int
	Raw   types.Log // Blockchain specific contextual infos
}

// FilterTransfer is a free log retrieval operation binding the contract event 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef.
//
// Solidity: event Transfer(address indexed from, address indexed to, uint256 value)
func (_MockERC20 *MockERC20Filterer) FilterTransfer(opts *bind.FilterOpts, from []common.Address, to []common.Address) (*MockERC20TransferIterator, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _MockERC20.contract.FilterLogs(opts, "Transfer", fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return &MockERC20TransferIterator{contract: _MockERC20.contract, event: "Transfer", logs: logs, sub: sub}, nil
}

// WatchTransfer is a free log subscription operation binding the contract event 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef.
//
// Solidity: event Transfer(address indexed from, address indexed to, uint256 value)
func (_MockERC20 *MockERC20Filterer) WatchTransfer(opts *bind.WatchOpts, sink chan<- *MockERC20Transfer, from []common.Address, to []common.Address) (event.Subscription, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _MockERC20.contract.WatchLogs(opts, "Transfer", fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(MockERC20Transfer)
				if err := _MockERC20.contract.UnpackLog(event, "Transfer", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseTransfer is a log parse operation binding the contract event 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef.
//
// Solidity: event Transfer(address indexed from, address indexed to, uint256 value)
func (_MockERC20 *MockERC20Filterer) ParseTransfer(log types.Log) (*MockERC20Transfer, error) {
	event := new(MockERC20Transfer)
	if err := _MockERC20.contract.UnpackLog(event, "Transfer", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

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

// FaucetMetaData contains all meta data concerning the Faucet contract.
var FaucetMetaData = &bind.MetaData{
	ABI: "[{\"type\":\"constructor\",\"inputs\":[{\"name\":\"_token\",\"type\":\"address\",\"internalType\":\"contractIERC20\"},{\"name\":\"_dripAmount\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"_cooldown\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"TOKEN\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"contractIERC20\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"cooldown\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"drip\",\"inputs\":[],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"dripAmount\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"dripTo\",\"inputs\":[{\"name\":\"recipient\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"lastDrip\",\"inputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"owner\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"setCooldown\",\"inputs\":[{\"name\":\"_cooldown\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"setDripAmount\",\"inputs\":[{\"name\":\"_dripAmount\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"setOwner\",\"inputs\":[{\"name\":\"_owner\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"withdraw\",\"inputs\":[{\"name\":\"amount\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"event\",\"name\":\"CooldownUpdated\",\"inputs\":[{\"name\":\"newCooldown\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"DripAmountUpdated\",\"inputs\":[{\"name\":\"newAmount\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"Dripped\",\"inputs\":[{\"name\":\"recipient\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"amount\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"OwnerUpdated\",\"inputs\":[{\"name\":\"newOwner\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"}],\"anonymous\":false}]",
	Bin: "0x60a03461009a57601f6107bb38819003918201601f19168301916001600160401b0383118484101761009e5780849260609460405283398101031261009a578051906001600160a01b038216820361009a5760406020820151910151916080523360018060a01b03195f5416175f5560015560025560405161070890816100b3823960805181818161011d0152818161026d01526104e30152f35b5f80fd5b634e487b7160e01b5f52604160045260245ffdfe6080806040526004361015610012575f80fd5b5f3560e01c9081630935f004146103d35750806313af40351461032c5780632e1a7d4d1461021e57806335a1529b146102015780634fc3f41a146101b5578063543f8c5814610169578063787a08a61461014c57806382bfefc8146101085780638da5cb5b146100e15780639f678cca146100c85763cabee26e14610095575f80fd5b346100c45760203660031901126100c4576004356001600160a01b03811681036100c4576100c2906104a6565b005b5f80fd5b346100c4575f3660031901126100c4576100c2336104a6565b346100c4575f3660031901126100c4575f546040516001600160a01b039091168152602090f35b346100c4575f3660031901126100c4576040517f00000000000000000000000000000000000000000000000000000000000000006001600160a01b03168152602090f35b346100c4575f3660031901126100c4576020600254604051908152f35b346100c45760203660031901126100c4577f33f3faee0788ab897d8f674abe1dde6d93ba901e4a1502161294734ba178e3c760206004356101a861045a565b80600155604051908152a1005b346100c45760203660031901126100c4577f583d8b24c5439ab7d810e51e37e8db41ba66f1168fd7b752ceae0c7681c5272c60206004356101f461045a565b80600255604051908152a1005b346100c4575f3660031901126100c4576020600154604051908152f35b346100c45760203660031901126100c45761023761045a565b5f5460405163a9059cbb60e01b81526001600160a01b03909116600480830191909152356024820152602081806044810103815f7f00000000000000000000000000000000000000000000000000000000000000006001600160a01b03165af1908115610321575f916102f2575b50156102ad57005b60405162461bcd60e51b815260206004820152601760248201527f4661756365743a207769746864726177206661696c65640000000000000000006044820152606490fd5b610314915060203d60201161031a575b61030c818361040c565b810190610442565b816102a5565b503d610302565b6040513d5f823e3d90fd5b346100c45760203660031901126100c4576004356001600160a01b038116908190036100c45761035a61045a565b8015610397575f80546001600160a01b031916821781557f4ffd725fc4a22075e9ec71c59edf9c38cdeb588a91b24fc5b61388c5be41282b9080a2005b60405162461bcd60e51b81526020600482015260146024820152734661756365743a207a65726f206164647265737360601b6044820152606490fd5b346100c45760203660031901126100c4576004356001600160a01b03811691908290036100c4576020915f526003825260405f20548152f35b90601f8019910116810190811067ffffffffffffffff82111761042e57604052565b634e487b7160e01b5f52604160045260245ffd5b908160209103126100c4575180151581036100c45790565b5f546001600160a01b0316330361046d57565b60405162461bcd60e51b81526020600482015260116024820152702330bab1b2ba1d103737ba1037bbb732b960791b6044820152606490fd5b6001600160a01b0381165f818152600360205260409020549091901580156106d2575b1561068d576040516370a0823160e01b81523060048201527f00000000000000000000000000000000000000000000000000000000000000006001600160a01b031690602081602481855afa908115610321575f9161065b575b5060015411610616575f838152600360209081526040808320429055600154905163a9059cbb60e01b81526001600160a01b0395909516600486015260248501529183916044918391905af1908115610321575f916105f7575b50156105b2577f0daf449977d5acafa35195e10b3eb92f97839892a6653afaba222379b58d8a9b6020600154604051908152a2565b60405162461bcd60e51b815260206004820152601760248201527f4661756365743a207472616e73666572206661696c65640000000000000000006044820152606490fd5b610610915060203d60201161031a5761030c818361040c565b5f61057d565b60405162461bcd60e51b815260206004820152601c60248201527f4661756365743a20696e73756666696369656e742062616c616e6365000000006044820152606490fd5b90506020813d602011610685575b816106766020938361040c565b810103126100c457515f610523565b3d9150610669565b60405162461bcd60e51b815260206004820152601760248201527f4661756365743a20636f6f6c646f776e206163746976650000000000000000006044820152606490fd5b50815f52600360205260405f205460025481018091116106f4574210156104c9565b634e487b7160e01b5f52601160045260245ffd",
}

// FaucetABI is the input ABI used to generate the binding from.
// Deprecated: Use FaucetMetaData.ABI instead.
var FaucetABI = FaucetMetaData.ABI

// FaucetBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use FaucetMetaData.Bin instead.
var FaucetBin = FaucetMetaData.Bin

// DeployFaucet deploys a new Ethereum contract, binding an instance of Faucet to it.
func DeployFaucet(auth *bind.TransactOpts, backend bind.ContractBackend, _token common.Address, _dripAmount *big.Int, _cooldown *big.Int) (common.Address, *types.Transaction, *Faucet, error) {
	parsed, err := FaucetMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(FaucetBin), backend, _token, _dripAmount, _cooldown)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &Faucet{FaucetCaller: FaucetCaller{contract: contract}, FaucetTransactor: FaucetTransactor{contract: contract}, FaucetFilterer: FaucetFilterer{contract: contract}}, nil
}

// Faucet is an auto generated Go binding around an Ethereum contract.
type Faucet struct {
	FaucetCaller     // Read-only binding to the contract
	FaucetTransactor // Write-only binding to the contract
	FaucetFilterer   // Log filterer for contract events
}

// FaucetCaller is an auto generated read-only Go binding around an Ethereum contract.
type FaucetCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// FaucetTransactor is an auto generated write-only Go binding around an Ethereum contract.
type FaucetTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// FaucetFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type FaucetFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// FaucetSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type FaucetSession struct {
	Contract     *Faucet           // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// FaucetCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type FaucetCallerSession struct {
	Contract *FaucetCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts // Call options to use throughout this session
}

// FaucetTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type FaucetTransactorSession struct {
	Contract     *FaucetTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// FaucetRaw is an auto generated low-level Go binding around an Ethereum contract.
type FaucetRaw struct {
	Contract *Faucet // Generic contract binding to access the raw methods on
}

// FaucetCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type FaucetCallerRaw struct {
	Contract *FaucetCaller // Generic read-only contract binding to access the raw methods on
}

// FaucetTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type FaucetTransactorRaw struct {
	Contract *FaucetTransactor // Generic write-only contract binding to access the raw methods on
}

// NewFaucet creates a new instance of Faucet, bound to a specific deployed contract.
func NewFaucet(address common.Address, backend bind.ContractBackend) (*Faucet, error) {
	contract, err := bindFaucet(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Faucet{FaucetCaller: FaucetCaller{contract: contract}, FaucetTransactor: FaucetTransactor{contract: contract}, FaucetFilterer: FaucetFilterer{contract: contract}}, nil
}

// NewFaucetCaller creates a new read-only instance of Faucet, bound to a specific deployed contract.
func NewFaucetCaller(address common.Address, caller bind.ContractCaller) (*FaucetCaller, error) {
	contract, err := bindFaucet(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &FaucetCaller{contract: contract}, nil
}

// NewFaucetTransactor creates a new write-only instance of Faucet, bound to a specific deployed contract.
func NewFaucetTransactor(address common.Address, transactor bind.ContractTransactor) (*FaucetTransactor, error) {
	contract, err := bindFaucet(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &FaucetTransactor{contract: contract}, nil
}

// NewFaucetFilterer creates a new log filterer instance of Faucet, bound to a specific deployed contract.
func NewFaucetFilterer(address common.Address, filterer bind.ContractFilterer) (*FaucetFilterer, error) {
	contract, err := bindFaucet(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &FaucetFilterer{contract: contract}, nil
}

// bindFaucet binds a generic wrapper to an already deployed contract.
func bindFaucet(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := FaucetMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Faucet *FaucetRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Faucet.Contract.FaucetCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Faucet *FaucetRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Faucet.Contract.FaucetTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Faucet *FaucetRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Faucet.Contract.FaucetTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Faucet *FaucetCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Faucet.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Faucet *FaucetTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Faucet.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Faucet *FaucetTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Faucet.Contract.contract.Transact(opts, method, params...)
}

// TOKEN is a free data retrieval call binding the contract method 0x82bfefc8.
//
// Solidity: function TOKEN() view returns(address)
func (_Faucet *FaucetCaller) TOKEN(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Faucet.contract.Call(opts, &out, "TOKEN")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// TOKEN is a free data retrieval call binding the contract method 0x82bfefc8.
//
// Solidity: function TOKEN() view returns(address)
func (_Faucet *FaucetSession) TOKEN() (common.Address, error) {
	return _Faucet.Contract.TOKEN(&_Faucet.CallOpts)
}

// TOKEN is a free data retrieval call binding the contract method 0x82bfefc8.
//
// Solidity: function TOKEN() view returns(address)
func (_Faucet *FaucetCallerSession) TOKEN() (common.Address, error) {
	return _Faucet.Contract.TOKEN(&_Faucet.CallOpts)
}

// Cooldown is a free data retrieval call binding the contract method 0x787a08a6.
//
// Solidity: function cooldown() view returns(uint256)
func (_Faucet *FaucetCaller) Cooldown(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Faucet.contract.Call(opts, &out, "cooldown")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Cooldown is a free data retrieval call binding the contract method 0x787a08a6.
//
// Solidity: function cooldown() view returns(uint256)
func (_Faucet *FaucetSession) Cooldown() (*big.Int, error) {
	return _Faucet.Contract.Cooldown(&_Faucet.CallOpts)
}

// Cooldown is a free data retrieval call binding the contract method 0x787a08a6.
//
// Solidity: function cooldown() view returns(uint256)
func (_Faucet *FaucetCallerSession) Cooldown() (*big.Int, error) {
	return _Faucet.Contract.Cooldown(&_Faucet.CallOpts)
}

// DripAmount is a free data retrieval call binding the contract method 0x35a1529b.
//
// Solidity: function dripAmount() view returns(uint256)
func (_Faucet *FaucetCaller) DripAmount(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Faucet.contract.Call(opts, &out, "dripAmount")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// DripAmount is a free data retrieval call binding the contract method 0x35a1529b.
//
// Solidity: function dripAmount() view returns(uint256)
func (_Faucet *FaucetSession) DripAmount() (*big.Int, error) {
	return _Faucet.Contract.DripAmount(&_Faucet.CallOpts)
}

// DripAmount is a free data retrieval call binding the contract method 0x35a1529b.
//
// Solidity: function dripAmount() view returns(uint256)
func (_Faucet *FaucetCallerSession) DripAmount() (*big.Int, error) {
	return _Faucet.Contract.DripAmount(&_Faucet.CallOpts)
}

// LastDrip is a free data retrieval call binding the contract method 0x0935f004.
//
// Solidity: function lastDrip(address ) view returns(uint256)
func (_Faucet *FaucetCaller) LastDrip(opts *bind.CallOpts, arg0 common.Address) (*big.Int, error) {
	var out []interface{}
	err := _Faucet.contract.Call(opts, &out, "lastDrip", arg0)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// LastDrip is a free data retrieval call binding the contract method 0x0935f004.
//
// Solidity: function lastDrip(address ) view returns(uint256)
func (_Faucet *FaucetSession) LastDrip(arg0 common.Address) (*big.Int, error) {
	return _Faucet.Contract.LastDrip(&_Faucet.CallOpts, arg0)
}

// LastDrip is a free data retrieval call binding the contract method 0x0935f004.
//
// Solidity: function lastDrip(address ) view returns(uint256)
func (_Faucet *FaucetCallerSession) LastDrip(arg0 common.Address) (*big.Int, error) {
	return _Faucet.Contract.LastDrip(&_Faucet.CallOpts, arg0)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_Faucet *FaucetCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Faucet.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_Faucet *FaucetSession) Owner() (common.Address, error) {
	return _Faucet.Contract.Owner(&_Faucet.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_Faucet *FaucetCallerSession) Owner() (common.Address, error) {
	return _Faucet.Contract.Owner(&_Faucet.CallOpts)
}

// Drip is a paid mutator transaction binding the contract method 0x9f678cca.
//
// Solidity: function drip() returns()
func (_Faucet *FaucetTransactor) Drip(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Faucet.contract.Transact(opts, "drip")
}

// Drip is a paid mutator transaction binding the contract method 0x9f678cca.
//
// Solidity: function drip() returns()
func (_Faucet *FaucetSession) Drip() (*types.Transaction, error) {
	return _Faucet.Contract.Drip(&_Faucet.TransactOpts)
}

// Drip is a paid mutator transaction binding the contract method 0x9f678cca.
//
// Solidity: function drip() returns()
func (_Faucet *FaucetTransactorSession) Drip() (*types.Transaction, error) {
	return _Faucet.Contract.Drip(&_Faucet.TransactOpts)
}

// DripTo is a paid mutator transaction binding the contract method 0xcabee26e.
//
// Solidity: function dripTo(address recipient) returns()
func (_Faucet *FaucetTransactor) DripTo(opts *bind.TransactOpts, recipient common.Address) (*types.Transaction, error) {
	return _Faucet.contract.Transact(opts, "dripTo", recipient)
}

// DripTo is a paid mutator transaction binding the contract method 0xcabee26e.
//
// Solidity: function dripTo(address recipient) returns()
func (_Faucet *FaucetSession) DripTo(recipient common.Address) (*types.Transaction, error) {
	return _Faucet.Contract.DripTo(&_Faucet.TransactOpts, recipient)
}

// DripTo is a paid mutator transaction binding the contract method 0xcabee26e.
//
// Solidity: function dripTo(address recipient) returns()
func (_Faucet *FaucetTransactorSession) DripTo(recipient common.Address) (*types.Transaction, error) {
	return _Faucet.Contract.DripTo(&_Faucet.TransactOpts, recipient)
}

// SetCooldown is a paid mutator transaction binding the contract method 0x4fc3f41a.
//
// Solidity: function setCooldown(uint256 _cooldown) returns()
func (_Faucet *FaucetTransactor) SetCooldown(opts *bind.TransactOpts, _cooldown *big.Int) (*types.Transaction, error) {
	return _Faucet.contract.Transact(opts, "setCooldown", _cooldown)
}

// SetCooldown is a paid mutator transaction binding the contract method 0x4fc3f41a.
//
// Solidity: function setCooldown(uint256 _cooldown) returns()
func (_Faucet *FaucetSession) SetCooldown(_cooldown *big.Int) (*types.Transaction, error) {
	return _Faucet.Contract.SetCooldown(&_Faucet.TransactOpts, _cooldown)
}

// SetCooldown is a paid mutator transaction binding the contract method 0x4fc3f41a.
//
// Solidity: function setCooldown(uint256 _cooldown) returns()
func (_Faucet *FaucetTransactorSession) SetCooldown(_cooldown *big.Int) (*types.Transaction, error) {
	return _Faucet.Contract.SetCooldown(&_Faucet.TransactOpts, _cooldown)
}

// SetDripAmount is a paid mutator transaction binding the contract method 0x543f8c58.
//
// Solidity: function setDripAmount(uint256 _dripAmount) returns()
func (_Faucet *FaucetTransactor) SetDripAmount(opts *bind.TransactOpts, _dripAmount *big.Int) (*types.Transaction, error) {
	return _Faucet.contract.Transact(opts, "setDripAmount", _dripAmount)
}

// SetDripAmount is a paid mutator transaction binding the contract method 0x543f8c58.
//
// Solidity: function setDripAmount(uint256 _dripAmount) returns()
func (_Faucet *FaucetSession) SetDripAmount(_dripAmount *big.Int) (*types.Transaction, error) {
	return _Faucet.Contract.SetDripAmount(&_Faucet.TransactOpts, _dripAmount)
}

// SetDripAmount is a paid mutator transaction binding the contract method 0x543f8c58.
//
// Solidity: function setDripAmount(uint256 _dripAmount) returns()
func (_Faucet *FaucetTransactorSession) SetDripAmount(_dripAmount *big.Int) (*types.Transaction, error) {
	return _Faucet.Contract.SetDripAmount(&_Faucet.TransactOpts, _dripAmount)
}

// SetOwner is a paid mutator transaction binding the contract method 0x13af4035.
//
// Solidity: function setOwner(address _owner) returns()
func (_Faucet *FaucetTransactor) SetOwner(opts *bind.TransactOpts, _owner common.Address) (*types.Transaction, error) {
	return _Faucet.contract.Transact(opts, "setOwner", _owner)
}

// SetOwner is a paid mutator transaction binding the contract method 0x13af4035.
//
// Solidity: function setOwner(address _owner) returns()
func (_Faucet *FaucetSession) SetOwner(_owner common.Address) (*types.Transaction, error) {
	return _Faucet.Contract.SetOwner(&_Faucet.TransactOpts, _owner)
}

// SetOwner is a paid mutator transaction binding the contract method 0x13af4035.
//
// Solidity: function setOwner(address _owner) returns()
func (_Faucet *FaucetTransactorSession) SetOwner(_owner common.Address) (*types.Transaction, error) {
	return _Faucet.Contract.SetOwner(&_Faucet.TransactOpts, _owner)
}

// Withdraw is a paid mutator transaction binding the contract method 0x2e1a7d4d.
//
// Solidity: function withdraw(uint256 amount) returns()
func (_Faucet *FaucetTransactor) Withdraw(opts *bind.TransactOpts, amount *big.Int) (*types.Transaction, error) {
	return _Faucet.contract.Transact(opts, "withdraw", amount)
}

// Withdraw is a paid mutator transaction binding the contract method 0x2e1a7d4d.
//
// Solidity: function withdraw(uint256 amount) returns()
func (_Faucet *FaucetSession) Withdraw(amount *big.Int) (*types.Transaction, error) {
	return _Faucet.Contract.Withdraw(&_Faucet.TransactOpts, amount)
}

// Withdraw is a paid mutator transaction binding the contract method 0x2e1a7d4d.
//
// Solidity: function withdraw(uint256 amount) returns()
func (_Faucet *FaucetTransactorSession) Withdraw(amount *big.Int) (*types.Transaction, error) {
	return _Faucet.Contract.Withdraw(&_Faucet.TransactOpts, amount)
}

// FaucetCooldownUpdatedIterator is returned from FilterCooldownUpdated and is used to iterate over the raw logs and unpacked data for CooldownUpdated events raised by the Faucet contract.
type FaucetCooldownUpdatedIterator struct {
	Event *FaucetCooldownUpdated // Event containing the contract specifics and raw log

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
func (it *FaucetCooldownUpdatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(FaucetCooldownUpdated)
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
		it.Event = new(FaucetCooldownUpdated)
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
func (it *FaucetCooldownUpdatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *FaucetCooldownUpdatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// FaucetCooldownUpdated represents a CooldownUpdated event raised by the Faucet contract.
type FaucetCooldownUpdated struct {
	NewCooldown *big.Int
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterCooldownUpdated is a free log retrieval operation binding the contract event 0x583d8b24c5439ab7d810e51e37e8db41ba66f1168fd7b752ceae0c7681c5272c.
//
// Solidity: event CooldownUpdated(uint256 newCooldown)
func (_Faucet *FaucetFilterer) FilterCooldownUpdated(opts *bind.FilterOpts) (*FaucetCooldownUpdatedIterator, error) {

	logs, sub, err := _Faucet.contract.FilterLogs(opts, "CooldownUpdated")
	if err != nil {
		return nil, err
	}
	return &FaucetCooldownUpdatedIterator{contract: _Faucet.contract, event: "CooldownUpdated", logs: logs, sub: sub}, nil
}

// WatchCooldownUpdated is a free log subscription operation binding the contract event 0x583d8b24c5439ab7d810e51e37e8db41ba66f1168fd7b752ceae0c7681c5272c.
//
// Solidity: event CooldownUpdated(uint256 newCooldown)
func (_Faucet *FaucetFilterer) WatchCooldownUpdated(opts *bind.WatchOpts, sink chan<- *FaucetCooldownUpdated) (event.Subscription, error) {

	logs, sub, err := _Faucet.contract.WatchLogs(opts, "CooldownUpdated")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(FaucetCooldownUpdated)
				if err := _Faucet.contract.UnpackLog(event, "CooldownUpdated", log); err != nil {
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

// ParseCooldownUpdated is a log parse operation binding the contract event 0x583d8b24c5439ab7d810e51e37e8db41ba66f1168fd7b752ceae0c7681c5272c.
//
// Solidity: event CooldownUpdated(uint256 newCooldown)
func (_Faucet *FaucetFilterer) ParseCooldownUpdated(log types.Log) (*FaucetCooldownUpdated, error) {
	event := new(FaucetCooldownUpdated)
	if err := _Faucet.contract.UnpackLog(event, "CooldownUpdated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// FaucetDripAmountUpdatedIterator is returned from FilterDripAmountUpdated and is used to iterate over the raw logs and unpacked data for DripAmountUpdated events raised by the Faucet contract.
type FaucetDripAmountUpdatedIterator struct {
	Event *FaucetDripAmountUpdated // Event containing the contract specifics and raw log

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
func (it *FaucetDripAmountUpdatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(FaucetDripAmountUpdated)
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
		it.Event = new(FaucetDripAmountUpdated)
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
func (it *FaucetDripAmountUpdatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *FaucetDripAmountUpdatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// FaucetDripAmountUpdated represents a DripAmountUpdated event raised by the Faucet contract.
type FaucetDripAmountUpdated struct {
	NewAmount *big.Int
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterDripAmountUpdated is a free log retrieval operation binding the contract event 0x33f3faee0788ab897d8f674abe1dde6d93ba901e4a1502161294734ba178e3c7.
//
// Solidity: event DripAmountUpdated(uint256 newAmount)
func (_Faucet *FaucetFilterer) FilterDripAmountUpdated(opts *bind.FilterOpts) (*FaucetDripAmountUpdatedIterator, error) {

	logs, sub, err := _Faucet.contract.FilterLogs(opts, "DripAmountUpdated")
	if err != nil {
		return nil, err
	}
	return &FaucetDripAmountUpdatedIterator{contract: _Faucet.contract, event: "DripAmountUpdated", logs: logs, sub: sub}, nil
}

// WatchDripAmountUpdated is a free log subscription operation binding the contract event 0x33f3faee0788ab897d8f674abe1dde6d93ba901e4a1502161294734ba178e3c7.
//
// Solidity: event DripAmountUpdated(uint256 newAmount)
func (_Faucet *FaucetFilterer) WatchDripAmountUpdated(opts *bind.WatchOpts, sink chan<- *FaucetDripAmountUpdated) (event.Subscription, error) {

	logs, sub, err := _Faucet.contract.WatchLogs(opts, "DripAmountUpdated")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(FaucetDripAmountUpdated)
				if err := _Faucet.contract.UnpackLog(event, "DripAmountUpdated", log); err != nil {
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

// ParseDripAmountUpdated is a log parse operation binding the contract event 0x33f3faee0788ab897d8f674abe1dde6d93ba901e4a1502161294734ba178e3c7.
//
// Solidity: event DripAmountUpdated(uint256 newAmount)
func (_Faucet *FaucetFilterer) ParseDripAmountUpdated(log types.Log) (*FaucetDripAmountUpdated, error) {
	event := new(FaucetDripAmountUpdated)
	if err := _Faucet.contract.UnpackLog(event, "DripAmountUpdated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// FaucetDrippedIterator is returned from FilterDripped and is used to iterate over the raw logs and unpacked data for Dripped events raised by the Faucet contract.
type FaucetDrippedIterator struct {
	Event *FaucetDripped // Event containing the contract specifics and raw log

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
func (it *FaucetDrippedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(FaucetDripped)
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
		it.Event = new(FaucetDripped)
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
func (it *FaucetDrippedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *FaucetDrippedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// FaucetDripped represents a Dripped event raised by the Faucet contract.
type FaucetDripped struct {
	Recipient common.Address
	Amount    *big.Int
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterDripped is a free log retrieval operation binding the contract event 0x0daf449977d5acafa35195e10b3eb92f97839892a6653afaba222379b58d8a9b.
//
// Solidity: event Dripped(address indexed recipient, uint256 amount)
func (_Faucet *FaucetFilterer) FilterDripped(opts *bind.FilterOpts, recipient []common.Address) (*FaucetDrippedIterator, error) {

	var recipientRule []interface{}
	for _, recipientItem := range recipient {
		recipientRule = append(recipientRule, recipientItem)
	}

	logs, sub, err := _Faucet.contract.FilterLogs(opts, "Dripped", recipientRule)
	if err != nil {
		return nil, err
	}
	return &FaucetDrippedIterator{contract: _Faucet.contract, event: "Dripped", logs: logs, sub: sub}, nil
}

// WatchDripped is a free log subscription operation binding the contract event 0x0daf449977d5acafa35195e10b3eb92f97839892a6653afaba222379b58d8a9b.
//
// Solidity: event Dripped(address indexed recipient, uint256 amount)
func (_Faucet *FaucetFilterer) WatchDripped(opts *bind.WatchOpts, sink chan<- *FaucetDripped, recipient []common.Address) (event.Subscription, error) {

	var recipientRule []interface{}
	for _, recipientItem := range recipient {
		recipientRule = append(recipientRule, recipientItem)
	}

	logs, sub, err := _Faucet.contract.WatchLogs(opts, "Dripped", recipientRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(FaucetDripped)
				if err := _Faucet.contract.UnpackLog(event, "Dripped", log); err != nil {
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

// ParseDripped is a log parse operation binding the contract event 0x0daf449977d5acafa35195e10b3eb92f97839892a6653afaba222379b58d8a9b.
//
// Solidity: event Dripped(address indexed recipient, uint256 amount)
func (_Faucet *FaucetFilterer) ParseDripped(log types.Log) (*FaucetDripped, error) {
	event := new(FaucetDripped)
	if err := _Faucet.contract.UnpackLog(event, "Dripped", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// FaucetOwnerUpdatedIterator is returned from FilterOwnerUpdated and is used to iterate over the raw logs and unpacked data for OwnerUpdated events raised by the Faucet contract.
type FaucetOwnerUpdatedIterator struct {
	Event *FaucetOwnerUpdated // Event containing the contract specifics and raw log

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
func (it *FaucetOwnerUpdatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(FaucetOwnerUpdated)
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
		it.Event = new(FaucetOwnerUpdated)
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
func (it *FaucetOwnerUpdatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *FaucetOwnerUpdatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// FaucetOwnerUpdated represents a OwnerUpdated event raised by the Faucet contract.
type FaucetOwnerUpdated struct {
	NewOwner common.Address
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterOwnerUpdated is a free log retrieval operation binding the contract event 0x4ffd725fc4a22075e9ec71c59edf9c38cdeb588a91b24fc5b61388c5be41282b.
//
// Solidity: event OwnerUpdated(address indexed newOwner)
func (_Faucet *FaucetFilterer) FilterOwnerUpdated(opts *bind.FilterOpts, newOwner []common.Address) (*FaucetOwnerUpdatedIterator, error) {

	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _Faucet.contract.FilterLogs(opts, "OwnerUpdated", newOwnerRule)
	if err != nil {
		return nil, err
	}
	return &FaucetOwnerUpdatedIterator{contract: _Faucet.contract, event: "OwnerUpdated", logs: logs, sub: sub}, nil
}

// WatchOwnerUpdated is a free log subscription operation binding the contract event 0x4ffd725fc4a22075e9ec71c59edf9c38cdeb588a91b24fc5b61388c5be41282b.
//
// Solidity: event OwnerUpdated(address indexed newOwner)
func (_Faucet *FaucetFilterer) WatchOwnerUpdated(opts *bind.WatchOpts, sink chan<- *FaucetOwnerUpdated, newOwner []common.Address) (event.Subscription, error) {

	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _Faucet.contract.WatchLogs(opts, "OwnerUpdated", newOwnerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(FaucetOwnerUpdated)
				if err := _Faucet.contract.UnpackLog(event, "OwnerUpdated", log); err != nil {
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

// ParseOwnerUpdated is a log parse operation binding the contract event 0x4ffd725fc4a22075e9ec71c59edf9c38cdeb588a91b24fc5b61388c5be41282b.
//
// Solidity: event OwnerUpdated(address indexed newOwner)
func (_Faucet *FaucetFilterer) ParseOwnerUpdated(log types.Log) (*FaucetOwnerUpdated, error) {
	event := new(FaucetOwnerUpdated)
	if err := _Faucet.contract.UnpackLog(event, "OwnerUpdated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

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

// ClearnetRegistryConfigMetaData contains all meta data concerning the ClearnetRegistryConfig contract.
var ClearnetRegistryConfigMetaData = &bind.MetaData{
	ABI: "[{\"type\":\"function\",\"name\":\"CONFIG\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"}]",
}

// ClearnetRegistryConfigABI is the input ABI used to generate the binding from.
// Deprecated: Use ClearnetRegistryConfigMetaData.ABI instead.
var ClearnetRegistryConfigABI = ClearnetRegistryConfigMetaData.ABI

// ClearnetRegistryConfig is an auto generated Go binding around an Ethereum contract.
type ClearnetRegistryConfig struct {
	ClearnetRegistryConfigCaller     // Read-only binding to the contract
	ClearnetRegistryConfigTransactor // Write-only binding to the contract
	ClearnetRegistryConfigFilterer   // Log filterer for contract events
}

// ClearnetRegistryConfigCaller is an auto generated read-only Go binding around an Ethereum contract.
type ClearnetRegistryConfigCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ClearnetRegistryConfigTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ClearnetRegistryConfigTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ClearnetRegistryConfigFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type ClearnetRegistryConfigFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ClearnetRegistryConfigSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ClearnetRegistryConfigSession struct {
	Contract     *ClearnetRegistryConfig // Generic contract binding to set the session for
	CallOpts     bind.CallOpts           // Call options to use throughout this session
	TransactOpts bind.TransactOpts       // Transaction auth options to use throughout this session
}

// ClearnetRegistryConfigCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ClearnetRegistryConfigCallerSession struct {
	Contract *ClearnetRegistryConfigCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts                 // Call options to use throughout this session
}

// ClearnetRegistryConfigTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ClearnetRegistryConfigTransactorSession struct {
	Contract     *ClearnetRegistryConfigTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts                 // Transaction auth options to use throughout this session
}

// ClearnetRegistryConfigRaw is an auto generated low-level Go binding around an Ethereum contract.
type ClearnetRegistryConfigRaw struct {
	Contract *ClearnetRegistryConfig // Generic contract binding to access the raw methods on
}

// ClearnetRegistryConfigCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ClearnetRegistryConfigCallerRaw struct {
	Contract *ClearnetRegistryConfigCaller // Generic read-only contract binding to access the raw methods on
}

// ClearnetRegistryConfigTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ClearnetRegistryConfigTransactorRaw struct {
	Contract *ClearnetRegistryConfigTransactor // Generic write-only contract binding to access the raw methods on
}

// NewClearnetRegistryConfig creates a new instance of ClearnetRegistryConfig, bound to a specific deployed contract.
func NewClearnetRegistryConfig(address common.Address, backend bind.ContractBackend) (*ClearnetRegistryConfig, error) {
	contract, err := bindClearnetRegistryConfig(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &ClearnetRegistryConfig{ClearnetRegistryConfigCaller: ClearnetRegistryConfigCaller{contract: contract}, ClearnetRegistryConfigTransactor: ClearnetRegistryConfigTransactor{contract: contract}, ClearnetRegistryConfigFilterer: ClearnetRegistryConfigFilterer{contract: contract}}, nil
}

// NewClearnetRegistryConfigCaller creates a new read-only instance of ClearnetRegistryConfig, bound to a specific deployed contract.
func NewClearnetRegistryConfigCaller(address common.Address, caller bind.ContractCaller) (*ClearnetRegistryConfigCaller, error) {
	contract, err := bindClearnetRegistryConfig(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ClearnetRegistryConfigCaller{contract: contract}, nil
}

// NewClearnetRegistryConfigTransactor creates a new write-only instance of ClearnetRegistryConfig, bound to a specific deployed contract.
func NewClearnetRegistryConfigTransactor(address common.Address, transactor bind.ContractTransactor) (*ClearnetRegistryConfigTransactor, error) {
	contract, err := bindClearnetRegistryConfig(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &ClearnetRegistryConfigTransactor{contract: contract}, nil
}

// NewClearnetRegistryConfigFilterer creates a new log filterer instance of ClearnetRegistryConfig, bound to a specific deployed contract.
func NewClearnetRegistryConfigFilterer(address common.Address, filterer bind.ContractFilterer) (*ClearnetRegistryConfigFilterer, error) {
	contract, err := bindClearnetRegistryConfig(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &ClearnetRegistryConfigFilterer{contract: contract}, nil
}

// bindClearnetRegistryConfig binds a generic wrapper to an already deployed contract.
func bindClearnetRegistryConfig(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := ClearnetRegistryConfigMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ClearnetRegistryConfig *ClearnetRegistryConfigRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ClearnetRegistryConfig.Contract.ClearnetRegistryConfigCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ClearnetRegistryConfig *ClearnetRegistryConfigRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ClearnetRegistryConfig.Contract.ClearnetRegistryConfigTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ClearnetRegistryConfig *ClearnetRegistryConfigRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ClearnetRegistryConfig.Contract.ClearnetRegistryConfigTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ClearnetRegistryConfig *ClearnetRegistryConfigCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ClearnetRegistryConfig.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ClearnetRegistryConfig *ClearnetRegistryConfigTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ClearnetRegistryConfig.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ClearnetRegistryConfig *ClearnetRegistryConfigTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ClearnetRegistryConfig.Contract.contract.Transact(opts, method, params...)
}

// CONFIG is a free data retrieval call binding the contract method 0xd92e82e4.
//
// Solidity: function CONFIG() view returns(address)
func (_ClearnetRegistryConfig *ClearnetRegistryConfigCaller) CONFIG(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _ClearnetRegistryConfig.contract.Call(opts, &out, "CONFIG")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// CONFIG is a free data retrieval call binding the contract method 0xd92e82e4.
//
// Solidity: function CONFIG() view returns(address)
func (_ClearnetRegistryConfig *ClearnetRegistryConfigSession) CONFIG() (common.Address, error) {
	return _ClearnetRegistryConfig.Contract.CONFIG(&_ClearnetRegistryConfig.CallOpts)
}

// CONFIG is a free data retrieval call binding the contract method 0xd92e82e4.
//
// Solidity: function CONFIG() view returns(address)
func (_ClearnetRegistryConfig *ClearnetRegistryConfigCallerSession) CONFIG() (common.Address, error) {
	return _ClearnetRegistryConfig.Contract.CONFIG(&_ClearnetRegistryConfig.CallOpts)
}

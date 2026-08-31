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

// ConfigMetaData contains all meta data concerning the Config contract.
var ConfigMetaData = &bind.MetaData{
	ABI: "[{\"type\":\"constructor\",\"inputs\":[{\"name\":\"owner_\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"configChecksumAtEpoch\",\"inputs\":[{\"name\":\"key\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"epoch\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"configChecksums\",\"inputs\":[{\"name\":\"key\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32[]\",\"internalType\":\"bytes32[]\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"configEpoch\",\"inputs\":[{\"name\":\"key\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint64\",\"internalType\":\"uint64\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"latestConfigChecksum\",\"inputs\":[{\"name\":\"key\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"owner\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"setConfig\",\"inputs\":[{\"name\":\"key\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"checksum\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"setConfigWithData\",\"inputs\":[{\"name\":\"key\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"data\",\"type\":\"bytes\",\"internalType\":\"bytes\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"event\",\"name\":\"ConfigSet\",\"inputs\":[{\"name\":\"key\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"},{\"name\":\"writer\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"checksum\",\"type\":\"bytes32\",\"indexed\":false,\"internalType\":\"bytes32\"},{\"name\":\"epoch\",\"type\":\"uint64\",\"indexed\":false,\"internalType\":\"uint64\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"ConfigSetWithData\",\"inputs\":[{\"name\":\"key\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"},{\"name\":\"writer\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"checksum\",\"type\":\"bytes32\",\"indexed\":false,\"internalType\":\"bytes32\"},{\"name\":\"epoch\",\"type\":\"uint64\",\"indexed\":false,\"internalType\":\"uint64\"},{\"name\":\"data\",\"type\":\"bytes\",\"indexed\":false,\"internalType\":\"bytes\"}],\"anonymous\":false},{\"type\":\"error\",\"name\":\"EmptyConfigKey\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"EpochOutOfRange\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"InvalidOwner\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"NotConfigOwner\",\"inputs\":[]}]",
	Bin: "0x60a03461008d57601f61059438819003918201601f19168301916001600160401b038311848410176100915780849260209460405283398101031261008d57516001600160a01b03811680820361008d571561007e576080526040516104ee90816100a682396080518181816101280152818161023801526103130152f35b6349e27cff60e01b5f5260045ffd5b5f80fd5b634e487b7160e01b5f52604160045260245ffdfe60806040526004361015610011575f80fd5b5f3560e01c80633cb37e57146103c557806346c736b5146103425780638da5cb5b146102fe5780639f0cee4a146101df578063af890358146101ac578063d1fd27b31461010f5763fec5bedb14610066575f80fd5b3461010b57602036600319011261010b576004355f525f60205260405f20604051806020835491828152019081935f5260205f20905f5b8181106100f557505050816100b3910382610452565b604051918291602083019060208452518091526040830191905f5b8181106100dc575050500390f35b82518452859450602093840193909201916001016100ce565b825484526020909301926001928301920161009d565b5f80fd5b3461010b57604036600319011261010b576024356004357f00000000000000000000000000000000000000000000000000000000000000006001600160a01b0316330361019d5767ffffffffffffffff6101698383610488565b6040519384521660208301527f952e9f054f8f14436d21495120b5658808398d89e8240fca0e0b5669e4dbb01360403393a3005b637138837360e11b5f5260045ffd5b3461010b57602036600319011261010b576004355f525f602052602067ffffffffffffffff60405f205416604051908152f35b3461010b57604036600319011261010b5760243560043567ffffffffffffffff821161010b573660238301121561010b5781600401359167ffffffffffffffff831161010b576024810190602484369201011161010b577f00000000000000000000000000000000000000000000000000000000000000006001600160a01b0316330361019d577f138bbba807de4352fcbdebae920532cde6774dba6fd049b88ea55bc3af412c00905f6080601f19601f87011695806040516102a560208a0182610452565b81815260208101908287833785602084830101525190209467ffffffffffffffff6102d0878a610488565b60405197885216602087015260606040870152816060870152838601378301015260808133958101030190a3005b3461010b575f36600319011261010b576040517f00000000000000000000000000000000000000000000000000000000000000006001600160a01b03168152602090f35b3461010b57604036600319011261010b576024356004355f525f60205260405f20811515806103ba575b156103ab575f1982019182116103975760209161038891610429565b90549060031b1c604051908152f35b634e487b7160e01b5f52601160045260245ffd5b6316f4c85360e01b5f5260045ffd5b50805482111561036c565b3461010b57602036600319011261010b57600435805f525f60205260405f20549081155f146103fc57505060205f5b604051908152f35b5f525f60205260405f205f1982019182116103975760209161041d91610429565b90549060031b1c6103f4565b805482101561043e575f5260205f2001905f90565b634e487b7160e01b5f52603260045260245ffd5b90601f8019910116810190811067ffffffffffffffff82111761047457604052565b634e487b7160e01b5f52604160045260245ffd5b80156104df575f525f60205260405f2080549168010000000000000000831015610474576104c583600167ffffffffffffffff9501845583610429565b819291549060031b91821b915f19901b1916179055541690565b6355a9397560e11b5f5260045ffd",
}

// ConfigABI is the input ABI used to generate the binding from.
// Deprecated: Use ConfigMetaData.ABI instead.
var ConfigABI = ConfigMetaData.ABI

// ConfigBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use ConfigMetaData.Bin instead.
var ConfigBin = ConfigMetaData.Bin

// DeployConfig deploys a new Ethereum contract, binding an instance of Config to it.
func DeployConfig(auth *bind.TransactOpts, backend bind.ContractBackend, owner_ common.Address) (common.Address, *types.Transaction, *Config, error) {
	parsed, err := ConfigMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(ConfigBin), backend, owner_)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &Config{ConfigCaller: ConfigCaller{contract: contract}, ConfigTransactor: ConfigTransactor{contract: contract}, ConfigFilterer: ConfigFilterer{contract: contract}}, nil
}

// Config is an auto generated Go binding around an Ethereum contract.
type Config struct {
	ConfigCaller     // Read-only binding to the contract
	ConfigTransactor // Write-only binding to the contract
	ConfigFilterer   // Log filterer for contract events
}

// ConfigCaller is an auto generated read-only Go binding around an Ethereum contract.
type ConfigCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ConfigTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ConfigTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ConfigFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type ConfigFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ConfigSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ConfigSession struct {
	Contract     *Config           // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// ConfigCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ConfigCallerSession struct {
	Contract *ConfigCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts // Call options to use throughout this session
}

// ConfigTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ConfigTransactorSession struct {
	Contract     *ConfigTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// ConfigRaw is an auto generated low-level Go binding around an Ethereum contract.
type ConfigRaw struct {
	Contract *Config // Generic contract binding to access the raw methods on
}

// ConfigCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ConfigCallerRaw struct {
	Contract *ConfigCaller // Generic read-only contract binding to access the raw methods on
}

// ConfigTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ConfigTransactorRaw struct {
	Contract *ConfigTransactor // Generic write-only contract binding to access the raw methods on
}

// NewConfig creates a new instance of Config, bound to a specific deployed contract.
func NewConfig(address common.Address, backend bind.ContractBackend) (*Config, error) {
	contract, err := bindConfig(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Config{ConfigCaller: ConfigCaller{contract: contract}, ConfigTransactor: ConfigTransactor{contract: contract}, ConfigFilterer: ConfigFilterer{contract: contract}}, nil
}

// NewConfigCaller creates a new read-only instance of Config, bound to a specific deployed contract.
func NewConfigCaller(address common.Address, caller bind.ContractCaller) (*ConfigCaller, error) {
	contract, err := bindConfig(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ConfigCaller{contract: contract}, nil
}

// NewConfigTransactor creates a new write-only instance of Config, bound to a specific deployed contract.
func NewConfigTransactor(address common.Address, transactor bind.ContractTransactor) (*ConfigTransactor, error) {
	contract, err := bindConfig(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &ConfigTransactor{contract: contract}, nil
}

// NewConfigFilterer creates a new log filterer instance of Config, bound to a specific deployed contract.
func NewConfigFilterer(address common.Address, filterer bind.ContractFilterer) (*ConfigFilterer, error) {
	contract, err := bindConfig(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &ConfigFilterer{contract: contract}, nil
}

// bindConfig binds a generic wrapper to an already deployed contract.
func bindConfig(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := ConfigMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Config *ConfigRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Config.Contract.ConfigCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Config *ConfigRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Config.Contract.ConfigTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Config *ConfigRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Config.Contract.ConfigTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Config *ConfigCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Config.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Config *ConfigTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Config.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Config *ConfigTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Config.Contract.contract.Transact(opts, method, params...)
}

// ConfigChecksumAtEpoch is a free data retrieval call binding the contract method 0x46c736b5.
//
// Solidity: function configChecksumAtEpoch(bytes32 key, uint256 epoch) view returns(bytes32)
func (_Config *ConfigCaller) ConfigChecksumAtEpoch(opts *bind.CallOpts, key [32]byte, epoch *big.Int) ([32]byte, error) {
	var out []interface{}
	err := _Config.contract.Call(opts, &out, "configChecksumAtEpoch", key, epoch)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// ConfigChecksumAtEpoch is a free data retrieval call binding the contract method 0x46c736b5.
//
// Solidity: function configChecksumAtEpoch(bytes32 key, uint256 epoch) view returns(bytes32)
func (_Config *ConfigSession) ConfigChecksumAtEpoch(key [32]byte, epoch *big.Int) ([32]byte, error) {
	return _Config.Contract.ConfigChecksumAtEpoch(&_Config.CallOpts, key, epoch)
}

// ConfigChecksumAtEpoch is a free data retrieval call binding the contract method 0x46c736b5.
//
// Solidity: function configChecksumAtEpoch(bytes32 key, uint256 epoch) view returns(bytes32)
func (_Config *ConfigCallerSession) ConfigChecksumAtEpoch(key [32]byte, epoch *big.Int) ([32]byte, error) {
	return _Config.Contract.ConfigChecksumAtEpoch(&_Config.CallOpts, key, epoch)
}

// ConfigChecksums is a free data retrieval call binding the contract method 0xfec5bedb.
//
// Solidity: function configChecksums(bytes32 key) view returns(bytes32[])
func (_Config *ConfigCaller) ConfigChecksums(opts *bind.CallOpts, key [32]byte) ([][32]byte, error) {
	var out []interface{}
	err := _Config.contract.Call(opts, &out, "configChecksums", key)

	if err != nil {
		return *new([][32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([][32]byte)).(*[][32]byte)

	return out0, err

}

// ConfigChecksums is a free data retrieval call binding the contract method 0xfec5bedb.
//
// Solidity: function configChecksums(bytes32 key) view returns(bytes32[])
func (_Config *ConfigSession) ConfigChecksums(key [32]byte) ([][32]byte, error) {
	return _Config.Contract.ConfigChecksums(&_Config.CallOpts, key)
}

// ConfigChecksums is a free data retrieval call binding the contract method 0xfec5bedb.
//
// Solidity: function configChecksums(bytes32 key) view returns(bytes32[])
func (_Config *ConfigCallerSession) ConfigChecksums(key [32]byte) ([][32]byte, error) {
	return _Config.Contract.ConfigChecksums(&_Config.CallOpts, key)
}

// ConfigEpoch is a free data retrieval call binding the contract method 0xaf890358.
//
// Solidity: function configEpoch(bytes32 key) view returns(uint64)
func (_Config *ConfigCaller) ConfigEpoch(opts *bind.CallOpts, key [32]byte) (uint64, error) {
	var out []interface{}
	err := _Config.contract.Call(opts, &out, "configEpoch", key)

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// ConfigEpoch is a free data retrieval call binding the contract method 0xaf890358.
//
// Solidity: function configEpoch(bytes32 key) view returns(uint64)
func (_Config *ConfigSession) ConfigEpoch(key [32]byte) (uint64, error) {
	return _Config.Contract.ConfigEpoch(&_Config.CallOpts, key)
}

// ConfigEpoch is a free data retrieval call binding the contract method 0xaf890358.
//
// Solidity: function configEpoch(bytes32 key) view returns(uint64)
func (_Config *ConfigCallerSession) ConfigEpoch(key [32]byte) (uint64, error) {
	return _Config.Contract.ConfigEpoch(&_Config.CallOpts, key)
}

// LatestConfigChecksum is a free data retrieval call binding the contract method 0x3cb37e57.
//
// Solidity: function latestConfigChecksum(bytes32 key) view returns(bytes32)
func (_Config *ConfigCaller) LatestConfigChecksum(opts *bind.CallOpts, key [32]byte) ([32]byte, error) {
	var out []interface{}
	err := _Config.contract.Call(opts, &out, "latestConfigChecksum", key)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// LatestConfigChecksum is a free data retrieval call binding the contract method 0x3cb37e57.
//
// Solidity: function latestConfigChecksum(bytes32 key) view returns(bytes32)
func (_Config *ConfigSession) LatestConfigChecksum(key [32]byte) ([32]byte, error) {
	return _Config.Contract.LatestConfigChecksum(&_Config.CallOpts, key)
}

// LatestConfigChecksum is a free data retrieval call binding the contract method 0x3cb37e57.
//
// Solidity: function latestConfigChecksum(bytes32 key) view returns(bytes32)
func (_Config *ConfigCallerSession) LatestConfigChecksum(key [32]byte) ([32]byte, error) {
	return _Config.Contract.LatestConfigChecksum(&_Config.CallOpts, key)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_Config *ConfigCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Config.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_Config *ConfigSession) Owner() (common.Address, error) {
	return _Config.Contract.Owner(&_Config.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_Config *ConfigCallerSession) Owner() (common.Address, error) {
	return _Config.Contract.Owner(&_Config.CallOpts)
}

// SetConfig is a paid mutator transaction binding the contract method 0xd1fd27b3.
//
// Solidity: function setConfig(bytes32 key, bytes32 checksum) returns()
func (_Config *ConfigTransactor) SetConfig(opts *bind.TransactOpts, key [32]byte, checksum [32]byte) (*types.Transaction, error) {
	return _Config.contract.Transact(opts, "setConfig", key, checksum)
}

// SetConfig is a paid mutator transaction binding the contract method 0xd1fd27b3.
//
// Solidity: function setConfig(bytes32 key, bytes32 checksum) returns()
func (_Config *ConfigSession) SetConfig(key [32]byte, checksum [32]byte) (*types.Transaction, error) {
	return _Config.Contract.SetConfig(&_Config.TransactOpts, key, checksum)
}

// SetConfig is a paid mutator transaction binding the contract method 0xd1fd27b3.
//
// Solidity: function setConfig(bytes32 key, bytes32 checksum) returns()
func (_Config *ConfigTransactorSession) SetConfig(key [32]byte, checksum [32]byte) (*types.Transaction, error) {
	return _Config.Contract.SetConfig(&_Config.TransactOpts, key, checksum)
}

// SetConfigWithData is a paid mutator transaction binding the contract method 0x9f0cee4a.
//
// Solidity: function setConfigWithData(bytes32 key, bytes data) returns()
func (_Config *ConfigTransactor) SetConfigWithData(opts *bind.TransactOpts, key [32]byte, data []byte) (*types.Transaction, error) {
	return _Config.contract.Transact(opts, "setConfigWithData", key, data)
}

// SetConfigWithData is a paid mutator transaction binding the contract method 0x9f0cee4a.
//
// Solidity: function setConfigWithData(bytes32 key, bytes data) returns()
func (_Config *ConfigSession) SetConfigWithData(key [32]byte, data []byte) (*types.Transaction, error) {
	return _Config.Contract.SetConfigWithData(&_Config.TransactOpts, key, data)
}

// SetConfigWithData is a paid mutator transaction binding the contract method 0x9f0cee4a.
//
// Solidity: function setConfigWithData(bytes32 key, bytes data) returns()
func (_Config *ConfigTransactorSession) SetConfigWithData(key [32]byte, data []byte) (*types.Transaction, error) {
	return _Config.Contract.SetConfigWithData(&_Config.TransactOpts, key, data)
}

// ConfigConfigSetIterator is returned from FilterConfigSet and is used to iterate over the raw logs and unpacked data for ConfigSet events raised by the Config contract.
type ConfigConfigSetIterator struct {
	Event *ConfigConfigSet // Event containing the contract specifics and raw log

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
func (it *ConfigConfigSetIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ConfigConfigSet)
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
		it.Event = new(ConfigConfigSet)
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
func (it *ConfigConfigSetIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ConfigConfigSetIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ConfigConfigSet represents a ConfigSet event raised by the Config contract.
type ConfigConfigSet struct {
	Key      [32]byte
	Writer   common.Address
	Checksum [32]byte
	Epoch    uint64
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterConfigSet is a free log retrieval operation binding the contract event 0x952e9f054f8f14436d21495120b5658808398d89e8240fca0e0b5669e4dbb013.
//
// Solidity: event ConfigSet(bytes32 indexed key, address indexed writer, bytes32 checksum, uint64 epoch)
func (_Config *ConfigFilterer) FilterConfigSet(opts *bind.FilterOpts, key [][32]byte, writer []common.Address) (*ConfigConfigSetIterator, error) {

	var keyRule []interface{}
	for _, keyItem := range key {
		keyRule = append(keyRule, keyItem)
	}
	var writerRule []interface{}
	for _, writerItem := range writer {
		writerRule = append(writerRule, writerItem)
	}

	logs, sub, err := _Config.contract.FilterLogs(opts, "ConfigSet", keyRule, writerRule)
	if err != nil {
		return nil, err
	}
	return &ConfigConfigSetIterator{contract: _Config.contract, event: "ConfigSet", logs: logs, sub: sub}, nil
}

// WatchConfigSet is a free log subscription operation binding the contract event 0x952e9f054f8f14436d21495120b5658808398d89e8240fca0e0b5669e4dbb013.
//
// Solidity: event ConfigSet(bytes32 indexed key, address indexed writer, bytes32 checksum, uint64 epoch)
func (_Config *ConfigFilterer) WatchConfigSet(opts *bind.WatchOpts, sink chan<- *ConfigConfigSet, key [][32]byte, writer []common.Address) (event.Subscription, error) {

	var keyRule []interface{}
	for _, keyItem := range key {
		keyRule = append(keyRule, keyItem)
	}
	var writerRule []interface{}
	for _, writerItem := range writer {
		writerRule = append(writerRule, writerItem)
	}

	logs, sub, err := _Config.contract.WatchLogs(opts, "ConfigSet", keyRule, writerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ConfigConfigSet)
				if err := _Config.contract.UnpackLog(event, "ConfigSet", log); err != nil {
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

// ParseConfigSet is a log parse operation binding the contract event 0x952e9f054f8f14436d21495120b5658808398d89e8240fca0e0b5669e4dbb013.
//
// Solidity: event ConfigSet(bytes32 indexed key, address indexed writer, bytes32 checksum, uint64 epoch)
func (_Config *ConfigFilterer) ParseConfigSet(log types.Log) (*ConfigConfigSet, error) {
	event := new(ConfigConfigSet)
	if err := _Config.contract.UnpackLog(event, "ConfigSet", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ConfigConfigSetWithDataIterator is returned from FilterConfigSetWithData and is used to iterate over the raw logs and unpacked data for ConfigSetWithData events raised by the Config contract.
type ConfigConfigSetWithDataIterator struct {
	Event *ConfigConfigSetWithData // Event containing the contract specifics and raw log

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
func (it *ConfigConfigSetWithDataIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ConfigConfigSetWithData)
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
		it.Event = new(ConfigConfigSetWithData)
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
func (it *ConfigConfigSetWithDataIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ConfigConfigSetWithDataIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ConfigConfigSetWithData represents a ConfigSetWithData event raised by the Config contract.
type ConfigConfigSetWithData struct {
	Key      [32]byte
	Writer   common.Address
	Checksum [32]byte
	Epoch    uint64
	Data     []byte
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterConfigSetWithData is a free log retrieval operation binding the contract event 0x138bbba807de4352fcbdebae920532cde6774dba6fd049b88ea55bc3af412c00.
//
// Solidity: event ConfigSetWithData(bytes32 indexed key, address indexed writer, bytes32 checksum, uint64 epoch, bytes data)
func (_Config *ConfigFilterer) FilterConfigSetWithData(opts *bind.FilterOpts, key [][32]byte, writer []common.Address) (*ConfigConfigSetWithDataIterator, error) {

	var keyRule []interface{}
	for _, keyItem := range key {
		keyRule = append(keyRule, keyItem)
	}
	var writerRule []interface{}
	for _, writerItem := range writer {
		writerRule = append(writerRule, writerItem)
	}

	logs, sub, err := _Config.contract.FilterLogs(opts, "ConfigSetWithData", keyRule, writerRule)
	if err != nil {
		return nil, err
	}
	return &ConfigConfigSetWithDataIterator{contract: _Config.contract, event: "ConfigSetWithData", logs: logs, sub: sub}, nil
}

// WatchConfigSetWithData is a free log subscription operation binding the contract event 0x138bbba807de4352fcbdebae920532cde6774dba6fd049b88ea55bc3af412c00.
//
// Solidity: event ConfigSetWithData(bytes32 indexed key, address indexed writer, bytes32 checksum, uint64 epoch, bytes data)
func (_Config *ConfigFilterer) WatchConfigSetWithData(opts *bind.WatchOpts, sink chan<- *ConfigConfigSetWithData, key [][32]byte, writer []common.Address) (event.Subscription, error) {

	var keyRule []interface{}
	for _, keyItem := range key {
		keyRule = append(keyRule, keyItem)
	}
	var writerRule []interface{}
	for _, writerItem := range writer {
		writerRule = append(writerRule, writerItem)
	}

	logs, sub, err := _Config.contract.WatchLogs(opts, "ConfigSetWithData", keyRule, writerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ConfigConfigSetWithData)
				if err := _Config.contract.UnpackLog(event, "ConfigSetWithData", log); err != nil {
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

// ParseConfigSetWithData is a log parse operation binding the contract event 0x138bbba807de4352fcbdebae920532cde6774dba6fd049b88ea55bc3af412c00.
//
// Solidity: event ConfigSetWithData(bytes32 indexed key, address indexed writer, bytes32 checksum, uint64 epoch, bytes data)
func (_Config *ConfigFilterer) ParseConfigSetWithData(log types.Log) (*ConfigConfigSetWithData, error) {
	event := new(ConfigConfigSetWithData)
	if err := _Config.contract.UnpackLog(event, "ConfigSetWithData", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

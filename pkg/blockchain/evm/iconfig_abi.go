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

// IConfigMetaData contains all meta data concerning the IConfig contract.
var IConfigMetaData = &bind.MetaData{
	ABI: "[{\"type\":\"function\",\"name\":\"configChecksumAtEpoch\",\"inputs\":[{\"name\":\"key\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"epoch\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"configChecksums\",\"inputs\":[{\"name\":\"key\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32[]\",\"internalType\":\"bytes32[]\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"configEpoch\",\"inputs\":[{\"name\":\"key\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint64\",\"internalType\":\"uint64\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"latestConfigChecksum\",\"inputs\":[{\"name\":\"key\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"owner\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"setConfig\",\"inputs\":[{\"name\":\"key\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"checksum\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"setConfigWithData\",\"inputs\":[{\"name\":\"key\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"data\",\"type\":\"bytes\",\"internalType\":\"bytes\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"event\",\"name\":\"ConfigSet\",\"inputs\":[{\"name\":\"key\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"},{\"name\":\"writer\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"checksum\",\"type\":\"bytes32\",\"indexed\":false,\"internalType\":\"bytes32\"},{\"name\":\"epoch\",\"type\":\"uint64\",\"indexed\":false,\"internalType\":\"uint64\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"ConfigSetWithData\",\"inputs\":[{\"name\":\"key\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"},{\"name\":\"writer\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"checksum\",\"type\":\"bytes32\",\"indexed\":false,\"internalType\":\"bytes32\"},{\"name\":\"epoch\",\"type\":\"uint64\",\"indexed\":false,\"internalType\":\"uint64\"},{\"name\":\"data\",\"type\":\"bytes\",\"indexed\":false,\"internalType\":\"bytes\"}],\"anonymous\":false},{\"type\":\"error\",\"name\":\"EmptyConfigKey\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"EpochOutOfRange\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"InvalidOwner\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"NotConfigOwner\",\"inputs\":[]}]",
}

// IConfigABI is the input ABI used to generate the binding from.
// Deprecated: Use IConfigMetaData.ABI instead.
var IConfigABI = IConfigMetaData.ABI

// IConfig is an auto generated Go binding around an Ethereum contract.
type IConfig struct {
	IConfigCaller     // Read-only binding to the contract
	IConfigTransactor // Write-only binding to the contract
	IConfigFilterer   // Log filterer for contract events
}

// IConfigCaller is an auto generated read-only Go binding around an Ethereum contract.
type IConfigCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IConfigTransactor is an auto generated write-only Go binding around an Ethereum contract.
type IConfigTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IConfigFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type IConfigFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IConfigSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type IConfigSession struct {
	Contract     *IConfig          // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// IConfigCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type IConfigCallerSession struct {
	Contract *IConfigCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts  // Call options to use throughout this session
}

// IConfigTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type IConfigTransactorSession struct {
	Contract     *IConfigTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts  // Transaction auth options to use throughout this session
}

// IConfigRaw is an auto generated low-level Go binding around an Ethereum contract.
type IConfigRaw struct {
	Contract *IConfig // Generic contract binding to access the raw methods on
}

// IConfigCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type IConfigCallerRaw struct {
	Contract *IConfigCaller // Generic read-only contract binding to access the raw methods on
}

// IConfigTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type IConfigTransactorRaw struct {
	Contract *IConfigTransactor // Generic write-only contract binding to access the raw methods on
}

// NewIConfig creates a new instance of IConfig, bound to a specific deployed contract.
func NewIConfig(address common.Address, backend bind.ContractBackend) (*IConfig, error) {
	contract, err := bindIConfig(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &IConfig{IConfigCaller: IConfigCaller{contract: contract}, IConfigTransactor: IConfigTransactor{contract: contract}, IConfigFilterer: IConfigFilterer{contract: contract}}, nil
}

// NewIConfigCaller creates a new read-only instance of IConfig, bound to a specific deployed contract.
func NewIConfigCaller(address common.Address, caller bind.ContractCaller) (*IConfigCaller, error) {
	contract, err := bindIConfig(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &IConfigCaller{contract: contract}, nil
}

// NewIConfigTransactor creates a new write-only instance of IConfig, bound to a specific deployed contract.
func NewIConfigTransactor(address common.Address, transactor bind.ContractTransactor) (*IConfigTransactor, error) {
	contract, err := bindIConfig(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &IConfigTransactor{contract: contract}, nil
}

// NewIConfigFilterer creates a new log filterer instance of IConfig, bound to a specific deployed contract.
func NewIConfigFilterer(address common.Address, filterer bind.ContractFilterer) (*IConfigFilterer, error) {
	contract, err := bindIConfig(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &IConfigFilterer{contract: contract}, nil
}

// bindIConfig binds a generic wrapper to an already deployed contract.
func bindIConfig(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := IConfigMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_IConfig *IConfigRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _IConfig.Contract.IConfigCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_IConfig *IConfigRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _IConfig.Contract.IConfigTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_IConfig *IConfigRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _IConfig.Contract.IConfigTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_IConfig *IConfigCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _IConfig.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_IConfig *IConfigTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _IConfig.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_IConfig *IConfigTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _IConfig.Contract.contract.Transact(opts, method, params...)
}

// ConfigChecksumAtEpoch is a free data retrieval call binding the contract method 0x46c736b5.
//
// Solidity: function configChecksumAtEpoch(bytes32 key, uint256 epoch) view returns(bytes32)
func (_IConfig *IConfigCaller) ConfigChecksumAtEpoch(opts *bind.CallOpts, key [32]byte, epoch *big.Int) ([32]byte, error) {
	var out []interface{}
	err := _IConfig.contract.Call(opts, &out, "configChecksumAtEpoch", key, epoch)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// ConfigChecksumAtEpoch is a free data retrieval call binding the contract method 0x46c736b5.
//
// Solidity: function configChecksumAtEpoch(bytes32 key, uint256 epoch) view returns(bytes32)
func (_IConfig *IConfigSession) ConfigChecksumAtEpoch(key [32]byte, epoch *big.Int) ([32]byte, error) {
	return _IConfig.Contract.ConfigChecksumAtEpoch(&_IConfig.CallOpts, key, epoch)
}

// ConfigChecksumAtEpoch is a free data retrieval call binding the contract method 0x46c736b5.
//
// Solidity: function configChecksumAtEpoch(bytes32 key, uint256 epoch) view returns(bytes32)
func (_IConfig *IConfigCallerSession) ConfigChecksumAtEpoch(key [32]byte, epoch *big.Int) ([32]byte, error) {
	return _IConfig.Contract.ConfigChecksumAtEpoch(&_IConfig.CallOpts, key, epoch)
}

// ConfigChecksums is a free data retrieval call binding the contract method 0xfec5bedb.
//
// Solidity: function configChecksums(bytes32 key) view returns(bytes32[])
func (_IConfig *IConfigCaller) ConfigChecksums(opts *bind.CallOpts, key [32]byte) ([][32]byte, error) {
	var out []interface{}
	err := _IConfig.contract.Call(opts, &out, "configChecksums", key)

	if err != nil {
		return *new([][32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([][32]byte)).(*[][32]byte)

	return out0, err

}

// ConfigChecksums is a free data retrieval call binding the contract method 0xfec5bedb.
//
// Solidity: function configChecksums(bytes32 key) view returns(bytes32[])
func (_IConfig *IConfigSession) ConfigChecksums(key [32]byte) ([][32]byte, error) {
	return _IConfig.Contract.ConfigChecksums(&_IConfig.CallOpts, key)
}

// ConfigChecksums is a free data retrieval call binding the contract method 0xfec5bedb.
//
// Solidity: function configChecksums(bytes32 key) view returns(bytes32[])
func (_IConfig *IConfigCallerSession) ConfigChecksums(key [32]byte) ([][32]byte, error) {
	return _IConfig.Contract.ConfigChecksums(&_IConfig.CallOpts, key)
}

// ConfigEpoch is a free data retrieval call binding the contract method 0xaf890358.
//
// Solidity: function configEpoch(bytes32 key) view returns(uint64)
func (_IConfig *IConfigCaller) ConfigEpoch(opts *bind.CallOpts, key [32]byte) (uint64, error) {
	var out []interface{}
	err := _IConfig.contract.Call(opts, &out, "configEpoch", key)

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// ConfigEpoch is a free data retrieval call binding the contract method 0xaf890358.
//
// Solidity: function configEpoch(bytes32 key) view returns(uint64)
func (_IConfig *IConfigSession) ConfigEpoch(key [32]byte) (uint64, error) {
	return _IConfig.Contract.ConfigEpoch(&_IConfig.CallOpts, key)
}

// ConfigEpoch is a free data retrieval call binding the contract method 0xaf890358.
//
// Solidity: function configEpoch(bytes32 key) view returns(uint64)
func (_IConfig *IConfigCallerSession) ConfigEpoch(key [32]byte) (uint64, error) {
	return _IConfig.Contract.ConfigEpoch(&_IConfig.CallOpts, key)
}

// LatestConfigChecksum is a free data retrieval call binding the contract method 0x3cb37e57.
//
// Solidity: function latestConfigChecksum(bytes32 key) view returns(bytes32)
func (_IConfig *IConfigCaller) LatestConfigChecksum(opts *bind.CallOpts, key [32]byte) ([32]byte, error) {
	var out []interface{}
	err := _IConfig.contract.Call(opts, &out, "latestConfigChecksum", key)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// LatestConfigChecksum is a free data retrieval call binding the contract method 0x3cb37e57.
//
// Solidity: function latestConfigChecksum(bytes32 key) view returns(bytes32)
func (_IConfig *IConfigSession) LatestConfigChecksum(key [32]byte) ([32]byte, error) {
	return _IConfig.Contract.LatestConfigChecksum(&_IConfig.CallOpts, key)
}

// LatestConfigChecksum is a free data retrieval call binding the contract method 0x3cb37e57.
//
// Solidity: function latestConfigChecksum(bytes32 key) view returns(bytes32)
func (_IConfig *IConfigCallerSession) LatestConfigChecksum(key [32]byte) ([32]byte, error) {
	return _IConfig.Contract.LatestConfigChecksum(&_IConfig.CallOpts, key)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_IConfig *IConfigCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _IConfig.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_IConfig *IConfigSession) Owner() (common.Address, error) {
	return _IConfig.Contract.Owner(&_IConfig.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_IConfig *IConfigCallerSession) Owner() (common.Address, error) {
	return _IConfig.Contract.Owner(&_IConfig.CallOpts)
}

// SetConfig is a paid mutator transaction binding the contract method 0xd1fd27b3.
//
// Solidity: function setConfig(bytes32 key, bytes32 checksum) returns()
func (_IConfig *IConfigTransactor) SetConfig(opts *bind.TransactOpts, key [32]byte, checksum [32]byte) (*types.Transaction, error) {
	return _IConfig.contract.Transact(opts, "setConfig", key, checksum)
}

// SetConfig is a paid mutator transaction binding the contract method 0xd1fd27b3.
//
// Solidity: function setConfig(bytes32 key, bytes32 checksum) returns()
func (_IConfig *IConfigSession) SetConfig(key [32]byte, checksum [32]byte) (*types.Transaction, error) {
	return _IConfig.Contract.SetConfig(&_IConfig.TransactOpts, key, checksum)
}

// SetConfig is a paid mutator transaction binding the contract method 0xd1fd27b3.
//
// Solidity: function setConfig(bytes32 key, bytes32 checksum) returns()
func (_IConfig *IConfigTransactorSession) SetConfig(key [32]byte, checksum [32]byte) (*types.Transaction, error) {
	return _IConfig.Contract.SetConfig(&_IConfig.TransactOpts, key, checksum)
}

// SetConfigWithData is a paid mutator transaction binding the contract method 0x9f0cee4a.
//
// Solidity: function setConfigWithData(bytes32 key, bytes data) returns()
func (_IConfig *IConfigTransactor) SetConfigWithData(opts *bind.TransactOpts, key [32]byte, data []byte) (*types.Transaction, error) {
	return _IConfig.contract.Transact(opts, "setConfigWithData", key, data)
}

// SetConfigWithData is a paid mutator transaction binding the contract method 0x9f0cee4a.
//
// Solidity: function setConfigWithData(bytes32 key, bytes data) returns()
func (_IConfig *IConfigSession) SetConfigWithData(key [32]byte, data []byte) (*types.Transaction, error) {
	return _IConfig.Contract.SetConfigWithData(&_IConfig.TransactOpts, key, data)
}

// SetConfigWithData is a paid mutator transaction binding the contract method 0x9f0cee4a.
//
// Solidity: function setConfigWithData(bytes32 key, bytes data) returns()
func (_IConfig *IConfigTransactorSession) SetConfigWithData(key [32]byte, data []byte) (*types.Transaction, error) {
	return _IConfig.Contract.SetConfigWithData(&_IConfig.TransactOpts, key, data)
}

// IConfigConfigSetIterator is returned from FilterConfigSet and is used to iterate over the raw logs and unpacked data for ConfigSet events raised by the IConfig contract.
type IConfigConfigSetIterator struct {
	Event *IConfigConfigSet // Event containing the contract specifics and raw log

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
func (it *IConfigConfigSetIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(IConfigConfigSet)
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
		it.Event = new(IConfigConfigSet)
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
func (it *IConfigConfigSetIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *IConfigConfigSetIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// IConfigConfigSet represents a ConfigSet event raised by the IConfig contract.
type IConfigConfigSet struct {
	Key      [32]byte
	Writer   common.Address
	Checksum [32]byte
	Epoch    uint64
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterConfigSet is a free log retrieval operation binding the contract event 0x952e9f054f8f14436d21495120b5658808398d89e8240fca0e0b5669e4dbb013.
//
// Solidity: event ConfigSet(bytes32 indexed key, address indexed writer, bytes32 checksum, uint64 epoch)
func (_IConfig *IConfigFilterer) FilterConfigSet(opts *bind.FilterOpts, key [][32]byte, writer []common.Address) (*IConfigConfigSetIterator, error) {

	var keyRule []interface{}
	for _, keyItem := range key {
		keyRule = append(keyRule, keyItem)
	}
	var writerRule []interface{}
	for _, writerItem := range writer {
		writerRule = append(writerRule, writerItem)
	}

	logs, sub, err := _IConfig.contract.FilterLogs(opts, "ConfigSet", keyRule, writerRule)
	if err != nil {
		return nil, err
	}
	return &IConfigConfigSetIterator{contract: _IConfig.contract, event: "ConfigSet", logs: logs, sub: sub}, nil
}

// WatchConfigSet is a free log subscription operation binding the contract event 0x952e9f054f8f14436d21495120b5658808398d89e8240fca0e0b5669e4dbb013.
//
// Solidity: event ConfigSet(bytes32 indexed key, address indexed writer, bytes32 checksum, uint64 epoch)
func (_IConfig *IConfigFilterer) WatchConfigSet(opts *bind.WatchOpts, sink chan<- *IConfigConfigSet, key [][32]byte, writer []common.Address) (event.Subscription, error) {

	var keyRule []interface{}
	for _, keyItem := range key {
		keyRule = append(keyRule, keyItem)
	}
	var writerRule []interface{}
	for _, writerItem := range writer {
		writerRule = append(writerRule, writerItem)
	}

	logs, sub, err := _IConfig.contract.WatchLogs(opts, "ConfigSet", keyRule, writerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(IConfigConfigSet)
				if err := _IConfig.contract.UnpackLog(event, "ConfigSet", log); err != nil {
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
func (_IConfig *IConfigFilterer) ParseConfigSet(log types.Log) (*IConfigConfigSet, error) {
	event := new(IConfigConfigSet)
	if err := _IConfig.contract.UnpackLog(event, "ConfigSet", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// IConfigConfigSetWithDataIterator is returned from FilterConfigSetWithData and is used to iterate over the raw logs and unpacked data for ConfigSetWithData events raised by the IConfig contract.
type IConfigConfigSetWithDataIterator struct {
	Event *IConfigConfigSetWithData // Event containing the contract specifics and raw log

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
func (it *IConfigConfigSetWithDataIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(IConfigConfigSetWithData)
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
		it.Event = new(IConfigConfigSetWithData)
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
func (it *IConfigConfigSetWithDataIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *IConfigConfigSetWithDataIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// IConfigConfigSetWithData represents a ConfigSetWithData event raised by the IConfig contract.
type IConfigConfigSetWithData struct {
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
func (_IConfig *IConfigFilterer) FilterConfigSetWithData(opts *bind.FilterOpts, key [][32]byte, writer []common.Address) (*IConfigConfigSetWithDataIterator, error) {

	var keyRule []interface{}
	for _, keyItem := range key {
		keyRule = append(keyRule, keyItem)
	}
	var writerRule []interface{}
	for _, writerItem := range writer {
		writerRule = append(writerRule, writerItem)
	}

	logs, sub, err := _IConfig.contract.FilterLogs(opts, "ConfigSetWithData", keyRule, writerRule)
	if err != nil {
		return nil, err
	}
	return &IConfigConfigSetWithDataIterator{contract: _IConfig.contract, event: "ConfigSetWithData", logs: logs, sub: sub}, nil
}

// WatchConfigSetWithData is a free log subscription operation binding the contract event 0x138bbba807de4352fcbdebae920532cde6774dba6fd049b88ea55bc3af412c00.
//
// Solidity: event ConfigSetWithData(bytes32 indexed key, address indexed writer, bytes32 checksum, uint64 epoch, bytes data)
func (_IConfig *IConfigFilterer) WatchConfigSetWithData(opts *bind.WatchOpts, sink chan<- *IConfigConfigSetWithData, key [][32]byte, writer []common.Address) (event.Subscription, error) {

	var keyRule []interface{}
	for _, keyItem := range key {
		keyRule = append(keyRule, keyItem)
	}
	var writerRule []interface{}
	for _, writerItem := range writer {
		writerRule = append(writerRule, writerItem)
	}

	logs, sub, err := _IConfig.contract.WatchLogs(opts, "ConfigSetWithData", keyRule, writerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(IConfigConfigSetWithData)
				if err := _IConfig.contract.UnpackLog(event, "ConfigSetWithData", log); err != nil {
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
func (_IConfig *IConfigFilterer) ParseConfigSetWithData(log types.Log) (*IConfigConfigSetWithData, error) {
	event := new(IConfigConfigSetWithData)
	if err := _IConfig.contract.UnpackLog(event, "ConfigSetWithData", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

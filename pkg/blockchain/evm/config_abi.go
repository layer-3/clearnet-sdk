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
	ABI: "[{\"type\":\"constructor\",\"inputs\":[{\"name\":\"owner_\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"acceptOwnership\",\"inputs\":[],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"configChecksumAt\",\"inputs\":[{\"name\":\"key\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"index\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"configChecksums\",\"inputs\":[{\"name\":\"key\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32[]\",\"internalType\":\"bytes32[]\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"configEpoch\",\"inputs\":[{\"name\":\"key\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint64\",\"internalType\":\"uint64\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"configWriter\",\"inputs\":[{\"name\":\"key\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"latestConfigChecksum\",\"inputs\":[{\"name\":\"key\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"owner\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"pendingOwner\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"renounceOwnership\",\"inputs\":[],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"setConfig\",\"inputs\":[{\"name\":\"key\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"checksum\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"setConfigWriter\",\"inputs\":[{\"name\":\"key\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"writer\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"transferOwnership\",\"inputs\":[{\"name\":\"newOwner\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"event\",\"name\":\"ConfigSet\",\"inputs\":[{\"name\":\"key\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"},{\"name\":\"writer\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"checksum\",\"type\":\"bytes32\",\"indexed\":false,\"internalType\":\"bytes32\"},{\"name\":\"epoch\",\"type\":\"uint64\",\"indexed\":false,\"internalType\":\"uint64\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"ConfigWriterSet\",\"inputs\":[{\"name\":\"key\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"},{\"name\":\"previousWriter\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"newWriter\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"OwnershipTransferStarted\",\"inputs\":[{\"name\":\"previousOwner\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"newOwner\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"OwnershipTransferred\",\"inputs\":[{\"name\":\"previousOwner\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"newOwner\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"}],\"anonymous\":false},{\"type\":\"error\",\"name\":\"EmptyConfigKey\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"EpochOutOfRange\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"NotConfigWriter\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"OwnableInvalidOwner\",\"inputs\":[{\"name\":\"owner\",\"type\":\"address\",\"internalType\":\"address\"}]},{\"type\":\"error\",\"name\":\"OwnableUnauthorizedAccount\",\"inputs\":[{\"name\":\"account\",\"type\":\"address\",\"internalType\":\"address\"}]}]",
	Bin: "0x60803460c157601f61070838819003918201601f19168301916001600160401b0383118484101760c55780849260209460405283398101031260c157516001600160a01b0381169081900360c157801560ae57600180546001600160a01b03199081169091555f80549182168317815560405192916001600160a01b0316907f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e09080a361062e90816100da8239f35b631e4fbdf760e01b5f525f60045260245ffd5b5f80fd5b634e487b7160e01b5f52604160045260245ffdfe60806040526004361015610011575f80fd5b5f3560e01c80632676450c146105925780633632ad79146105605780633cb37e57146104e6578063715018a61461048357806379ba5097146103fe5780638da5cb5b146103d7578063af890358146103a3578063d1fd27b3146102d2578063e30c3978146102aa578063f2fde38b14610237578063fec5bedb146101675763fee579951461009d575f80fd5b34610163576040366003190112610163576024356001600160a01b0381169060043590829003610163578015610154575f8181526003602052604081205490546001600160a01b039182169291163314801561014b575b1561013c575f81815260036020526040812080546001600160a01b031916851790557f46f01f011115ff538e9d22228830993d46f2ed2657987b0840069c92f3ae163e9080a4005b637187556f60e11b5f5260045ffd5b508133146100f4565b6355a9397560e11b5f5260045ffd5b5f80fd5b34610163576020366003190112610163576004355f52600260205260405f2060405190816020825491828152019081925f5260205f20905f5b81811061022157505050829003601f01601f191682019167ffffffffffffffff83118184101761020d5790829182604052602083019060208452518091526040830191905f5b8181106101f4575050500390f35b82518452859450602093840193909201916001016101e6565b634e487b7160e01b5f52604160045260245ffd5b82548452602090930192600192830192016101a0565b34610163576020366003190112610163576004356001600160a01b038116908190036101635761026561061b565b600180546001600160a01b031916821790555f80546001600160a01b0316907f38d16b8cac22d99fc7c124b9cd0de2d3fa1faef420bfe791d8c362d765e227009080a3005b34610163575f366003190112610163576001546040516001600160a01b039091168152602090f35b34610163576040366003190112610163576004355f818152600360205260409020546001600160a01b0316906024353315158061039a575b1561013c57815f52600260205260405f20908154906801000000000000000082101561020d57610363827f952e9f054f8f14436d21495120b5658808398d89e8240fca0e0b5669e4dbb0139460016040950181556105f2565b81549060031b9083821b915f19901b1916179055835f52600260205267ffffffffffffffff825f20541682519182526020820152a3005b5082331461030a565b34610163576020366003190112610163576004355f526002602052602067ffffffffffffffff60405f205416604051908152f35b34610163575f366003190112610163575f546040516001600160a01b039091168152602090f35b34610163575f36600319011261016357600154336001600160a01b039091160361047057600180546001600160a01b03199081169091555f805433928116831782556001600160a01b0316907f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e09080a3005b63118cdaa760e01b5f523360045260245ffd5b34610163575f3660031901126101635761049b61061b565b600180546001600160a01b03199081169091555f80549182168155906001600160a01b03167f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e08280a3005b3461016357602036600319011261016357600435805f52600260205260405f20549081155f1461051e57505060205f5b604051908152f35b5f52600260205260405f205f19820191821161054c57602091610540916105f2565b90549060031b1c610516565b634e487b7160e01b5f52601160045260245ffd5b34610163576020366003190112610163576004355f526003602052602060018060a01b0360405f205416604051908152f35b346101635760403660031901126101635760043560243590805f52600260205260405f20548210156105e3576020916105d4915f526002835260405f206105f2565b90549060031b1c604051908152f35b6316f4c85360e01b5f5260045ffd5b8054821015610607575f5260205f2001905f90565b634e487b7160e01b5f52603260045260245ffd5b5f546001600160a01b031633036104705756",
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

// ConfigChecksumAt is a free data retrieval call binding the contract method 0x2676450c.
//
// Solidity: function configChecksumAt(bytes32 key, uint256 index) view returns(bytes32)
func (_Config *ConfigCaller) ConfigChecksumAt(opts *bind.CallOpts, key [32]byte, index *big.Int) ([32]byte, error) {
	var out []interface{}
	err := _Config.contract.Call(opts, &out, "configChecksumAt", key, index)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// ConfigChecksumAt is a free data retrieval call binding the contract method 0x2676450c.
//
// Solidity: function configChecksumAt(bytes32 key, uint256 index) view returns(bytes32)
func (_Config *ConfigSession) ConfigChecksumAt(key [32]byte, index *big.Int) ([32]byte, error) {
	return _Config.Contract.ConfigChecksumAt(&_Config.CallOpts, key, index)
}

// ConfigChecksumAt is a free data retrieval call binding the contract method 0x2676450c.
//
// Solidity: function configChecksumAt(bytes32 key, uint256 index) view returns(bytes32)
func (_Config *ConfigCallerSession) ConfigChecksumAt(key [32]byte, index *big.Int) ([32]byte, error) {
	return _Config.Contract.ConfigChecksumAt(&_Config.CallOpts, key, index)
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

// ConfigWriter is a free data retrieval call binding the contract method 0x3632ad79.
//
// Solidity: function configWriter(bytes32 key) view returns(address)
func (_Config *ConfigCaller) ConfigWriter(opts *bind.CallOpts, key [32]byte) (common.Address, error) {
	var out []interface{}
	err := _Config.contract.Call(opts, &out, "configWriter", key)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// ConfigWriter is a free data retrieval call binding the contract method 0x3632ad79.
//
// Solidity: function configWriter(bytes32 key) view returns(address)
func (_Config *ConfigSession) ConfigWriter(key [32]byte) (common.Address, error) {
	return _Config.Contract.ConfigWriter(&_Config.CallOpts, key)
}

// ConfigWriter is a free data retrieval call binding the contract method 0x3632ad79.
//
// Solidity: function configWriter(bytes32 key) view returns(address)
func (_Config *ConfigCallerSession) ConfigWriter(key [32]byte) (common.Address, error) {
	return _Config.Contract.ConfigWriter(&_Config.CallOpts, key)
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

// PendingOwner is a free data retrieval call binding the contract method 0xe30c3978.
//
// Solidity: function pendingOwner() view returns(address)
func (_Config *ConfigCaller) PendingOwner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Config.contract.Call(opts, &out, "pendingOwner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// PendingOwner is a free data retrieval call binding the contract method 0xe30c3978.
//
// Solidity: function pendingOwner() view returns(address)
func (_Config *ConfigSession) PendingOwner() (common.Address, error) {
	return _Config.Contract.PendingOwner(&_Config.CallOpts)
}

// PendingOwner is a free data retrieval call binding the contract method 0xe30c3978.
//
// Solidity: function pendingOwner() view returns(address)
func (_Config *ConfigCallerSession) PendingOwner() (common.Address, error) {
	return _Config.Contract.PendingOwner(&_Config.CallOpts)
}

// AcceptOwnership is a paid mutator transaction binding the contract method 0x79ba5097.
//
// Solidity: function acceptOwnership() returns()
func (_Config *ConfigTransactor) AcceptOwnership(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Config.contract.Transact(opts, "acceptOwnership")
}

// AcceptOwnership is a paid mutator transaction binding the contract method 0x79ba5097.
//
// Solidity: function acceptOwnership() returns()
func (_Config *ConfigSession) AcceptOwnership() (*types.Transaction, error) {
	return _Config.Contract.AcceptOwnership(&_Config.TransactOpts)
}

// AcceptOwnership is a paid mutator transaction binding the contract method 0x79ba5097.
//
// Solidity: function acceptOwnership() returns()
func (_Config *ConfigTransactorSession) AcceptOwnership() (*types.Transaction, error) {
	return _Config.Contract.AcceptOwnership(&_Config.TransactOpts)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_Config *ConfigTransactor) RenounceOwnership(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Config.contract.Transact(opts, "renounceOwnership")
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_Config *ConfigSession) RenounceOwnership() (*types.Transaction, error) {
	return _Config.Contract.RenounceOwnership(&_Config.TransactOpts)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_Config *ConfigTransactorSession) RenounceOwnership() (*types.Transaction, error) {
	return _Config.Contract.RenounceOwnership(&_Config.TransactOpts)
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

// SetConfigWriter is a paid mutator transaction binding the contract method 0xfee57995.
//
// Solidity: function setConfigWriter(bytes32 key, address writer) returns()
func (_Config *ConfigTransactor) SetConfigWriter(opts *bind.TransactOpts, key [32]byte, writer common.Address) (*types.Transaction, error) {
	return _Config.contract.Transact(opts, "setConfigWriter", key, writer)
}

// SetConfigWriter is a paid mutator transaction binding the contract method 0xfee57995.
//
// Solidity: function setConfigWriter(bytes32 key, address writer) returns()
func (_Config *ConfigSession) SetConfigWriter(key [32]byte, writer common.Address) (*types.Transaction, error) {
	return _Config.Contract.SetConfigWriter(&_Config.TransactOpts, key, writer)
}

// SetConfigWriter is a paid mutator transaction binding the contract method 0xfee57995.
//
// Solidity: function setConfigWriter(bytes32 key, address writer) returns()
func (_Config *ConfigTransactorSession) SetConfigWriter(key [32]byte, writer common.Address) (*types.Transaction, error) {
	return _Config.Contract.SetConfigWriter(&_Config.TransactOpts, key, writer)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_Config *ConfigTransactor) TransferOwnership(opts *bind.TransactOpts, newOwner common.Address) (*types.Transaction, error) {
	return _Config.contract.Transact(opts, "transferOwnership", newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_Config *ConfigSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _Config.Contract.TransferOwnership(&_Config.TransactOpts, newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_Config *ConfigTransactorSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _Config.Contract.TransferOwnership(&_Config.TransactOpts, newOwner)
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

// ConfigConfigWriterSetIterator is returned from FilterConfigWriterSet and is used to iterate over the raw logs and unpacked data for ConfigWriterSet events raised by the Config contract.
type ConfigConfigWriterSetIterator struct {
	Event *ConfigConfigWriterSet // Event containing the contract specifics and raw log

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
func (it *ConfigConfigWriterSetIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ConfigConfigWriterSet)
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
		it.Event = new(ConfigConfigWriterSet)
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
func (it *ConfigConfigWriterSetIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ConfigConfigWriterSetIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ConfigConfigWriterSet represents a ConfigWriterSet event raised by the Config contract.
type ConfigConfigWriterSet struct {
	Key            [32]byte
	PreviousWriter common.Address
	NewWriter      common.Address
	Raw            types.Log // Blockchain specific contextual infos
}

// FilterConfigWriterSet is a free log retrieval operation binding the contract event 0x46f01f011115ff538e9d22228830993d46f2ed2657987b0840069c92f3ae163e.
//
// Solidity: event ConfigWriterSet(bytes32 indexed key, address indexed previousWriter, address indexed newWriter)
func (_Config *ConfigFilterer) FilterConfigWriterSet(opts *bind.FilterOpts, key [][32]byte, previousWriter []common.Address, newWriter []common.Address) (*ConfigConfigWriterSetIterator, error) {

	var keyRule []interface{}
	for _, keyItem := range key {
		keyRule = append(keyRule, keyItem)
	}
	var previousWriterRule []interface{}
	for _, previousWriterItem := range previousWriter {
		previousWriterRule = append(previousWriterRule, previousWriterItem)
	}
	var newWriterRule []interface{}
	for _, newWriterItem := range newWriter {
		newWriterRule = append(newWriterRule, newWriterItem)
	}

	logs, sub, err := _Config.contract.FilterLogs(opts, "ConfigWriterSet", keyRule, previousWriterRule, newWriterRule)
	if err != nil {
		return nil, err
	}
	return &ConfigConfigWriterSetIterator{contract: _Config.contract, event: "ConfigWriterSet", logs: logs, sub: sub}, nil
}

// WatchConfigWriterSet is a free log subscription operation binding the contract event 0x46f01f011115ff538e9d22228830993d46f2ed2657987b0840069c92f3ae163e.
//
// Solidity: event ConfigWriterSet(bytes32 indexed key, address indexed previousWriter, address indexed newWriter)
func (_Config *ConfigFilterer) WatchConfigWriterSet(opts *bind.WatchOpts, sink chan<- *ConfigConfigWriterSet, key [][32]byte, previousWriter []common.Address, newWriter []common.Address) (event.Subscription, error) {

	var keyRule []interface{}
	for _, keyItem := range key {
		keyRule = append(keyRule, keyItem)
	}
	var previousWriterRule []interface{}
	for _, previousWriterItem := range previousWriter {
		previousWriterRule = append(previousWriterRule, previousWriterItem)
	}
	var newWriterRule []interface{}
	for _, newWriterItem := range newWriter {
		newWriterRule = append(newWriterRule, newWriterItem)
	}

	logs, sub, err := _Config.contract.WatchLogs(opts, "ConfigWriterSet", keyRule, previousWriterRule, newWriterRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ConfigConfigWriterSet)
				if err := _Config.contract.UnpackLog(event, "ConfigWriterSet", log); err != nil {
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

// ParseConfigWriterSet is a log parse operation binding the contract event 0x46f01f011115ff538e9d22228830993d46f2ed2657987b0840069c92f3ae163e.
//
// Solidity: event ConfigWriterSet(bytes32 indexed key, address indexed previousWriter, address indexed newWriter)
func (_Config *ConfigFilterer) ParseConfigWriterSet(log types.Log) (*ConfigConfigWriterSet, error) {
	event := new(ConfigConfigWriterSet)
	if err := _Config.contract.UnpackLog(event, "ConfigWriterSet", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ConfigOwnershipTransferStartedIterator is returned from FilterOwnershipTransferStarted and is used to iterate over the raw logs and unpacked data for OwnershipTransferStarted events raised by the Config contract.
type ConfigOwnershipTransferStartedIterator struct {
	Event *ConfigOwnershipTransferStarted // Event containing the contract specifics and raw log

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
func (it *ConfigOwnershipTransferStartedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ConfigOwnershipTransferStarted)
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
		it.Event = new(ConfigOwnershipTransferStarted)
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
func (it *ConfigOwnershipTransferStartedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ConfigOwnershipTransferStartedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ConfigOwnershipTransferStarted represents a OwnershipTransferStarted event raised by the Config contract.
type ConfigOwnershipTransferStarted struct {
	PreviousOwner common.Address
	NewOwner      common.Address
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterOwnershipTransferStarted is a free log retrieval operation binding the contract event 0x38d16b8cac22d99fc7c124b9cd0de2d3fa1faef420bfe791d8c362d765e22700.
//
// Solidity: event OwnershipTransferStarted(address indexed previousOwner, address indexed newOwner)
func (_Config *ConfigFilterer) FilterOwnershipTransferStarted(opts *bind.FilterOpts, previousOwner []common.Address, newOwner []common.Address) (*ConfigOwnershipTransferStartedIterator, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _Config.contract.FilterLogs(opts, "OwnershipTransferStarted", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return &ConfigOwnershipTransferStartedIterator{contract: _Config.contract, event: "OwnershipTransferStarted", logs: logs, sub: sub}, nil
}

// WatchOwnershipTransferStarted is a free log subscription operation binding the contract event 0x38d16b8cac22d99fc7c124b9cd0de2d3fa1faef420bfe791d8c362d765e22700.
//
// Solidity: event OwnershipTransferStarted(address indexed previousOwner, address indexed newOwner)
func (_Config *ConfigFilterer) WatchOwnershipTransferStarted(opts *bind.WatchOpts, sink chan<- *ConfigOwnershipTransferStarted, previousOwner []common.Address, newOwner []common.Address) (event.Subscription, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _Config.contract.WatchLogs(opts, "OwnershipTransferStarted", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ConfigOwnershipTransferStarted)
				if err := _Config.contract.UnpackLog(event, "OwnershipTransferStarted", log); err != nil {
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

// ParseOwnershipTransferStarted is a log parse operation binding the contract event 0x38d16b8cac22d99fc7c124b9cd0de2d3fa1faef420bfe791d8c362d765e22700.
//
// Solidity: event OwnershipTransferStarted(address indexed previousOwner, address indexed newOwner)
func (_Config *ConfigFilterer) ParseOwnershipTransferStarted(log types.Log) (*ConfigOwnershipTransferStarted, error) {
	event := new(ConfigOwnershipTransferStarted)
	if err := _Config.contract.UnpackLog(event, "OwnershipTransferStarted", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ConfigOwnershipTransferredIterator is returned from FilterOwnershipTransferred and is used to iterate over the raw logs and unpacked data for OwnershipTransferred events raised by the Config contract.
type ConfigOwnershipTransferredIterator struct {
	Event *ConfigOwnershipTransferred // Event containing the contract specifics and raw log

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
func (it *ConfigOwnershipTransferredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ConfigOwnershipTransferred)
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
		it.Event = new(ConfigOwnershipTransferred)
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
func (it *ConfigOwnershipTransferredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ConfigOwnershipTransferredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ConfigOwnershipTransferred represents a OwnershipTransferred event raised by the Config contract.
type ConfigOwnershipTransferred struct {
	PreviousOwner common.Address
	NewOwner      common.Address
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterOwnershipTransferred is a free log retrieval operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_Config *ConfigFilterer) FilterOwnershipTransferred(opts *bind.FilterOpts, previousOwner []common.Address, newOwner []common.Address) (*ConfigOwnershipTransferredIterator, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _Config.contract.FilterLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return &ConfigOwnershipTransferredIterator{contract: _Config.contract, event: "OwnershipTransferred", logs: logs, sub: sub}, nil
}

// WatchOwnershipTransferred is a free log subscription operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_Config *ConfigFilterer) WatchOwnershipTransferred(opts *bind.WatchOpts, sink chan<- *ConfigOwnershipTransferred, previousOwner []common.Address, newOwner []common.Address) (event.Subscription, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _Config.contract.WatchLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ConfigOwnershipTransferred)
				if err := _Config.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
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

// ParseOwnershipTransferred is a log parse operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_Config *ConfigFilterer) ParseOwnershipTransferred(log types.Log) (*ConfigOwnershipTransferred, error) {
	event := new(ConfigOwnershipTransferred)
	if err := _Config.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

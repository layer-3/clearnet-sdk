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

// NodeRecord is an auto generated low-level Go binding around an user-defined struct.
type NodeRecord struct {
	Operator           common.Address
	ActivatedAt        uint64
	DeactivatedAt      uint64
	VestedAt           uint64
	Index              uint32
	TokenId            uint32
	OperatorCollateral *big.Int
	SponsorCollateral  *big.Int
	BlsPubkeyG1        [2]*big.Int
	BlsPubkeyG2        [4]*big.Int
}

// ClearnetRegistryMetaData contains all meta data concerning the ClearnetRegistry contract.
var ClearnetRegistryMetaData = &bind.MetaData{
	ABI: "[{\"type\":\"function\",\"name\":\"getNodeById\",\"inputs\":[{\"name\":\"nodeId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[{\"name\":\"\",\"type\":\"tuple\",\"internalType\":\"structNodeRecord\",\"components\":[{\"name\":\"operator\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"activatedAt\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"deactivatedAt\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"vestedAt\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"index\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"tokenId\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"operatorCollateral\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"sponsorCollateral\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"blsPubkeyG1\",\"type\":\"uint256[2]\",\"internalType\":\"uint256[2]\"},{\"name\":\"blsPubkeyG2\",\"type\":\"uint256[4]\",\"internalType\":\"uint256[4]\"}]}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getNodeIds\",\"inputs\":[{\"name\":\"offset\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"limit\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32[]\",\"internalType\":\"bytes32[]\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getNodes\",\"inputs\":[{\"name\":\"offset\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"limit\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"tuple[]\",\"internalType\":\"structNodeRecord[]\",\"components\":[{\"name\":\"operator\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"activatedAt\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"deactivatedAt\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"vestedAt\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"index\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"tokenId\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"operatorCollateral\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"sponsorCollateral\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"blsPubkeyG1\",\"type\":\"uint256[2]\",\"internalType\":\"uint256[2]\"},{\"name\":\"blsPubkeyG2\",\"type\":\"uint256[4]\",\"internalType\":\"uint256[4]\"}]}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"totalNodes\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"event\",\"name\":\"NodeActivated\",\"inputs\":[{\"name\":\"operator\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"nodeId\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"},{\"name\":\"tokenId\",\"type\":\"uint32\",\"indexed\":true,\"internalType\":\"uint32\"},{\"name\":\"collateral\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"},{\"name\":\"vestedAt\",\"type\":\"uint64\",\"indexed\":false,\"internalType\":\"uint64\"},{\"name\":\"blsPubkeyG2\",\"type\":\"uint256[4]\",\"indexed\":false,\"internalType\":\"uint256[4]\"}],\"anonymous\":false}]",
}

// ClearnetRegistryABI is the input ABI used to generate the binding from.
// Deprecated: Use ClearnetRegistryMetaData.ABI instead.
var ClearnetRegistryABI = ClearnetRegistryMetaData.ABI

// ClearnetRegistry is an auto generated Go binding around an Ethereum contract.
type ClearnetRegistry struct {
	ClearnetRegistryCaller     // Read-only binding to the contract
	ClearnetRegistryTransactor // Write-only binding to the contract
	ClearnetRegistryFilterer   // Log filterer for contract events
}

// ClearnetRegistryCaller is an auto generated read-only Go binding around an Ethereum contract.
type ClearnetRegistryCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ClearnetRegistryTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ClearnetRegistryTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ClearnetRegistryFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type ClearnetRegistryFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ClearnetRegistrySession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ClearnetRegistrySession struct {
	Contract     *ClearnetRegistry // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// ClearnetRegistryCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ClearnetRegistryCallerSession struct {
	Contract *ClearnetRegistryCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts           // Call options to use throughout this session
}

// ClearnetRegistryTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ClearnetRegistryTransactorSession struct {
	Contract     *ClearnetRegistryTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts           // Transaction auth options to use throughout this session
}

// ClearnetRegistryRaw is an auto generated low-level Go binding around an Ethereum contract.
type ClearnetRegistryRaw struct {
	Contract *ClearnetRegistry // Generic contract binding to access the raw methods on
}

// ClearnetRegistryCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ClearnetRegistryCallerRaw struct {
	Contract *ClearnetRegistryCaller // Generic read-only contract binding to access the raw methods on
}

// ClearnetRegistryTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ClearnetRegistryTransactorRaw struct {
	Contract *ClearnetRegistryTransactor // Generic write-only contract binding to access the raw methods on
}

// NewClearnetRegistry creates a new instance of ClearnetRegistry, bound to a specific deployed contract.
func NewClearnetRegistry(address common.Address, backend bind.ContractBackend) (*ClearnetRegistry, error) {
	contract, err := bindClearnetRegistry(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &ClearnetRegistry{ClearnetRegistryCaller: ClearnetRegistryCaller{contract: contract}, ClearnetRegistryTransactor: ClearnetRegistryTransactor{contract: contract}, ClearnetRegistryFilterer: ClearnetRegistryFilterer{contract: contract}}, nil
}

// NewClearnetRegistryCaller creates a new read-only instance of ClearnetRegistry, bound to a specific deployed contract.
func NewClearnetRegistryCaller(address common.Address, caller bind.ContractCaller) (*ClearnetRegistryCaller, error) {
	contract, err := bindClearnetRegistry(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ClearnetRegistryCaller{contract: contract}, nil
}

// NewClearnetRegistryTransactor creates a new write-only instance of ClearnetRegistry, bound to a specific deployed contract.
func NewClearnetRegistryTransactor(address common.Address, transactor bind.ContractTransactor) (*ClearnetRegistryTransactor, error) {
	contract, err := bindClearnetRegistry(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &ClearnetRegistryTransactor{contract: contract}, nil
}

// NewClearnetRegistryFilterer creates a new log filterer instance of ClearnetRegistry, bound to a specific deployed contract.
func NewClearnetRegistryFilterer(address common.Address, filterer bind.ContractFilterer) (*ClearnetRegistryFilterer, error) {
	contract, err := bindClearnetRegistry(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &ClearnetRegistryFilterer{contract: contract}, nil
}

// bindClearnetRegistry binds a generic wrapper to an already deployed contract.
func bindClearnetRegistry(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := ClearnetRegistryMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ClearnetRegistry *ClearnetRegistryRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ClearnetRegistry.Contract.ClearnetRegistryCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ClearnetRegistry *ClearnetRegistryRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ClearnetRegistry.Contract.ClearnetRegistryTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ClearnetRegistry *ClearnetRegistryRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ClearnetRegistry.Contract.ClearnetRegistryTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ClearnetRegistry *ClearnetRegistryCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ClearnetRegistry.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ClearnetRegistry *ClearnetRegistryTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ClearnetRegistry.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ClearnetRegistry *ClearnetRegistryTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ClearnetRegistry.Contract.contract.Transact(opts, method, params...)
}

// GetNodeById is a free data retrieval call binding the contract method 0x8899cf50.
//
// Solidity: function getNodeById(bytes32 nodeId) view returns((address,uint64,uint64,uint64,uint32,uint32,uint256,uint256,uint256[2],uint256[4]))
func (_ClearnetRegistry *ClearnetRegistryCaller) GetNodeById(opts *bind.CallOpts, nodeId [32]byte) (NodeRecord, error) {
	var out []interface{}
	err := _ClearnetRegistry.contract.Call(opts, &out, "getNodeById", nodeId)

	if err != nil {
		return *new(NodeRecord), err
	}

	out0 := *abi.ConvertType(out[0], new(NodeRecord)).(*NodeRecord)

	return out0, err

}

// GetNodeById is a free data retrieval call binding the contract method 0x8899cf50.
//
// Solidity: function getNodeById(bytes32 nodeId) view returns((address,uint64,uint64,uint64,uint32,uint32,uint256,uint256,uint256[2],uint256[4]))
func (_ClearnetRegistry *ClearnetRegistrySession) GetNodeById(nodeId [32]byte) (NodeRecord, error) {
	return _ClearnetRegistry.Contract.GetNodeById(&_ClearnetRegistry.CallOpts, nodeId)
}

// GetNodeById is a free data retrieval call binding the contract method 0x8899cf50.
//
// Solidity: function getNodeById(bytes32 nodeId) view returns((address,uint64,uint64,uint64,uint32,uint32,uint256,uint256,uint256[2],uint256[4]))
func (_ClearnetRegistry *ClearnetRegistryCallerSession) GetNodeById(nodeId [32]byte) (NodeRecord, error) {
	return _ClearnetRegistry.Contract.GetNodeById(&_ClearnetRegistry.CallOpts, nodeId)
}

// GetNodeIds is a free data retrieval call binding the contract method 0xbdc43e92.
//
// Solidity: function getNodeIds(uint256 offset, uint256 limit) view returns(bytes32[])
func (_ClearnetRegistry *ClearnetRegistryCaller) GetNodeIds(opts *bind.CallOpts, offset *big.Int, limit *big.Int) ([][32]byte, error) {
	var out []interface{}
	err := _ClearnetRegistry.contract.Call(opts, &out, "getNodeIds", offset, limit)

	if err != nil {
		return *new([][32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([][32]byte)).(*[][32]byte)

	return out0, err

}

// GetNodeIds is a free data retrieval call binding the contract method 0xbdc43e92.
//
// Solidity: function getNodeIds(uint256 offset, uint256 limit) view returns(bytes32[])
func (_ClearnetRegistry *ClearnetRegistrySession) GetNodeIds(offset *big.Int, limit *big.Int) ([][32]byte, error) {
	return _ClearnetRegistry.Contract.GetNodeIds(&_ClearnetRegistry.CallOpts, offset, limit)
}

// GetNodeIds is a free data retrieval call binding the contract method 0xbdc43e92.
//
// Solidity: function getNodeIds(uint256 offset, uint256 limit) view returns(bytes32[])
func (_ClearnetRegistry *ClearnetRegistryCallerSession) GetNodeIds(offset *big.Int, limit *big.Int) ([][32]byte, error) {
	return _ClearnetRegistry.Contract.GetNodeIds(&_ClearnetRegistry.CallOpts, offset, limit)
}

// GetNodes is a free data retrieval call binding the contract method 0x038d67e8.
//
// Solidity: function getNodes(uint256 offset, uint256 limit) view returns((address,uint64,uint64,uint64,uint32,uint32,uint256,uint256,uint256[2],uint256[4])[])
func (_ClearnetRegistry *ClearnetRegistryCaller) GetNodes(opts *bind.CallOpts, offset *big.Int, limit *big.Int) ([]NodeRecord, error) {
	var out []interface{}
	err := _ClearnetRegistry.contract.Call(opts, &out, "getNodes", offset, limit)

	if err != nil {
		return *new([]NodeRecord), err
	}

	out0 := *abi.ConvertType(out[0], new([]NodeRecord)).(*[]NodeRecord)

	return out0, err

}

// GetNodes is a free data retrieval call binding the contract method 0x038d67e8.
//
// Solidity: function getNodes(uint256 offset, uint256 limit) view returns((address,uint64,uint64,uint64,uint32,uint32,uint256,uint256,uint256[2],uint256[4])[])
func (_ClearnetRegistry *ClearnetRegistrySession) GetNodes(offset *big.Int, limit *big.Int) ([]NodeRecord, error) {
	return _ClearnetRegistry.Contract.GetNodes(&_ClearnetRegistry.CallOpts, offset, limit)
}

// GetNodes is a free data retrieval call binding the contract method 0x038d67e8.
//
// Solidity: function getNodes(uint256 offset, uint256 limit) view returns((address,uint64,uint64,uint64,uint32,uint32,uint256,uint256,uint256[2],uint256[4])[])
func (_ClearnetRegistry *ClearnetRegistryCallerSession) GetNodes(offset *big.Int, limit *big.Int) ([]NodeRecord, error) {
	return _ClearnetRegistry.Contract.GetNodes(&_ClearnetRegistry.CallOpts, offset, limit)
}

// TotalNodes is a free data retrieval call binding the contract method 0x9592d424.
//
// Solidity: function totalNodes() view returns(uint256)
func (_ClearnetRegistry *ClearnetRegistryCaller) TotalNodes(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _ClearnetRegistry.contract.Call(opts, &out, "totalNodes")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TotalNodes is a free data retrieval call binding the contract method 0x9592d424.
//
// Solidity: function totalNodes() view returns(uint256)
func (_ClearnetRegistry *ClearnetRegistrySession) TotalNodes() (*big.Int, error) {
	return _ClearnetRegistry.Contract.TotalNodes(&_ClearnetRegistry.CallOpts)
}

// TotalNodes is a free data retrieval call binding the contract method 0x9592d424.
//
// Solidity: function totalNodes() view returns(uint256)
func (_ClearnetRegistry *ClearnetRegistryCallerSession) TotalNodes() (*big.Int, error) {
	return _ClearnetRegistry.Contract.TotalNodes(&_ClearnetRegistry.CallOpts)
}

// ClearnetRegistryNodeActivatedIterator is returned from FilterNodeActivated and is used to iterate over the raw logs and unpacked data for NodeActivated events raised by the ClearnetRegistry contract.
type ClearnetRegistryNodeActivatedIterator struct {
	Event *ClearnetRegistryNodeActivated // Event containing the contract specifics and raw log

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
func (it *ClearnetRegistryNodeActivatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ClearnetRegistryNodeActivated)
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
		it.Event = new(ClearnetRegistryNodeActivated)
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
func (it *ClearnetRegistryNodeActivatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ClearnetRegistryNodeActivatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ClearnetRegistryNodeActivated represents a NodeActivated event raised by the ClearnetRegistry contract.
type ClearnetRegistryNodeActivated struct {
	Operator    common.Address
	NodeId      [32]byte
	TokenId     uint32
	Collateral  *big.Int
	VestedAt    uint64
	BlsPubkeyG2 [4]*big.Int
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterNodeActivated is a free log retrieval operation binding the contract event 0x58cee7261629c956b111bc684df727bd2e9b0f5d954e24b93908951c431cd13e.
//
// Solidity: event NodeActivated(address indexed operator, bytes32 indexed nodeId, uint32 indexed tokenId, uint256 collateral, uint64 vestedAt, uint256[4] blsPubkeyG2)
func (_ClearnetRegistry *ClearnetRegistryFilterer) FilterNodeActivated(opts *bind.FilterOpts, operator []common.Address, nodeId [][32]byte, tokenId []uint32) (*ClearnetRegistryNodeActivatedIterator, error) {

	var operatorRule []interface{}
	for _, operatorItem := range operator {
		operatorRule = append(operatorRule, operatorItem)
	}
	var nodeIdRule []interface{}
	for _, nodeIdItem := range nodeId {
		nodeIdRule = append(nodeIdRule, nodeIdItem)
	}
	var tokenIdRule []interface{}
	for _, tokenIdItem := range tokenId {
		tokenIdRule = append(tokenIdRule, tokenIdItem)
	}

	logs, sub, err := _ClearnetRegistry.contract.FilterLogs(opts, "NodeActivated", operatorRule, nodeIdRule, tokenIdRule)
	if err != nil {
		return nil, err
	}
	return &ClearnetRegistryNodeActivatedIterator{contract: _ClearnetRegistry.contract, event: "NodeActivated", logs: logs, sub: sub}, nil
}

// WatchNodeActivated is a free log subscription operation binding the contract event 0x58cee7261629c956b111bc684df727bd2e9b0f5d954e24b93908951c431cd13e.
//
// Solidity: event NodeActivated(address indexed operator, bytes32 indexed nodeId, uint32 indexed tokenId, uint256 collateral, uint64 vestedAt, uint256[4] blsPubkeyG2)
func (_ClearnetRegistry *ClearnetRegistryFilterer) WatchNodeActivated(opts *bind.WatchOpts, sink chan<- *ClearnetRegistryNodeActivated, operator []common.Address, nodeId [][32]byte, tokenId []uint32) (event.Subscription, error) {

	var operatorRule []interface{}
	for _, operatorItem := range operator {
		operatorRule = append(operatorRule, operatorItem)
	}
	var nodeIdRule []interface{}
	for _, nodeIdItem := range nodeId {
		nodeIdRule = append(nodeIdRule, nodeIdItem)
	}
	var tokenIdRule []interface{}
	for _, tokenIdItem := range tokenId {
		tokenIdRule = append(tokenIdRule, tokenIdItem)
	}

	logs, sub, err := _ClearnetRegistry.contract.WatchLogs(opts, "NodeActivated", operatorRule, nodeIdRule, tokenIdRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ClearnetRegistryNodeActivated)
				if err := _ClearnetRegistry.contract.UnpackLog(event, "NodeActivated", log); err != nil {
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

// ParseNodeActivated is a log parse operation binding the contract event 0x58cee7261629c956b111bc684df727bd2e9b0f5d954e24b93908951c431cd13e.
//
// Solidity: event NodeActivated(address indexed operator, bytes32 indexed nodeId, uint32 indexed tokenId, uint256 collateral, uint64 vestedAt, uint256[4] blsPubkeyG2)
func (_ClearnetRegistry *ClearnetRegistryFilterer) ParseNodeActivated(log types.Log) (*ClearnetRegistryNodeActivated, error) {
	event := new(ClearnetRegistryNodeActivated)
	if err := _ClearnetRegistry.contract.UnpackLog(event, "NodeActivated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ClearnetRegistryProtocolMetaData contains all meta data concerning the ClearnetRegistryProtocol contract.
var ClearnetRegistryProtocolMetaData = &bind.MetaData{
	ABI: "[{\"type\":\"function\",\"name\":\"BASE_PRICE\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"NODE_ID\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"TARGET_PRICE\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"UNBONDING_PERIOD\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint64\",\"internalType\":\"uint64\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"WARMUP_WINDOW\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint64\",\"internalType\":\"uint64\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"activate\",\"inputs\":[{\"name\":\"tokenId\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"blsPubkeyG1\",\"type\":\"uint256[2]\",\"internalType\":\"uint256[2]\"},{\"name\":\"blsPubkeyG2\",\"type\":\"uint256[4]\",\"internalType\":\"uint256[4]\"},{\"name\":\"operatorCollateral\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"nodeId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"activeCount\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint32\",\"internalType\":\"uint32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"floorPrice\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"fund\",\"inputs\":[{\"name\":\"tokenId\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"amount\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"getNodeByBlsG2Hash\",\"inputs\":[{\"name\":\"blsG2Hash\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getNodeById\",\"inputs\":[{\"name\":\"nodeId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[{\"name\":\"\",\"type\":\"tuple\",\"internalType\":\"structNodeRecord\",\"components\":[{\"name\":\"operator\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"activatedAt\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"deactivatedAt\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"vestedAt\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"index\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"tokenId\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"operatorCollateral\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"sponsorCollateral\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"blsPubkeyG1\",\"type\":\"uint256[2]\",\"internalType\":\"uint256[2]\"},{\"name\":\"blsPubkeyG2\",\"type\":\"uint256[4]\",\"internalType\":\"uint256[4]\"}]}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getNodeId\",\"inputs\":[{\"name\":\"tokenId\",\"type\":\"uint32\",\"internalType\":\"uint32\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getNodeIds\",\"inputs\":[{\"name\":\"offset\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"limit\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32[]\",\"internalType\":\"bytes32[]\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getNodes\",\"inputs\":[{\"name\":\"offset\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"limit\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"tuple[]\",\"internalType\":\"structNodeRecord[]\",\"components\":[{\"name\":\"operator\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"activatedAt\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"deactivatedAt\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"vestedAt\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"index\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"tokenId\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"operatorCollateral\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"sponsorCollateral\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"blsPubkeyG1\",\"type\":\"uint256[2]\",\"internalType\":\"uint256[2]\"},{\"name\":\"blsPubkeyG2\",\"type\":\"uint256[4]\",\"internalType\":\"uint256[4]\"}]}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"liability\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"register\",\"inputs\":[{\"name\":\"blsPubkeyG1\",\"type\":\"uint256[2]\",\"internalType\":\"uint256[2]\"},{\"name\":\"blsPubkeyG2\",\"type\":\"uint256[4]\",\"internalType\":\"uint256[4]\"},{\"name\":\"operatorCollateral\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"tokenId\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"nodeId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"release\",\"inputs\":[{\"name\":\"tokenId\",\"type\":\"uint32\",\"internalType\":\"uint32\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"totalNodes\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"unlock\",\"inputs\":[{\"name\":\"tokenId\",\"type\":\"uint32\",\"internalType\":\"uint32\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"event\",\"name\":\"NodeActivated\",\"inputs\":[{\"name\":\"operator\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"nodeId\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"},{\"name\":\"tokenId\",\"type\":\"uint32\",\"indexed\":true,\"internalType\":\"uint32\"},{\"name\":\"collateral\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"},{\"name\":\"vestedAt\",\"type\":\"uint64\",\"indexed\":false,\"internalType\":\"uint64\"},{\"name\":\"blsPubkeyG2\",\"type\":\"uint256[4]\",\"indexed\":false,\"internalType\":\"uint256[4]\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"NodeFunded\",\"inputs\":[{\"name\":\"payer\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"nodeId\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"},{\"name\":\"tokenId\",\"type\":\"uint32\",\"indexed\":true,\"internalType\":\"uint32\"},{\"name\":\"amount\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"},{\"name\":\"totalCollateral\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"NodeReleased\",\"inputs\":[{\"name\":\"operator\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"nodeId\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"},{\"name\":\"tokenId\",\"type\":\"uint32\",\"indexed\":true,\"internalType\":\"uint32\"},{\"name\":\"collateral\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"NodeUnlocked\",\"inputs\":[{\"name\":\"operator\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"nodeId\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"},{\"name\":\"availableAt\",\"type\":\"uint64\",\"indexed\":false,\"internalType\":\"uint64\"}],\"anonymous\":false}]",
}

// ClearnetRegistryProtocolABI is the input ABI used to generate the binding from.
// Deprecated: Use ClearnetRegistryProtocolMetaData.ABI instead.
var ClearnetRegistryProtocolABI = ClearnetRegistryProtocolMetaData.ABI

// ClearnetRegistryProtocol is an auto generated Go binding around an Ethereum contract.
type ClearnetRegistryProtocol struct {
	ClearnetRegistryProtocolCaller     // Read-only binding to the contract
	ClearnetRegistryProtocolTransactor // Write-only binding to the contract
	ClearnetRegistryProtocolFilterer   // Log filterer for contract events
}

// ClearnetRegistryProtocolCaller is an auto generated read-only Go binding around an Ethereum contract.
type ClearnetRegistryProtocolCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ClearnetRegistryProtocolTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ClearnetRegistryProtocolTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ClearnetRegistryProtocolFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type ClearnetRegistryProtocolFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ClearnetRegistryProtocolSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ClearnetRegistryProtocolSession struct {
	Contract     *ClearnetRegistryProtocol // Generic contract binding to set the session for
	CallOpts     bind.CallOpts             // Call options to use throughout this session
	TransactOpts bind.TransactOpts         // Transaction auth options to use throughout this session
}

// ClearnetRegistryProtocolCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ClearnetRegistryProtocolCallerSession struct {
	Contract *ClearnetRegistryProtocolCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts                   // Call options to use throughout this session
}

// ClearnetRegistryProtocolTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ClearnetRegistryProtocolTransactorSession struct {
	Contract     *ClearnetRegistryProtocolTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts                   // Transaction auth options to use throughout this session
}

// ClearnetRegistryProtocolRaw is an auto generated low-level Go binding around an Ethereum contract.
type ClearnetRegistryProtocolRaw struct {
	Contract *ClearnetRegistryProtocol // Generic contract binding to access the raw methods on
}

// ClearnetRegistryProtocolCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ClearnetRegistryProtocolCallerRaw struct {
	Contract *ClearnetRegistryProtocolCaller // Generic read-only contract binding to access the raw methods on
}

// ClearnetRegistryProtocolTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ClearnetRegistryProtocolTransactorRaw struct {
	Contract *ClearnetRegistryProtocolTransactor // Generic write-only contract binding to access the raw methods on
}

// NewClearnetRegistryProtocol creates a new instance of ClearnetRegistryProtocol, bound to a specific deployed contract.
func NewClearnetRegistryProtocol(address common.Address, backend bind.ContractBackend) (*ClearnetRegistryProtocol, error) {
	contract, err := bindClearnetRegistryProtocol(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &ClearnetRegistryProtocol{ClearnetRegistryProtocolCaller: ClearnetRegistryProtocolCaller{contract: contract}, ClearnetRegistryProtocolTransactor: ClearnetRegistryProtocolTransactor{contract: contract}, ClearnetRegistryProtocolFilterer: ClearnetRegistryProtocolFilterer{contract: contract}}, nil
}

// NewClearnetRegistryProtocolCaller creates a new read-only instance of ClearnetRegistryProtocol, bound to a specific deployed contract.
func NewClearnetRegistryProtocolCaller(address common.Address, caller bind.ContractCaller) (*ClearnetRegistryProtocolCaller, error) {
	contract, err := bindClearnetRegistryProtocol(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ClearnetRegistryProtocolCaller{contract: contract}, nil
}

// NewClearnetRegistryProtocolTransactor creates a new write-only instance of ClearnetRegistryProtocol, bound to a specific deployed contract.
func NewClearnetRegistryProtocolTransactor(address common.Address, transactor bind.ContractTransactor) (*ClearnetRegistryProtocolTransactor, error) {
	contract, err := bindClearnetRegistryProtocol(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &ClearnetRegistryProtocolTransactor{contract: contract}, nil
}

// NewClearnetRegistryProtocolFilterer creates a new log filterer instance of ClearnetRegistryProtocol, bound to a specific deployed contract.
func NewClearnetRegistryProtocolFilterer(address common.Address, filterer bind.ContractFilterer) (*ClearnetRegistryProtocolFilterer, error) {
	contract, err := bindClearnetRegistryProtocol(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &ClearnetRegistryProtocolFilterer{contract: contract}, nil
}

// bindClearnetRegistryProtocol binds a generic wrapper to an already deployed contract.
func bindClearnetRegistryProtocol(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := ClearnetRegistryProtocolMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ClearnetRegistryProtocol *ClearnetRegistryProtocolRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ClearnetRegistryProtocol.Contract.ClearnetRegistryProtocolCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ClearnetRegistryProtocol *ClearnetRegistryProtocolRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ClearnetRegistryProtocol.Contract.ClearnetRegistryProtocolTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ClearnetRegistryProtocol *ClearnetRegistryProtocolRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ClearnetRegistryProtocol.Contract.ClearnetRegistryProtocolTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ClearnetRegistryProtocol *ClearnetRegistryProtocolCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ClearnetRegistryProtocol.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ClearnetRegistryProtocol *ClearnetRegistryProtocolTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ClearnetRegistryProtocol.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ClearnetRegistryProtocol *ClearnetRegistryProtocolTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ClearnetRegistryProtocol.Contract.contract.Transact(opts, method, params...)
}

// BASEPRICE is a free data retrieval call binding the contract method 0xf86325ed.
//
// Solidity: function BASE_PRICE() view returns(uint256)
func (_ClearnetRegistryProtocol *ClearnetRegistryProtocolCaller) BASEPRICE(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _ClearnetRegistryProtocol.contract.Call(opts, &out, "BASE_PRICE")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// BASEPRICE is a free data retrieval call binding the contract method 0xf86325ed.
//
// Solidity: function BASE_PRICE() view returns(uint256)
func (_ClearnetRegistryProtocol *ClearnetRegistryProtocolSession) BASEPRICE() (*big.Int, error) {
	return _ClearnetRegistryProtocol.Contract.BASEPRICE(&_ClearnetRegistryProtocol.CallOpts)
}

// BASEPRICE is a free data retrieval call binding the contract method 0xf86325ed.
//
// Solidity: function BASE_PRICE() view returns(uint256)
func (_ClearnetRegistryProtocol *ClearnetRegistryProtocolCallerSession) BASEPRICE() (*big.Int, error) {
	return _ClearnetRegistryProtocol.Contract.BASEPRICE(&_ClearnetRegistryProtocol.CallOpts)
}

// NODEID is a free data retrieval call binding the contract method 0xef695be8.
//
// Solidity: function NODE_ID() view returns(address)
func (_ClearnetRegistryProtocol *ClearnetRegistryProtocolCaller) NODEID(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _ClearnetRegistryProtocol.contract.Call(opts, &out, "NODE_ID")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// NODEID is a free data retrieval call binding the contract method 0xef695be8.
//
// Solidity: function NODE_ID() view returns(address)
func (_ClearnetRegistryProtocol *ClearnetRegistryProtocolSession) NODEID() (common.Address, error) {
	return _ClearnetRegistryProtocol.Contract.NODEID(&_ClearnetRegistryProtocol.CallOpts)
}

// NODEID is a free data retrieval call binding the contract method 0xef695be8.
//
// Solidity: function NODE_ID() view returns(address)
func (_ClearnetRegistryProtocol *ClearnetRegistryProtocolCallerSession) NODEID() (common.Address, error) {
	return _ClearnetRegistryProtocol.Contract.NODEID(&_ClearnetRegistryProtocol.CallOpts)
}

// TARGETPRICE is a free data retrieval call binding the contract method 0x7e99ce59.
//
// Solidity: function TARGET_PRICE() view returns(uint256)
func (_ClearnetRegistryProtocol *ClearnetRegistryProtocolCaller) TARGETPRICE(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _ClearnetRegistryProtocol.contract.Call(opts, &out, "TARGET_PRICE")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TARGETPRICE is a free data retrieval call binding the contract method 0x7e99ce59.
//
// Solidity: function TARGET_PRICE() view returns(uint256)
func (_ClearnetRegistryProtocol *ClearnetRegistryProtocolSession) TARGETPRICE() (*big.Int, error) {
	return _ClearnetRegistryProtocol.Contract.TARGETPRICE(&_ClearnetRegistryProtocol.CallOpts)
}

// TARGETPRICE is a free data retrieval call binding the contract method 0x7e99ce59.
//
// Solidity: function TARGET_PRICE() view returns(uint256)
func (_ClearnetRegistryProtocol *ClearnetRegistryProtocolCallerSession) TARGETPRICE() (*big.Int, error) {
	return _ClearnetRegistryProtocol.Contract.TARGETPRICE(&_ClearnetRegistryProtocol.CallOpts)
}

// UNBONDINGPERIOD is a free data retrieval call binding the contract method 0xd9a912ec.
//
// Solidity: function UNBONDING_PERIOD() view returns(uint64)
func (_ClearnetRegistryProtocol *ClearnetRegistryProtocolCaller) UNBONDINGPERIOD(opts *bind.CallOpts) (uint64, error) {
	var out []interface{}
	err := _ClearnetRegistryProtocol.contract.Call(opts, &out, "UNBONDING_PERIOD")

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// UNBONDINGPERIOD is a free data retrieval call binding the contract method 0xd9a912ec.
//
// Solidity: function UNBONDING_PERIOD() view returns(uint64)
func (_ClearnetRegistryProtocol *ClearnetRegistryProtocolSession) UNBONDINGPERIOD() (uint64, error) {
	return _ClearnetRegistryProtocol.Contract.UNBONDINGPERIOD(&_ClearnetRegistryProtocol.CallOpts)
}

// UNBONDINGPERIOD is a free data retrieval call binding the contract method 0xd9a912ec.
//
// Solidity: function UNBONDING_PERIOD() view returns(uint64)
func (_ClearnetRegistryProtocol *ClearnetRegistryProtocolCallerSession) UNBONDINGPERIOD() (uint64, error) {
	return _ClearnetRegistryProtocol.Contract.UNBONDINGPERIOD(&_ClearnetRegistryProtocol.CallOpts)
}

// WARMUPWINDOW is a free data retrieval call binding the contract method 0x0d420090.
//
// Solidity: function WARMUP_WINDOW() view returns(uint64)
func (_ClearnetRegistryProtocol *ClearnetRegistryProtocolCaller) WARMUPWINDOW(opts *bind.CallOpts) (uint64, error) {
	var out []interface{}
	err := _ClearnetRegistryProtocol.contract.Call(opts, &out, "WARMUP_WINDOW")

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// WARMUPWINDOW is a free data retrieval call binding the contract method 0x0d420090.
//
// Solidity: function WARMUP_WINDOW() view returns(uint64)
func (_ClearnetRegistryProtocol *ClearnetRegistryProtocolSession) WARMUPWINDOW() (uint64, error) {
	return _ClearnetRegistryProtocol.Contract.WARMUPWINDOW(&_ClearnetRegistryProtocol.CallOpts)
}

// WARMUPWINDOW is a free data retrieval call binding the contract method 0x0d420090.
//
// Solidity: function WARMUP_WINDOW() view returns(uint64)
func (_ClearnetRegistryProtocol *ClearnetRegistryProtocolCallerSession) WARMUPWINDOW() (uint64, error) {
	return _ClearnetRegistryProtocol.Contract.WARMUPWINDOW(&_ClearnetRegistryProtocol.CallOpts)
}

// ActiveCount is a free data retrieval call binding the contract method 0x4331ed1f.
//
// Solidity: function activeCount() view returns(uint32)
func (_ClearnetRegistryProtocol *ClearnetRegistryProtocolCaller) ActiveCount(opts *bind.CallOpts) (uint32, error) {
	var out []interface{}
	err := _ClearnetRegistryProtocol.contract.Call(opts, &out, "activeCount")

	if err != nil {
		return *new(uint32), err
	}

	out0 := *abi.ConvertType(out[0], new(uint32)).(*uint32)

	return out0, err

}

// ActiveCount is a free data retrieval call binding the contract method 0x4331ed1f.
//
// Solidity: function activeCount() view returns(uint32)
func (_ClearnetRegistryProtocol *ClearnetRegistryProtocolSession) ActiveCount() (uint32, error) {
	return _ClearnetRegistryProtocol.Contract.ActiveCount(&_ClearnetRegistryProtocol.CallOpts)
}

// ActiveCount is a free data retrieval call binding the contract method 0x4331ed1f.
//
// Solidity: function activeCount() view returns(uint32)
func (_ClearnetRegistryProtocol *ClearnetRegistryProtocolCallerSession) ActiveCount() (uint32, error) {
	return _ClearnetRegistryProtocol.Contract.ActiveCount(&_ClearnetRegistryProtocol.CallOpts)
}

// FloorPrice is a free data retrieval call binding the contract method 0x9363c812.
//
// Solidity: function floorPrice() view returns(uint256)
func (_ClearnetRegistryProtocol *ClearnetRegistryProtocolCaller) FloorPrice(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _ClearnetRegistryProtocol.contract.Call(opts, &out, "floorPrice")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// FloorPrice is a free data retrieval call binding the contract method 0x9363c812.
//
// Solidity: function floorPrice() view returns(uint256)
func (_ClearnetRegistryProtocol *ClearnetRegistryProtocolSession) FloorPrice() (*big.Int, error) {
	return _ClearnetRegistryProtocol.Contract.FloorPrice(&_ClearnetRegistryProtocol.CallOpts)
}

// FloorPrice is a free data retrieval call binding the contract method 0x9363c812.
//
// Solidity: function floorPrice() view returns(uint256)
func (_ClearnetRegistryProtocol *ClearnetRegistryProtocolCallerSession) FloorPrice() (*big.Int, error) {
	return _ClearnetRegistryProtocol.Contract.FloorPrice(&_ClearnetRegistryProtocol.CallOpts)
}

// GetNodeByBlsG2Hash is a free data retrieval call binding the contract method 0x18071936.
//
// Solidity: function getNodeByBlsG2Hash(bytes32 blsG2Hash) view returns(bytes32)
func (_ClearnetRegistryProtocol *ClearnetRegistryProtocolCaller) GetNodeByBlsG2Hash(opts *bind.CallOpts, blsG2Hash [32]byte) ([32]byte, error) {
	var out []interface{}
	err := _ClearnetRegistryProtocol.contract.Call(opts, &out, "getNodeByBlsG2Hash", blsG2Hash)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// GetNodeByBlsG2Hash is a free data retrieval call binding the contract method 0x18071936.
//
// Solidity: function getNodeByBlsG2Hash(bytes32 blsG2Hash) view returns(bytes32)
func (_ClearnetRegistryProtocol *ClearnetRegistryProtocolSession) GetNodeByBlsG2Hash(blsG2Hash [32]byte) ([32]byte, error) {
	return _ClearnetRegistryProtocol.Contract.GetNodeByBlsG2Hash(&_ClearnetRegistryProtocol.CallOpts, blsG2Hash)
}

// GetNodeByBlsG2Hash is a free data retrieval call binding the contract method 0x18071936.
//
// Solidity: function getNodeByBlsG2Hash(bytes32 blsG2Hash) view returns(bytes32)
func (_ClearnetRegistryProtocol *ClearnetRegistryProtocolCallerSession) GetNodeByBlsG2Hash(blsG2Hash [32]byte) ([32]byte, error) {
	return _ClearnetRegistryProtocol.Contract.GetNodeByBlsG2Hash(&_ClearnetRegistryProtocol.CallOpts, blsG2Hash)
}

// GetNodeById is a free data retrieval call binding the contract method 0x8899cf50.
//
// Solidity: function getNodeById(bytes32 nodeId) view returns((address,uint64,uint64,uint64,uint32,uint32,uint256,uint256,uint256[2],uint256[4]))
func (_ClearnetRegistryProtocol *ClearnetRegistryProtocolCaller) GetNodeById(opts *bind.CallOpts, nodeId [32]byte) (NodeRecord, error) {
	var out []interface{}
	err := _ClearnetRegistryProtocol.contract.Call(opts, &out, "getNodeById", nodeId)

	if err != nil {
		return *new(NodeRecord), err
	}

	out0 := *abi.ConvertType(out[0], new(NodeRecord)).(*NodeRecord)

	return out0, err

}

// GetNodeById is a free data retrieval call binding the contract method 0x8899cf50.
//
// Solidity: function getNodeById(bytes32 nodeId) view returns((address,uint64,uint64,uint64,uint32,uint32,uint256,uint256,uint256[2],uint256[4]))
func (_ClearnetRegistryProtocol *ClearnetRegistryProtocolSession) GetNodeById(nodeId [32]byte) (NodeRecord, error) {
	return _ClearnetRegistryProtocol.Contract.GetNodeById(&_ClearnetRegistryProtocol.CallOpts, nodeId)
}

// GetNodeById is a free data retrieval call binding the contract method 0x8899cf50.
//
// Solidity: function getNodeById(bytes32 nodeId) view returns((address,uint64,uint64,uint64,uint32,uint32,uint256,uint256,uint256[2],uint256[4]))
func (_ClearnetRegistryProtocol *ClearnetRegistryProtocolCallerSession) GetNodeById(nodeId [32]byte) (NodeRecord, error) {
	return _ClearnetRegistryProtocol.Contract.GetNodeById(&_ClearnetRegistryProtocol.CallOpts, nodeId)
}

// GetNodeId is a free data retrieval call binding the contract method 0x6ef67bae.
//
// Solidity: function getNodeId(uint32 tokenId) view returns(bytes32)
func (_ClearnetRegistryProtocol *ClearnetRegistryProtocolCaller) GetNodeId(opts *bind.CallOpts, tokenId uint32) ([32]byte, error) {
	var out []interface{}
	err := _ClearnetRegistryProtocol.contract.Call(opts, &out, "getNodeId", tokenId)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// GetNodeId is a free data retrieval call binding the contract method 0x6ef67bae.
//
// Solidity: function getNodeId(uint32 tokenId) view returns(bytes32)
func (_ClearnetRegistryProtocol *ClearnetRegistryProtocolSession) GetNodeId(tokenId uint32) ([32]byte, error) {
	return _ClearnetRegistryProtocol.Contract.GetNodeId(&_ClearnetRegistryProtocol.CallOpts, tokenId)
}

// GetNodeId is a free data retrieval call binding the contract method 0x6ef67bae.
//
// Solidity: function getNodeId(uint32 tokenId) view returns(bytes32)
func (_ClearnetRegistryProtocol *ClearnetRegistryProtocolCallerSession) GetNodeId(tokenId uint32) ([32]byte, error) {
	return _ClearnetRegistryProtocol.Contract.GetNodeId(&_ClearnetRegistryProtocol.CallOpts, tokenId)
}

// GetNodeIds is a free data retrieval call binding the contract method 0xbdc43e92.
//
// Solidity: function getNodeIds(uint256 offset, uint256 limit) view returns(bytes32[])
func (_ClearnetRegistryProtocol *ClearnetRegistryProtocolCaller) GetNodeIds(opts *bind.CallOpts, offset *big.Int, limit *big.Int) ([][32]byte, error) {
	var out []interface{}
	err := _ClearnetRegistryProtocol.contract.Call(opts, &out, "getNodeIds", offset, limit)

	if err != nil {
		return *new([][32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([][32]byte)).(*[][32]byte)

	return out0, err

}

// GetNodeIds is a free data retrieval call binding the contract method 0xbdc43e92.
//
// Solidity: function getNodeIds(uint256 offset, uint256 limit) view returns(bytes32[])
func (_ClearnetRegistryProtocol *ClearnetRegistryProtocolSession) GetNodeIds(offset *big.Int, limit *big.Int) ([][32]byte, error) {
	return _ClearnetRegistryProtocol.Contract.GetNodeIds(&_ClearnetRegistryProtocol.CallOpts, offset, limit)
}

// GetNodeIds is a free data retrieval call binding the contract method 0xbdc43e92.
//
// Solidity: function getNodeIds(uint256 offset, uint256 limit) view returns(bytes32[])
func (_ClearnetRegistryProtocol *ClearnetRegistryProtocolCallerSession) GetNodeIds(offset *big.Int, limit *big.Int) ([][32]byte, error) {
	return _ClearnetRegistryProtocol.Contract.GetNodeIds(&_ClearnetRegistryProtocol.CallOpts, offset, limit)
}

// GetNodes is a free data retrieval call binding the contract method 0x038d67e8.
//
// Solidity: function getNodes(uint256 offset, uint256 limit) view returns((address,uint64,uint64,uint64,uint32,uint32,uint256,uint256,uint256[2],uint256[4])[])
func (_ClearnetRegistryProtocol *ClearnetRegistryProtocolCaller) GetNodes(opts *bind.CallOpts, offset *big.Int, limit *big.Int) ([]NodeRecord, error) {
	var out []interface{}
	err := _ClearnetRegistryProtocol.contract.Call(opts, &out, "getNodes", offset, limit)

	if err != nil {
		return *new([]NodeRecord), err
	}

	out0 := *abi.ConvertType(out[0], new([]NodeRecord)).(*[]NodeRecord)

	return out0, err

}

// GetNodes is a free data retrieval call binding the contract method 0x038d67e8.
//
// Solidity: function getNodes(uint256 offset, uint256 limit) view returns((address,uint64,uint64,uint64,uint32,uint32,uint256,uint256,uint256[2],uint256[4])[])
func (_ClearnetRegistryProtocol *ClearnetRegistryProtocolSession) GetNodes(offset *big.Int, limit *big.Int) ([]NodeRecord, error) {
	return _ClearnetRegistryProtocol.Contract.GetNodes(&_ClearnetRegistryProtocol.CallOpts, offset, limit)
}

// GetNodes is a free data retrieval call binding the contract method 0x038d67e8.
//
// Solidity: function getNodes(uint256 offset, uint256 limit) view returns((address,uint64,uint64,uint64,uint32,uint32,uint256,uint256,uint256[2],uint256[4])[])
func (_ClearnetRegistryProtocol *ClearnetRegistryProtocolCallerSession) GetNodes(offset *big.Int, limit *big.Int) ([]NodeRecord, error) {
	return _ClearnetRegistryProtocol.Contract.GetNodes(&_ClearnetRegistryProtocol.CallOpts, offset, limit)
}

// Liability is a free data retrieval call binding the contract method 0x705727b5.
//
// Solidity: function liability() view returns(uint256)
func (_ClearnetRegistryProtocol *ClearnetRegistryProtocolCaller) Liability(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _ClearnetRegistryProtocol.contract.Call(opts, &out, "liability")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Liability is a free data retrieval call binding the contract method 0x705727b5.
//
// Solidity: function liability() view returns(uint256)
func (_ClearnetRegistryProtocol *ClearnetRegistryProtocolSession) Liability() (*big.Int, error) {
	return _ClearnetRegistryProtocol.Contract.Liability(&_ClearnetRegistryProtocol.CallOpts)
}

// Liability is a free data retrieval call binding the contract method 0x705727b5.
//
// Solidity: function liability() view returns(uint256)
func (_ClearnetRegistryProtocol *ClearnetRegistryProtocolCallerSession) Liability() (*big.Int, error) {
	return _ClearnetRegistryProtocol.Contract.Liability(&_ClearnetRegistryProtocol.CallOpts)
}

// TotalNodes is a free data retrieval call binding the contract method 0x9592d424.
//
// Solidity: function totalNodes() view returns(uint256)
func (_ClearnetRegistryProtocol *ClearnetRegistryProtocolCaller) TotalNodes(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _ClearnetRegistryProtocol.contract.Call(opts, &out, "totalNodes")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TotalNodes is a free data retrieval call binding the contract method 0x9592d424.
//
// Solidity: function totalNodes() view returns(uint256)
func (_ClearnetRegistryProtocol *ClearnetRegistryProtocolSession) TotalNodes() (*big.Int, error) {
	return _ClearnetRegistryProtocol.Contract.TotalNodes(&_ClearnetRegistryProtocol.CallOpts)
}

// TotalNodes is a free data retrieval call binding the contract method 0x9592d424.
//
// Solidity: function totalNodes() view returns(uint256)
func (_ClearnetRegistryProtocol *ClearnetRegistryProtocolCallerSession) TotalNodes() (*big.Int, error) {
	return _ClearnetRegistryProtocol.Contract.TotalNodes(&_ClearnetRegistryProtocol.CallOpts)
}

// Activate is a paid mutator transaction binding the contract method 0xad122111.
//
// Solidity: function activate(uint32 tokenId, uint256[2] blsPubkeyG1, uint256[4] blsPubkeyG2, uint256 operatorCollateral) returns(bytes32 nodeId)
func (_ClearnetRegistryProtocol *ClearnetRegistryProtocolTransactor) Activate(opts *bind.TransactOpts, tokenId uint32, blsPubkeyG1 [2]*big.Int, blsPubkeyG2 [4]*big.Int, operatorCollateral *big.Int) (*types.Transaction, error) {
	return _ClearnetRegistryProtocol.contract.Transact(opts, "activate", tokenId, blsPubkeyG1, blsPubkeyG2, operatorCollateral)
}

// Activate is a paid mutator transaction binding the contract method 0xad122111.
//
// Solidity: function activate(uint32 tokenId, uint256[2] blsPubkeyG1, uint256[4] blsPubkeyG2, uint256 operatorCollateral) returns(bytes32 nodeId)
func (_ClearnetRegistryProtocol *ClearnetRegistryProtocolSession) Activate(tokenId uint32, blsPubkeyG1 [2]*big.Int, blsPubkeyG2 [4]*big.Int, operatorCollateral *big.Int) (*types.Transaction, error) {
	return _ClearnetRegistryProtocol.Contract.Activate(&_ClearnetRegistryProtocol.TransactOpts, tokenId, blsPubkeyG1, blsPubkeyG2, operatorCollateral)
}

// Activate is a paid mutator transaction binding the contract method 0xad122111.
//
// Solidity: function activate(uint32 tokenId, uint256[2] blsPubkeyG1, uint256[4] blsPubkeyG2, uint256 operatorCollateral) returns(bytes32 nodeId)
func (_ClearnetRegistryProtocol *ClearnetRegistryProtocolTransactorSession) Activate(tokenId uint32, blsPubkeyG1 [2]*big.Int, blsPubkeyG2 [4]*big.Int, operatorCollateral *big.Int) (*types.Transaction, error) {
	return _ClearnetRegistryProtocol.Contract.Activate(&_ClearnetRegistryProtocol.TransactOpts, tokenId, blsPubkeyG1, blsPubkeyG2, operatorCollateral)
}

// Fund is a paid mutator transaction binding the contract method 0x2db75d40.
//
// Solidity: function fund(uint32 tokenId, uint256 amount) returns()
func (_ClearnetRegistryProtocol *ClearnetRegistryProtocolTransactor) Fund(opts *bind.TransactOpts, tokenId uint32, amount *big.Int) (*types.Transaction, error) {
	return _ClearnetRegistryProtocol.contract.Transact(opts, "fund", tokenId, amount)
}

// Fund is a paid mutator transaction binding the contract method 0x2db75d40.
//
// Solidity: function fund(uint32 tokenId, uint256 amount) returns()
func (_ClearnetRegistryProtocol *ClearnetRegistryProtocolSession) Fund(tokenId uint32, amount *big.Int) (*types.Transaction, error) {
	return _ClearnetRegistryProtocol.Contract.Fund(&_ClearnetRegistryProtocol.TransactOpts, tokenId, amount)
}

// Fund is a paid mutator transaction binding the contract method 0x2db75d40.
//
// Solidity: function fund(uint32 tokenId, uint256 amount) returns()
func (_ClearnetRegistryProtocol *ClearnetRegistryProtocolTransactorSession) Fund(tokenId uint32, amount *big.Int) (*types.Transaction, error) {
	return _ClearnetRegistryProtocol.Contract.Fund(&_ClearnetRegistryProtocol.TransactOpts, tokenId, amount)
}

// Register is a paid mutator transaction binding the contract method 0x7fdd1867.
//
// Solidity: function register(uint256[2] blsPubkeyG1, uint256[4] blsPubkeyG2, uint256 operatorCollateral) returns(uint32 tokenId, bytes32 nodeId)
func (_ClearnetRegistryProtocol *ClearnetRegistryProtocolTransactor) Register(opts *bind.TransactOpts, blsPubkeyG1 [2]*big.Int, blsPubkeyG2 [4]*big.Int, operatorCollateral *big.Int) (*types.Transaction, error) {
	return _ClearnetRegistryProtocol.contract.Transact(opts, "register", blsPubkeyG1, blsPubkeyG2, operatorCollateral)
}

// Register is a paid mutator transaction binding the contract method 0x7fdd1867.
//
// Solidity: function register(uint256[2] blsPubkeyG1, uint256[4] blsPubkeyG2, uint256 operatorCollateral) returns(uint32 tokenId, bytes32 nodeId)
func (_ClearnetRegistryProtocol *ClearnetRegistryProtocolSession) Register(blsPubkeyG1 [2]*big.Int, blsPubkeyG2 [4]*big.Int, operatorCollateral *big.Int) (*types.Transaction, error) {
	return _ClearnetRegistryProtocol.Contract.Register(&_ClearnetRegistryProtocol.TransactOpts, blsPubkeyG1, blsPubkeyG2, operatorCollateral)
}

// Register is a paid mutator transaction binding the contract method 0x7fdd1867.
//
// Solidity: function register(uint256[2] blsPubkeyG1, uint256[4] blsPubkeyG2, uint256 operatorCollateral) returns(uint32 tokenId, bytes32 nodeId)
func (_ClearnetRegistryProtocol *ClearnetRegistryProtocolTransactorSession) Register(blsPubkeyG1 [2]*big.Int, blsPubkeyG2 [4]*big.Int, operatorCollateral *big.Int) (*types.Transaction, error) {
	return _ClearnetRegistryProtocol.Contract.Register(&_ClearnetRegistryProtocol.TransactOpts, blsPubkeyG1, blsPubkeyG2, operatorCollateral)
}

// Release is a paid mutator transaction binding the contract method 0xdda42b37.
//
// Solidity: function release(uint32 tokenId) returns()
func (_ClearnetRegistryProtocol *ClearnetRegistryProtocolTransactor) Release(opts *bind.TransactOpts, tokenId uint32) (*types.Transaction, error) {
	return _ClearnetRegistryProtocol.contract.Transact(opts, "release", tokenId)
}

// Release is a paid mutator transaction binding the contract method 0xdda42b37.
//
// Solidity: function release(uint32 tokenId) returns()
func (_ClearnetRegistryProtocol *ClearnetRegistryProtocolSession) Release(tokenId uint32) (*types.Transaction, error) {
	return _ClearnetRegistryProtocol.Contract.Release(&_ClearnetRegistryProtocol.TransactOpts, tokenId)
}

// Release is a paid mutator transaction binding the contract method 0xdda42b37.
//
// Solidity: function release(uint32 tokenId) returns()
func (_ClearnetRegistryProtocol *ClearnetRegistryProtocolTransactorSession) Release(tokenId uint32) (*types.Transaction, error) {
	return _ClearnetRegistryProtocol.Contract.Release(&_ClearnetRegistryProtocol.TransactOpts, tokenId)
}

// Unlock is a paid mutator transaction binding the contract method 0x67d93c81.
//
// Solidity: function unlock(uint32 tokenId) returns()
func (_ClearnetRegistryProtocol *ClearnetRegistryProtocolTransactor) Unlock(opts *bind.TransactOpts, tokenId uint32) (*types.Transaction, error) {
	return _ClearnetRegistryProtocol.contract.Transact(opts, "unlock", tokenId)
}

// Unlock is a paid mutator transaction binding the contract method 0x67d93c81.
//
// Solidity: function unlock(uint32 tokenId) returns()
func (_ClearnetRegistryProtocol *ClearnetRegistryProtocolSession) Unlock(tokenId uint32) (*types.Transaction, error) {
	return _ClearnetRegistryProtocol.Contract.Unlock(&_ClearnetRegistryProtocol.TransactOpts, tokenId)
}

// Unlock is a paid mutator transaction binding the contract method 0x67d93c81.
//
// Solidity: function unlock(uint32 tokenId) returns()
func (_ClearnetRegistryProtocol *ClearnetRegistryProtocolTransactorSession) Unlock(tokenId uint32) (*types.Transaction, error) {
	return _ClearnetRegistryProtocol.Contract.Unlock(&_ClearnetRegistryProtocol.TransactOpts, tokenId)
}

// ClearnetRegistryProtocolNodeActivatedIterator is returned from FilterNodeActivated and is used to iterate over the raw logs and unpacked data for NodeActivated events raised by the ClearnetRegistryProtocol contract.
type ClearnetRegistryProtocolNodeActivatedIterator struct {
	Event *ClearnetRegistryProtocolNodeActivated // Event containing the contract specifics and raw log

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
func (it *ClearnetRegistryProtocolNodeActivatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ClearnetRegistryProtocolNodeActivated)
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
		it.Event = new(ClearnetRegistryProtocolNodeActivated)
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
func (it *ClearnetRegistryProtocolNodeActivatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ClearnetRegistryProtocolNodeActivatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ClearnetRegistryProtocolNodeActivated represents a NodeActivated event raised by the ClearnetRegistryProtocol contract.
type ClearnetRegistryProtocolNodeActivated struct {
	Operator    common.Address
	NodeId      [32]byte
	TokenId     uint32
	Collateral  *big.Int
	VestedAt    uint64
	BlsPubkeyG2 [4]*big.Int
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterNodeActivated is a free log retrieval operation binding the contract event 0x58cee7261629c956b111bc684df727bd2e9b0f5d954e24b93908951c431cd13e.
//
// Solidity: event NodeActivated(address indexed operator, bytes32 indexed nodeId, uint32 indexed tokenId, uint256 collateral, uint64 vestedAt, uint256[4] blsPubkeyG2)
func (_ClearnetRegistryProtocol *ClearnetRegistryProtocolFilterer) FilterNodeActivated(opts *bind.FilterOpts, operator []common.Address, nodeId [][32]byte, tokenId []uint32) (*ClearnetRegistryProtocolNodeActivatedIterator, error) {

	var operatorRule []interface{}
	for _, operatorItem := range operator {
		operatorRule = append(operatorRule, operatorItem)
	}
	var nodeIdRule []interface{}
	for _, nodeIdItem := range nodeId {
		nodeIdRule = append(nodeIdRule, nodeIdItem)
	}
	var tokenIdRule []interface{}
	for _, tokenIdItem := range tokenId {
		tokenIdRule = append(tokenIdRule, tokenIdItem)
	}

	logs, sub, err := _ClearnetRegistryProtocol.contract.FilterLogs(opts, "NodeActivated", operatorRule, nodeIdRule, tokenIdRule)
	if err != nil {
		return nil, err
	}
	return &ClearnetRegistryProtocolNodeActivatedIterator{contract: _ClearnetRegistryProtocol.contract, event: "NodeActivated", logs: logs, sub: sub}, nil
}

// WatchNodeActivated is a free log subscription operation binding the contract event 0x58cee7261629c956b111bc684df727bd2e9b0f5d954e24b93908951c431cd13e.
//
// Solidity: event NodeActivated(address indexed operator, bytes32 indexed nodeId, uint32 indexed tokenId, uint256 collateral, uint64 vestedAt, uint256[4] blsPubkeyG2)
func (_ClearnetRegistryProtocol *ClearnetRegistryProtocolFilterer) WatchNodeActivated(opts *bind.WatchOpts, sink chan<- *ClearnetRegistryProtocolNodeActivated, operator []common.Address, nodeId [][32]byte, tokenId []uint32) (event.Subscription, error) {

	var operatorRule []interface{}
	for _, operatorItem := range operator {
		operatorRule = append(operatorRule, operatorItem)
	}
	var nodeIdRule []interface{}
	for _, nodeIdItem := range nodeId {
		nodeIdRule = append(nodeIdRule, nodeIdItem)
	}
	var tokenIdRule []interface{}
	for _, tokenIdItem := range tokenId {
		tokenIdRule = append(tokenIdRule, tokenIdItem)
	}

	logs, sub, err := _ClearnetRegistryProtocol.contract.WatchLogs(opts, "NodeActivated", operatorRule, nodeIdRule, tokenIdRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ClearnetRegistryProtocolNodeActivated)
				if err := _ClearnetRegistryProtocol.contract.UnpackLog(event, "NodeActivated", log); err != nil {
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

// ParseNodeActivated is a log parse operation binding the contract event 0x58cee7261629c956b111bc684df727bd2e9b0f5d954e24b93908951c431cd13e.
//
// Solidity: event NodeActivated(address indexed operator, bytes32 indexed nodeId, uint32 indexed tokenId, uint256 collateral, uint64 vestedAt, uint256[4] blsPubkeyG2)
func (_ClearnetRegistryProtocol *ClearnetRegistryProtocolFilterer) ParseNodeActivated(log types.Log) (*ClearnetRegistryProtocolNodeActivated, error) {
	event := new(ClearnetRegistryProtocolNodeActivated)
	if err := _ClearnetRegistryProtocol.contract.UnpackLog(event, "NodeActivated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ClearnetRegistryProtocolNodeFundedIterator is returned from FilterNodeFunded and is used to iterate over the raw logs and unpacked data for NodeFunded events raised by the ClearnetRegistryProtocol contract.
type ClearnetRegistryProtocolNodeFundedIterator struct {
	Event *ClearnetRegistryProtocolNodeFunded // Event containing the contract specifics and raw log

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
func (it *ClearnetRegistryProtocolNodeFundedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ClearnetRegistryProtocolNodeFunded)
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
		it.Event = new(ClearnetRegistryProtocolNodeFunded)
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
func (it *ClearnetRegistryProtocolNodeFundedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ClearnetRegistryProtocolNodeFundedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ClearnetRegistryProtocolNodeFunded represents a NodeFunded event raised by the ClearnetRegistryProtocol contract.
type ClearnetRegistryProtocolNodeFunded struct {
	Payer           common.Address
	NodeId          [32]byte
	TokenId         uint32
	Amount          *big.Int
	TotalCollateral *big.Int
	Raw             types.Log // Blockchain specific contextual infos
}

// FilterNodeFunded is a free log retrieval operation binding the contract event 0x12341d30af78a74af3697daeaf7b1662bc9b723f6aa8a3402bf3d8b3f87a0772.
//
// Solidity: event NodeFunded(address indexed payer, bytes32 indexed nodeId, uint32 indexed tokenId, uint256 amount, uint256 totalCollateral)
func (_ClearnetRegistryProtocol *ClearnetRegistryProtocolFilterer) FilterNodeFunded(opts *bind.FilterOpts, payer []common.Address, nodeId [][32]byte, tokenId []uint32) (*ClearnetRegistryProtocolNodeFundedIterator, error) {

	var payerRule []interface{}
	for _, payerItem := range payer {
		payerRule = append(payerRule, payerItem)
	}
	var nodeIdRule []interface{}
	for _, nodeIdItem := range nodeId {
		nodeIdRule = append(nodeIdRule, nodeIdItem)
	}
	var tokenIdRule []interface{}
	for _, tokenIdItem := range tokenId {
		tokenIdRule = append(tokenIdRule, tokenIdItem)
	}

	logs, sub, err := _ClearnetRegistryProtocol.contract.FilterLogs(opts, "NodeFunded", payerRule, nodeIdRule, tokenIdRule)
	if err != nil {
		return nil, err
	}
	return &ClearnetRegistryProtocolNodeFundedIterator{contract: _ClearnetRegistryProtocol.contract, event: "NodeFunded", logs: logs, sub: sub}, nil
}

// WatchNodeFunded is a free log subscription operation binding the contract event 0x12341d30af78a74af3697daeaf7b1662bc9b723f6aa8a3402bf3d8b3f87a0772.
//
// Solidity: event NodeFunded(address indexed payer, bytes32 indexed nodeId, uint32 indexed tokenId, uint256 amount, uint256 totalCollateral)
func (_ClearnetRegistryProtocol *ClearnetRegistryProtocolFilterer) WatchNodeFunded(opts *bind.WatchOpts, sink chan<- *ClearnetRegistryProtocolNodeFunded, payer []common.Address, nodeId [][32]byte, tokenId []uint32) (event.Subscription, error) {

	var payerRule []interface{}
	for _, payerItem := range payer {
		payerRule = append(payerRule, payerItem)
	}
	var nodeIdRule []interface{}
	for _, nodeIdItem := range nodeId {
		nodeIdRule = append(nodeIdRule, nodeIdItem)
	}
	var tokenIdRule []interface{}
	for _, tokenIdItem := range tokenId {
		tokenIdRule = append(tokenIdRule, tokenIdItem)
	}

	logs, sub, err := _ClearnetRegistryProtocol.contract.WatchLogs(opts, "NodeFunded", payerRule, nodeIdRule, tokenIdRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ClearnetRegistryProtocolNodeFunded)
				if err := _ClearnetRegistryProtocol.contract.UnpackLog(event, "NodeFunded", log); err != nil {
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

// ParseNodeFunded is a log parse operation binding the contract event 0x12341d30af78a74af3697daeaf7b1662bc9b723f6aa8a3402bf3d8b3f87a0772.
//
// Solidity: event NodeFunded(address indexed payer, bytes32 indexed nodeId, uint32 indexed tokenId, uint256 amount, uint256 totalCollateral)
func (_ClearnetRegistryProtocol *ClearnetRegistryProtocolFilterer) ParseNodeFunded(log types.Log) (*ClearnetRegistryProtocolNodeFunded, error) {
	event := new(ClearnetRegistryProtocolNodeFunded)
	if err := _ClearnetRegistryProtocol.contract.UnpackLog(event, "NodeFunded", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ClearnetRegistryProtocolNodeReleasedIterator is returned from FilterNodeReleased and is used to iterate over the raw logs and unpacked data for NodeReleased events raised by the ClearnetRegistryProtocol contract.
type ClearnetRegistryProtocolNodeReleasedIterator struct {
	Event *ClearnetRegistryProtocolNodeReleased // Event containing the contract specifics and raw log

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
func (it *ClearnetRegistryProtocolNodeReleasedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ClearnetRegistryProtocolNodeReleased)
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
		it.Event = new(ClearnetRegistryProtocolNodeReleased)
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
func (it *ClearnetRegistryProtocolNodeReleasedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ClearnetRegistryProtocolNodeReleasedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ClearnetRegistryProtocolNodeReleased represents a NodeReleased event raised by the ClearnetRegistryProtocol contract.
type ClearnetRegistryProtocolNodeReleased struct {
	Operator   common.Address
	NodeId     [32]byte
	TokenId    uint32
	Collateral *big.Int
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterNodeReleased is a free log retrieval operation binding the contract event 0x4f72a5ea49c0470a55beb3953816abf5c92fc73003b1049c241b133a0863208c.
//
// Solidity: event NodeReleased(address indexed operator, bytes32 indexed nodeId, uint32 indexed tokenId, uint256 collateral)
func (_ClearnetRegistryProtocol *ClearnetRegistryProtocolFilterer) FilterNodeReleased(opts *bind.FilterOpts, operator []common.Address, nodeId [][32]byte, tokenId []uint32) (*ClearnetRegistryProtocolNodeReleasedIterator, error) {

	var operatorRule []interface{}
	for _, operatorItem := range operator {
		operatorRule = append(operatorRule, operatorItem)
	}
	var nodeIdRule []interface{}
	for _, nodeIdItem := range nodeId {
		nodeIdRule = append(nodeIdRule, nodeIdItem)
	}
	var tokenIdRule []interface{}
	for _, tokenIdItem := range tokenId {
		tokenIdRule = append(tokenIdRule, tokenIdItem)
	}

	logs, sub, err := _ClearnetRegistryProtocol.contract.FilterLogs(opts, "NodeReleased", operatorRule, nodeIdRule, tokenIdRule)
	if err != nil {
		return nil, err
	}
	return &ClearnetRegistryProtocolNodeReleasedIterator{contract: _ClearnetRegistryProtocol.contract, event: "NodeReleased", logs: logs, sub: sub}, nil
}

// WatchNodeReleased is a free log subscription operation binding the contract event 0x4f72a5ea49c0470a55beb3953816abf5c92fc73003b1049c241b133a0863208c.
//
// Solidity: event NodeReleased(address indexed operator, bytes32 indexed nodeId, uint32 indexed tokenId, uint256 collateral)
func (_ClearnetRegistryProtocol *ClearnetRegistryProtocolFilterer) WatchNodeReleased(opts *bind.WatchOpts, sink chan<- *ClearnetRegistryProtocolNodeReleased, operator []common.Address, nodeId [][32]byte, tokenId []uint32) (event.Subscription, error) {

	var operatorRule []interface{}
	for _, operatorItem := range operator {
		operatorRule = append(operatorRule, operatorItem)
	}
	var nodeIdRule []interface{}
	for _, nodeIdItem := range nodeId {
		nodeIdRule = append(nodeIdRule, nodeIdItem)
	}
	var tokenIdRule []interface{}
	for _, tokenIdItem := range tokenId {
		tokenIdRule = append(tokenIdRule, tokenIdItem)
	}

	logs, sub, err := _ClearnetRegistryProtocol.contract.WatchLogs(opts, "NodeReleased", operatorRule, nodeIdRule, tokenIdRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ClearnetRegistryProtocolNodeReleased)
				if err := _ClearnetRegistryProtocol.contract.UnpackLog(event, "NodeReleased", log); err != nil {
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

// ParseNodeReleased is a log parse operation binding the contract event 0x4f72a5ea49c0470a55beb3953816abf5c92fc73003b1049c241b133a0863208c.
//
// Solidity: event NodeReleased(address indexed operator, bytes32 indexed nodeId, uint32 indexed tokenId, uint256 collateral)
func (_ClearnetRegistryProtocol *ClearnetRegistryProtocolFilterer) ParseNodeReleased(log types.Log) (*ClearnetRegistryProtocolNodeReleased, error) {
	event := new(ClearnetRegistryProtocolNodeReleased)
	if err := _ClearnetRegistryProtocol.contract.UnpackLog(event, "NodeReleased", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ClearnetRegistryProtocolNodeUnlockedIterator is returned from FilterNodeUnlocked and is used to iterate over the raw logs and unpacked data for NodeUnlocked events raised by the ClearnetRegistryProtocol contract.
type ClearnetRegistryProtocolNodeUnlockedIterator struct {
	Event *ClearnetRegistryProtocolNodeUnlocked // Event containing the contract specifics and raw log

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
func (it *ClearnetRegistryProtocolNodeUnlockedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ClearnetRegistryProtocolNodeUnlocked)
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
		it.Event = new(ClearnetRegistryProtocolNodeUnlocked)
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
func (it *ClearnetRegistryProtocolNodeUnlockedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ClearnetRegistryProtocolNodeUnlockedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ClearnetRegistryProtocolNodeUnlocked represents a NodeUnlocked event raised by the ClearnetRegistryProtocol contract.
type ClearnetRegistryProtocolNodeUnlocked struct {
	Operator    common.Address
	NodeId      [32]byte
	AvailableAt uint64
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterNodeUnlocked is a free log retrieval operation binding the contract event 0x0c833c7c9f5b9b8ed0085d7959eb025f59fa32a55b8d55223a819bc7c58db345.
//
// Solidity: event NodeUnlocked(address indexed operator, bytes32 indexed nodeId, uint64 availableAt)
func (_ClearnetRegistryProtocol *ClearnetRegistryProtocolFilterer) FilterNodeUnlocked(opts *bind.FilterOpts, operator []common.Address, nodeId [][32]byte) (*ClearnetRegistryProtocolNodeUnlockedIterator, error) {

	var operatorRule []interface{}
	for _, operatorItem := range operator {
		operatorRule = append(operatorRule, operatorItem)
	}
	var nodeIdRule []interface{}
	for _, nodeIdItem := range nodeId {
		nodeIdRule = append(nodeIdRule, nodeIdItem)
	}

	logs, sub, err := _ClearnetRegistryProtocol.contract.FilterLogs(opts, "NodeUnlocked", operatorRule, nodeIdRule)
	if err != nil {
		return nil, err
	}
	return &ClearnetRegistryProtocolNodeUnlockedIterator{contract: _ClearnetRegistryProtocol.contract, event: "NodeUnlocked", logs: logs, sub: sub}, nil
}

// WatchNodeUnlocked is a free log subscription operation binding the contract event 0x0c833c7c9f5b9b8ed0085d7959eb025f59fa32a55b8d55223a819bc7c58db345.
//
// Solidity: event NodeUnlocked(address indexed operator, bytes32 indexed nodeId, uint64 availableAt)
func (_ClearnetRegistryProtocol *ClearnetRegistryProtocolFilterer) WatchNodeUnlocked(opts *bind.WatchOpts, sink chan<- *ClearnetRegistryProtocolNodeUnlocked, operator []common.Address, nodeId [][32]byte) (event.Subscription, error) {

	var operatorRule []interface{}
	for _, operatorItem := range operator {
		operatorRule = append(operatorRule, operatorItem)
	}
	var nodeIdRule []interface{}
	for _, nodeIdItem := range nodeId {
		nodeIdRule = append(nodeIdRule, nodeIdItem)
	}

	logs, sub, err := _ClearnetRegistryProtocol.contract.WatchLogs(opts, "NodeUnlocked", operatorRule, nodeIdRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ClearnetRegistryProtocolNodeUnlocked)
				if err := _ClearnetRegistryProtocol.contract.UnpackLog(event, "NodeUnlocked", log); err != nil {
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

// ParseNodeUnlocked is a log parse operation binding the contract event 0x0c833c7c9f5b9b8ed0085d7959eb025f59fa32a55b8d55223a819bc7c58db345.
//
// Solidity: event NodeUnlocked(address indexed operator, bytes32 indexed nodeId, uint64 availableAt)
func (_ClearnetRegistryProtocol *ClearnetRegistryProtocolFilterer) ParseNodeUnlocked(log types.Log) (*ClearnetRegistryProtocolNodeUnlocked, error) {
	event := new(ClearnetRegistryProtocolNodeUnlocked)
	if err := _ClearnetRegistryProtocol.contract.UnpackLog(event, "NodeUnlocked", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

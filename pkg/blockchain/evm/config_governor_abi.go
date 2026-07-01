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

// ConfigGovernorMetaData contains all meta data concerning the ConfigGovernor contract.
var ConfigGovernorMetaData = &bind.MetaData{
	ABI: "[{\"type\":\"constructor\",\"inputs\":[{\"name\":\"config_\",\"type\":\"address\",\"internalType\":\"contractIConfig\"},{\"name\":\"initialOperators\",\"type\":\"address[]\",\"internalType\":\"address[]\"},{\"name\":\"initialThreshold\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"owner_\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"acceptOwnership\",\"inputs\":[],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"config\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"isOperator\",\"inputs\":[{\"name\":\"addr\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"operatorNonce\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"operators\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address[]\",\"internalType\":\"address[]\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"owner\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"pendingOwner\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"renounceOwnership\",\"inputs\":[],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"resetOperators\",\"inputs\":[{\"name\":\"newOperators\",\"type\":\"address[]\",\"internalType\":\"address[]\"},{\"name\":\"newThreshold\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"setConfig\",\"inputs\":[{\"name\":\"key\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"checksum\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"expectedEpoch\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"signatures\",\"type\":\"bytes[]\",\"internalType\":\"bytes[]\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"threshold\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"transferOwnership\",\"inputs\":[{\"name\":\"newOwner\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"updateOperators\",\"inputs\":[{\"name\":\"newOperators\",\"type\":\"address[]\",\"internalType\":\"address[]\"},{\"name\":\"newThreshold\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"operatorNonce_\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"signatures\",\"type\":\"bytes[]\",\"internalType\":\"bytes[]\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"event\",\"name\":\"ConfigCommitted\",\"inputs\":[{\"name\":\"key\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"},{\"name\":\"checksum\",\"type\":\"bytes32\",\"indexed\":false,\"internalType\":\"bytes32\"},{\"name\":\"newEpoch\",\"type\":\"uint64\",\"indexed\":false,\"internalType\":\"uint64\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"OperatorsUpdated\",\"inputs\":[{\"name\":\"newOperators\",\"type\":\"address[]\",\"indexed\":false,\"internalType\":\"address[]\"},{\"name\":\"newThreshold\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"OwnershipTransferStarted\",\"inputs\":[{\"name\":\"previousOwner\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"newOwner\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"OwnershipTransferred\",\"inputs\":[{\"name\":\"previousOwner\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"newOwner\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"}],\"anonymous\":false},{\"type\":\"error\",\"name\":\"BelowThreshold\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"ECDSAInvalidSignature\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"ECDSAInvalidSignatureLength\",\"inputs\":[{\"name\":\"length\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]},{\"type\":\"error\",\"name\":\"ECDSAInvalidSignatureS\",\"inputs\":[{\"name\":\"s\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}]},{\"type\":\"error\",\"name\":\"InvalidThreshold\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"NotAnOperator\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"NotEnoughOperators\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"OperatorsNotSorted\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"OwnableInvalidOwner\",\"inputs\":[{\"name\":\"owner\",\"type\":\"address\",\"internalType\":\"address\"}]},{\"type\":\"error\",\"name\":\"OwnableUnauthorizedAccount\",\"inputs\":[{\"name\":\"account\",\"type\":\"address\",\"internalType\":\"address\"}]},{\"type\":\"error\",\"name\":\"SignaturesNotOrdered\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"UnexpectedEpoch\",\"inputs\":[{\"name\":\"expected\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"actual\",\"type\":\"uint64\",\"internalType\":\"uint64\"}]},{\"type\":\"error\",\"name\":\"UnexpectedNonce\",\"inputs\":[{\"name\":\"expected\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"actual\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]},{\"type\":\"error\",\"name\":\"ZeroOperator\",\"inputs\":[]}]",
	Bin: "0x60a0604052346103d1576111a580380380610019816103d5565b9283398101906080818303126103d15780516001600160a01b03811691908281036103d15760208201516001600160401b0381116103d15782019284601f850112156103d1578351946001600160401b038611610374578560051b9460206100828188016103d5565b809881520190602082978201019283116103d157602001905b8282106103b9575050506040830151926001600160a01b03906100c0906060016103fa565b1680156103a657600180546001600160a01b03199081169091555f8054918216831781556001600160a01b03909116907f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e09080a3156102435760805280156103975780835110610388576003835110610388575f5b600554811015610189577f036b6384b5eca791c62761152d0c79bb0604c104a5fb6f4eb0703f3154bb3db08101546001600160a01b03165f908152600260205260409020805460ff19169055600101610135565b50905f5b8351811015610252576001600160a01b036101a8828661040e565b51161561024357806101e7575b6001906001600160a01b036101ca828761040e565b51165f52600260205260405f208260ff198254161790550161018d565b6001600160a01b036101f9828661040e565b51165f19820182811161022f576001600160a01b0390610219908761040e565b5116106101b557638f244e9560e01b5f5260045ffd5b634e487b7160e01b5f52601160045260245ffd5b6389961e9160e01b5f5260045ffd5b508251909291906001600160401b038111610374576801000000000000000081116103745760055481600555808210610349575b508360055f5260205f205f5b83811061032c57505050508060035560405191604083019060408452518091526060830193905f5b81811061030d577ffbfe9d8242d8f40f67fc06928a0be4790057ee8f0e228f9d84789828124be0c78580888760208301520390a1604051610d6e908161043782396080518181816103d7015261063a0152f35b82516001600160a01b03168652602095860195909201916001016102ba565b82516001600160a01b031681830155602090920191600101610292565b60055f52818060205f20019103905f5b828110610367575050610286565b5f82820155600101610359565b634e487b7160e01b5f52604160045260245ffd5b633c3db75760e01b5f5260045ffd5b63aabd5a0960e01b5f5260045ffd5b631e4fbdf760e01b5f525f60045260245ffd5b602080916103c6846103fa565b81520191019061009b565b5f80fd5b6040519190601f01601f191682016001600160401b0381118382101761037457604052565b51906001600160a01b03821682036103d157565b80518210156104225760209160051b010190565b634e487b7160e01b5f52603260045260245ffdfe6080806040526004361015610012575f80fd5b5f905f3560e01c90816342cde4e81461075f5750806362b9510c146107095780636d70f7ae146106cc578063715018a61461066957806379502c551461062557806379ba5097146105a057806382f36dbd146103765780638da5cb5b1461034f578063c0d8d832146101e6578063e30c3978146101bd578063e673df8a1461013a578063f2fde38b146100cd5763fc4f74f5146100ad575f80fd5b346100ca57806003193601126100ca576020600454604051908152f35b80fd5b50346100ca5760203660031901126100ca576100e76107a9565b6100ef6108a1565b600180546001600160a01b0319166001600160a01b0392831690811790915582549091167f38d16b8cac22d99fc7c124b9cd0de2d3fa1faef420bfe791d8c362d765e227008380a380f35b50346100ca57806003193601126100ca5760405180916020600554928381520191600582525f516020610d4e5f395f51905f52915b81811061019e5761019a856101868187038261080f565b6040519182916020835260208301906107d3565b0390f35b82546001600160a01b031684526020909301926001928301920161016f565b50346100ca57806003193601126100ca576001546040516001600160a01b039091168152602090f35b50346100ca5760803660031901126100ca576004356001600160401b03811161034b57610217903690600401610779565b6044359291602435916064356001600160401b0381116103475761023f903690600401610779565b600454918288036103305760405197602089018560608b01604083525260808a0199878a5b88811061030b575050926102f892600195928561029a816103089e9f8e60406103039f9e9d9b015203601f19810183528261080f565b5190209060405190602082019246845230604084015260a06060840152600f60c08401526e7570646174654f70657261746f727360881b60e0840152608083015260a082015260e081526102f06101008261080f565b519020610b09565b016004553691610844565b6108dc565b80f35b909b6020806001928f610323858060a01b03916107bf565b168152019d019101610264565b604487898563018af85d60e51b8352600452602452fd5b8480fd5b5080fd5b50346100ca57806003193601126100ca57546040516001600160a01b039091168152602090f35b503461053d57608036600319011261053d576004356024356044356001600160401b03811680910361053d576064356001600160401b03811161053d576103c1903690600401610779565b6040516315f1206b60e31b8152600481018690527f00000000000000000000000000000000000000000000000000000000000000006001600160a01b03169290602081602481875afa8015610532575f90610557575b6001600160401b0391501684810361054157509061048491604051602081019046825230604082015260c06060820152600960e082015268736574436f6e66696760b81b6101008201528860808201528760a08201528660c082015261010081526102f06101208261080f565b803b1561053d575f809160446040518094819363d1fd27b360e01b83528960048401528860248401525af180156105325761051c575b506001016001600160401b038111610508577fc71b358da013b9c5fb3a85ec8abed4f2e9d97ed717fda612859e6f552a8fd3fd916001600160401b036040928351928352166020820152a280f35b634e487b7160e01b84526011600452602484fd5b6105299194505f9061080f565b5f9260016104ba565b6040513d5f823e3d90fd5b5f80fd5b84633e481fd360e01b5f5260045260245260445ffd5b506020813d602011610598575b816105716020938361080f565b8101031261053d57516001600160401b038116810361053d576001600160401b0390610417565b3d9150610564565b3461053d575f36600319011261053d57600154336001600160a01b039091160361061257600180546001600160a01b03199081169091555f805433928116831782556001600160a01b0316907f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e09080a3005b63118cdaa760e01b5f523360045260245ffd5b3461053d575f36600319011261053d576040517f00000000000000000000000000000000000000000000000000000000000000006001600160a01b03168152602090f35b3461053d575f36600319011261053d576106816108a1565b600180546001600160a01b03199081169091555f80549182168155906001600160a01b03167f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e08280a3005b3461053d57602036600319011261053d576001600160a01b036106ed6107a9565b165f526002602052602060ff60405f2054166040519015158152f35b3461053d57604036600319011261053d576004356001600160401b03811161053d5761030361073f61075d923690600401610779565b91906107496108a1565b600160045401600455602435923691610844565b005b3461053d575f36600319011261053d576020906003548152f35b9181601f8401121561053d578235916001600160401b03831161053d576020808501948460051b01011161053d57565b600435906001600160a01b038216820361053d57565b35906001600160a01b038216820361053d57565b90602080835192838152019201905f5b8181106107f05750505090565b82516001600160a01b03168452602093840193909201916001016107e3565b90601f801991011681019081106001600160401b0382111761083057604052565b634e487b7160e01b5f52604160045260245ffd5b929190926001600160401b038411610830578360051b90602060405161086c8285018261080f565b809681520191810192831161053d57905b82821061088957505050565b60208091610896846107bf565b81520191019061087d565b5f546001600160a01b0316330361061257565b80518210156108c85760209160051b010190565b634e487b7160e01b5f52603260045260245ffd5b91908015610afa5780835110610aeb576003835110610aeb575f5b600554811015610938575f516020610d4e5f395f51905f528101546001600160a01b03165f908152600260205260409020805460ff191690556001016108f7565b50905f5b8351811015610a01576001600160a01b0361095782866108b4565b5116156109f25780610996575b6001906001600160a01b0361097982876108b4565b51165f52600260205260405f208260ff198254161790550161093c565b6001600160a01b036109a882866108b4565b51165f1982018281116109de576001600160a01b03906109c890876108b4565b51161061096457638f244e9560e01b5f5260045ffd5b634e487b7160e01b5f52601160045260245ffd5b6389961e9160e01b5f5260045ffd5b50919081516001600160401b03811161083057680100000000000000008111610830578060055481600555808210610abd575b50506020830160055f525f5b828110610a93575050507ffbfe9d8242d8f40f67fc06928a0be4790057ee8f0e228f9d84789828124be0c79181610a88926003556040519283926040845260408401906107d3565b9060208301520390a1565b81516001600160a01b03165f516020610d4e5f395f51905f52820155602090910190600101610a40565b035f5b818110610acf57829150610a34565b5f8382015f516020610d4e5f395f51905f520155600101610ac0565b633c3db75760e01b5f5260045ffd5b63aabd5a0960e01b5f5260045ffd5b9060035490818410610c0f575f948592835b86881015610c00578760051b840135601e198536030181121561053d578401908135916001600160401b03831161053d576020810190833603821361053d5760405190610b72601f8601601f19166020018361080f565b848252602085369201011161053d575f602085610ba496610b9b95838601378301015288610c1d565b90939193610c57565b6001600160a01b038281169116811115610bf1575f52600260205260ff60405f20541615610be257935f1981146109de576001978801970193610b1b565b63d857ba2b60e01b5f5260045ffd5b6303941dd360e21b5f5260045ffd5b509450945050905010610c0f57565b625713a160e91b5f5260045ffd5b8151919060418303610c4d57610c469250602082015190606060408401519301515f1a90610ccb565b9192909190565b50505f9160029190565b6004811015610cb75780610c69575050565b60018103610c805763f645eedf60e01b5f5260045ffd5b60028103610c9b575063fce698f760e01b5f5260045260245ffd5b600314610ca55750565b6335e2f38360e21b5f5260045260245ffd5b634e487b7160e01b5f52602160045260245ffd5b91907f7fffffffffffffffffffffffffffffff5d576e7357a4501ddfe92f46681b20a08411610d42579160209360809260ff5f9560405194855216868401526040830152606082015282805260015afa15610532575f516001600160a01b03811615610d3857905f905f90565b505f906001905f90565b5050505f916003919056fe036b6384b5eca791c62761152d0c79bb0604c104a5fb6f4eb0703f3154bb3db0",
}

// ConfigGovernorABI is the input ABI used to generate the binding from.
// Deprecated: Use ConfigGovernorMetaData.ABI instead.
var ConfigGovernorABI = ConfigGovernorMetaData.ABI

// ConfigGovernorBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use ConfigGovernorMetaData.Bin instead.
var ConfigGovernorBin = ConfigGovernorMetaData.Bin

// DeployConfigGovernor deploys a new Ethereum contract, binding an instance of ConfigGovernor to it.
func DeployConfigGovernor(auth *bind.TransactOpts, backend bind.ContractBackend, config_ common.Address, initialOperators []common.Address, initialThreshold *big.Int, owner_ common.Address) (common.Address, *types.Transaction, *ConfigGovernor, error) {
	parsed, err := ConfigGovernorMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(ConfigGovernorBin), backend, config_, initialOperators, initialThreshold, owner_)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &ConfigGovernor{ConfigGovernorCaller: ConfigGovernorCaller{contract: contract}, ConfigGovernorTransactor: ConfigGovernorTransactor{contract: contract}, ConfigGovernorFilterer: ConfigGovernorFilterer{contract: contract}}, nil
}

// ConfigGovernor is an auto generated Go binding around an Ethereum contract.
type ConfigGovernor struct {
	ConfigGovernorCaller     // Read-only binding to the contract
	ConfigGovernorTransactor // Write-only binding to the contract
	ConfigGovernorFilterer   // Log filterer for contract events
}

// ConfigGovernorCaller is an auto generated read-only Go binding around an Ethereum contract.
type ConfigGovernorCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ConfigGovernorTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ConfigGovernorTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ConfigGovernorFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type ConfigGovernorFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ConfigGovernorSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ConfigGovernorSession struct {
	Contract     *ConfigGovernor   // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// ConfigGovernorCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ConfigGovernorCallerSession struct {
	Contract *ConfigGovernorCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts         // Call options to use throughout this session
}

// ConfigGovernorTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ConfigGovernorTransactorSession struct {
	Contract     *ConfigGovernorTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts         // Transaction auth options to use throughout this session
}

// ConfigGovernorRaw is an auto generated low-level Go binding around an Ethereum contract.
type ConfigGovernorRaw struct {
	Contract *ConfigGovernor // Generic contract binding to access the raw methods on
}

// ConfigGovernorCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ConfigGovernorCallerRaw struct {
	Contract *ConfigGovernorCaller // Generic read-only contract binding to access the raw methods on
}

// ConfigGovernorTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ConfigGovernorTransactorRaw struct {
	Contract *ConfigGovernorTransactor // Generic write-only contract binding to access the raw methods on
}

// NewConfigGovernor creates a new instance of ConfigGovernor, bound to a specific deployed contract.
func NewConfigGovernor(address common.Address, backend bind.ContractBackend) (*ConfigGovernor, error) {
	contract, err := bindConfigGovernor(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &ConfigGovernor{ConfigGovernorCaller: ConfigGovernorCaller{contract: contract}, ConfigGovernorTransactor: ConfigGovernorTransactor{contract: contract}, ConfigGovernorFilterer: ConfigGovernorFilterer{contract: contract}}, nil
}

// NewConfigGovernorCaller creates a new read-only instance of ConfigGovernor, bound to a specific deployed contract.
func NewConfigGovernorCaller(address common.Address, caller bind.ContractCaller) (*ConfigGovernorCaller, error) {
	contract, err := bindConfigGovernor(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ConfigGovernorCaller{contract: contract}, nil
}

// NewConfigGovernorTransactor creates a new write-only instance of ConfigGovernor, bound to a specific deployed contract.
func NewConfigGovernorTransactor(address common.Address, transactor bind.ContractTransactor) (*ConfigGovernorTransactor, error) {
	contract, err := bindConfigGovernor(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &ConfigGovernorTransactor{contract: contract}, nil
}

// NewConfigGovernorFilterer creates a new log filterer instance of ConfigGovernor, bound to a specific deployed contract.
func NewConfigGovernorFilterer(address common.Address, filterer bind.ContractFilterer) (*ConfigGovernorFilterer, error) {
	contract, err := bindConfigGovernor(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &ConfigGovernorFilterer{contract: contract}, nil
}

// bindConfigGovernor binds a generic wrapper to an already deployed contract.
func bindConfigGovernor(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := ConfigGovernorMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ConfigGovernor *ConfigGovernorRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ConfigGovernor.Contract.ConfigGovernorCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ConfigGovernor *ConfigGovernorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ConfigGovernor.Contract.ConfigGovernorTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ConfigGovernor *ConfigGovernorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ConfigGovernor.Contract.ConfigGovernorTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ConfigGovernor *ConfigGovernorCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ConfigGovernor.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ConfigGovernor *ConfigGovernorTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ConfigGovernor.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ConfigGovernor *ConfigGovernorTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ConfigGovernor.Contract.contract.Transact(opts, method, params...)
}

// Config is a free data retrieval call binding the contract method 0x79502c55.
//
// Solidity: function config() view returns(address)
func (_ConfigGovernor *ConfigGovernorCaller) Config(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _ConfigGovernor.contract.Call(opts, &out, "config")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Config is a free data retrieval call binding the contract method 0x79502c55.
//
// Solidity: function config() view returns(address)
func (_ConfigGovernor *ConfigGovernorSession) Config() (common.Address, error) {
	return _ConfigGovernor.Contract.Config(&_ConfigGovernor.CallOpts)
}

// Config is a free data retrieval call binding the contract method 0x79502c55.
//
// Solidity: function config() view returns(address)
func (_ConfigGovernor *ConfigGovernorCallerSession) Config() (common.Address, error) {
	return _ConfigGovernor.Contract.Config(&_ConfigGovernor.CallOpts)
}

// IsOperator is a free data retrieval call binding the contract method 0x6d70f7ae.
//
// Solidity: function isOperator(address addr) view returns(bool)
func (_ConfigGovernor *ConfigGovernorCaller) IsOperator(opts *bind.CallOpts, addr common.Address) (bool, error) {
	var out []interface{}
	err := _ConfigGovernor.contract.Call(opts, &out, "isOperator", addr)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsOperator is a free data retrieval call binding the contract method 0x6d70f7ae.
//
// Solidity: function isOperator(address addr) view returns(bool)
func (_ConfigGovernor *ConfigGovernorSession) IsOperator(addr common.Address) (bool, error) {
	return _ConfigGovernor.Contract.IsOperator(&_ConfigGovernor.CallOpts, addr)
}

// IsOperator is a free data retrieval call binding the contract method 0x6d70f7ae.
//
// Solidity: function isOperator(address addr) view returns(bool)
func (_ConfigGovernor *ConfigGovernorCallerSession) IsOperator(addr common.Address) (bool, error) {
	return _ConfigGovernor.Contract.IsOperator(&_ConfigGovernor.CallOpts, addr)
}

// OperatorNonce is a free data retrieval call binding the contract method 0xfc4f74f5.
//
// Solidity: function operatorNonce() view returns(uint256)
func (_ConfigGovernor *ConfigGovernorCaller) OperatorNonce(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _ConfigGovernor.contract.Call(opts, &out, "operatorNonce")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// OperatorNonce is a free data retrieval call binding the contract method 0xfc4f74f5.
//
// Solidity: function operatorNonce() view returns(uint256)
func (_ConfigGovernor *ConfigGovernorSession) OperatorNonce() (*big.Int, error) {
	return _ConfigGovernor.Contract.OperatorNonce(&_ConfigGovernor.CallOpts)
}

// OperatorNonce is a free data retrieval call binding the contract method 0xfc4f74f5.
//
// Solidity: function operatorNonce() view returns(uint256)
func (_ConfigGovernor *ConfigGovernorCallerSession) OperatorNonce() (*big.Int, error) {
	return _ConfigGovernor.Contract.OperatorNonce(&_ConfigGovernor.CallOpts)
}

// Operators is a free data retrieval call binding the contract method 0xe673df8a.
//
// Solidity: function operators() view returns(address[])
func (_ConfigGovernor *ConfigGovernorCaller) Operators(opts *bind.CallOpts) ([]common.Address, error) {
	var out []interface{}
	err := _ConfigGovernor.contract.Call(opts, &out, "operators")

	if err != nil {
		return *new([]common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new([]common.Address)).(*[]common.Address)

	return out0, err

}

// Operators is a free data retrieval call binding the contract method 0xe673df8a.
//
// Solidity: function operators() view returns(address[])
func (_ConfigGovernor *ConfigGovernorSession) Operators() ([]common.Address, error) {
	return _ConfigGovernor.Contract.Operators(&_ConfigGovernor.CallOpts)
}

// Operators is a free data retrieval call binding the contract method 0xe673df8a.
//
// Solidity: function operators() view returns(address[])
func (_ConfigGovernor *ConfigGovernorCallerSession) Operators() ([]common.Address, error) {
	return _ConfigGovernor.Contract.Operators(&_ConfigGovernor.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_ConfigGovernor *ConfigGovernorCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _ConfigGovernor.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_ConfigGovernor *ConfigGovernorSession) Owner() (common.Address, error) {
	return _ConfigGovernor.Contract.Owner(&_ConfigGovernor.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_ConfigGovernor *ConfigGovernorCallerSession) Owner() (common.Address, error) {
	return _ConfigGovernor.Contract.Owner(&_ConfigGovernor.CallOpts)
}

// PendingOwner is a free data retrieval call binding the contract method 0xe30c3978.
//
// Solidity: function pendingOwner() view returns(address)
func (_ConfigGovernor *ConfigGovernorCaller) PendingOwner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _ConfigGovernor.contract.Call(opts, &out, "pendingOwner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// PendingOwner is a free data retrieval call binding the contract method 0xe30c3978.
//
// Solidity: function pendingOwner() view returns(address)
func (_ConfigGovernor *ConfigGovernorSession) PendingOwner() (common.Address, error) {
	return _ConfigGovernor.Contract.PendingOwner(&_ConfigGovernor.CallOpts)
}

// PendingOwner is a free data retrieval call binding the contract method 0xe30c3978.
//
// Solidity: function pendingOwner() view returns(address)
func (_ConfigGovernor *ConfigGovernorCallerSession) PendingOwner() (common.Address, error) {
	return _ConfigGovernor.Contract.PendingOwner(&_ConfigGovernor.CallOpts)
}

// Threshold is a free data retrieval call binding the contract method 0x42cde4e8.
//
// Solidity: function threshold() view returns(uint256)
func (_ConfigGovernor *ConfigGovernorCaller) Threshold(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _ConfigGovernor.contract.Call(opts, &out, "threshold")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Threshold is a free data retrieval call binding the contract method 0x42cde4e8.
//
// Solidity: function threshold() view returns(uint256)
func (_ConfigGovernor *ConfigGovernorSession) Threshold() (*big.Int, error) {
	return _ConfigGovernor.Contract.Threshold(&_ConfigGovernor.CallOpts)
}

// Threshold is a free data retrieval call binding the contract method 0x42cde4e8.
//
// Solidity: function threshold() view returns(uint256)
func (_ConfigGovernor *ConfigGovernorCallerSession) Threshold() (*big.Int, error) {
	return _ConfigGovernor.Contract.Threshold(&_ConfigGovernor.CallOpts)
}

// AcceptOwnership is a paid mutator transaction binding the contract method 0x79ba5097.
//
// Solidity: function acceptOwnership() returns()
func (_ConfigGovernor *ConfigGovernorTransactor) AcceptOwnership(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ConfigGovernor.contract.Transact(opts, "acceptOwnership")
}

// AcceptOwnership is a paid mutator transaction binding the contract method 0x79ba5097.
//
// Solidity: function acceptOwnership() returns()
func (_ConfigGovernor *ConfigGovernorSession) AcceptOwnership() (*types.Transaction, error) {
	return _ConfigGovernor.Contract.AcceptOwnership(&_ConfigGovernor.TransactOpts)
}

// AcceptOwnership is a paid mutator transaction binding the contract method 0x79ba5097.
//
// Solidity: function acceptOwnership() returns()
func (_ConfigGovernor *ConfigGovernorTransactorSession) AcceptOwnership() (*types.Transaction, error) {
	return _ConfigGovernor.Contract.AcceptOwnership(&_ConfigGovernor.TransactOpts)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_ConfigGovernor *ConfigGovernorTransactor) RenounceOwnership(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ConfigGovernor.contract.Transact(opts, "renounceOwnership")
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_ConfigGovernor *ConfigGovernorSession) RenounceOwnership() (*types.Transaction, error) {
	return _ConfigGovernor.Contract.RenounceOwnership(&_ConfigGovernor.TransactOpts)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_ConfigGovernor *ConfigGovernorTransactorSession) RenounceOwnership() (*types.Transaction, error) {
	return _ConfigGovernor.Contract.RenounceOwnership(&_ConfigGovernor.TransactOpts)
}

// ResetOperators is a paid mutator transaction binding the contract method 0x62b9510c.
//
// Solidity: function resetOperators(address[] newOperators, uint256 newThreshold) returns()
func (_ConfigGovernor *ConfigGovernorTransactor) ResetOperators(opts *bind.TransactOpts, newOperators []common.Address, newThreshold *big.Int) (*types.Transaction, error) {
	return _ConfigGovernor.contract.Transact(opts, "resetOperators", newOperators, newThreshold)
}

// ResetOperators is a paid mutator transaction binding the contract method 0x62b9510c.
//
// Solidity: function resetOperators(address[] newOperators, uint256 newThreshold) returns()
func (_ConfigGovernor *ConfigGovernorSession) ResetOperators(newOperators []common.Address, newThreshold *big.Int) (*types.Transaction, error) {
	return _ConfigGovernor.Contract.ResetOperators(&_ConfigGovernor.TransactOpts, newOperators, newThreshold)
}

// ResetOperators is a paid mutator transaction binding the contract method 0x62b9510c.
//
// Solidity: function resetOperators(address[] newOperators, uint256 newThreshold) returns()
func (_ConfigGovernor *ConfigGovernorTransactorSession) ResetOperators(newOperators []common.Address, newThreshold *big.Int) (*types.Transaction, error) {
	return _ConfigGovernor.Contract.ResetOperators(&_ConfigGovernor.TransactOpts, newOperators, newThreshold)
}

// SetConfig is a paid mutator transaction binding the contract method 0x82f36dbd.
//
// Solidity: function setConfig(bytes32 key, bytes32 checksum, uint64 expectedEpoch, bytes[] signatures) returns()
func (_ConfigGovernor *ConfigGovernorTransactor) SetConfig(opts *bind.TransactOpts, key [32]byte, checksum [32]byte, expectedEpoch uint64, signatures [][]byte) (*types.Transaction, error) {
	return _ConfigGovernor.contract.Transact(opts, "setConfig", key, checksum, expectedEpoch, signatures)
}

// SetConfig is a paid mutator transaction binding the contract method 0x82f36dbd.
//
// Solidity: function setConfig(bytes32 key, bytes32 checksum, uint64 expectedEpoch, bytes[] signatures) returns()
func (_ConfigGovernor *ConfigGovernorSession) SetConfig(key [32]byte, checksum [32]byte, expectedEpoch uint64, signatures [][]byte) (*types.Transaction, error) {
	return _ConfigGovernor.Contract.SetConfig(&_ConfigGovernor.TransactOpts, key, checksum, expectedEpoch, signatures)
}

// SetConfig is a paid mutator transaction binding the contract method 0x82f36dbd.
//
// Solidity: function setConfig(bytes32 key, bytes32 checksum, uint64 expectedEpoch, bytes[] signatures) returns()
func (_ConfigGovernor *ConfigGovernorTransactorSession) SetConfig(key [32]byte, checksum [32]byte, expectedEpoch uint64, signatures [][]byte) (*types.Transaction, error) {
	return _ConfigGovernor.Contract.SetConfig(&_ConfigGovernor.TransactOpts, key, checksum, expectedEpoch, signatures)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_ConfigGovernor *ConfigGovernorTransactor) TransferOwnership(opts *bind.TransactOpts, newOwner common.Address) (*types.Transaction, error) {
	return _ConfigGovernor.contract.Transact(opts, "transferOwnership", newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_ConfigGovernor *ConfigGovernorSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _ConfigGovernor.Contract.TransferOwnership(&_ConfigGovernor.TransactOpts, newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_ConfigGovernor *ConfigGovernorTransactorSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _ConfigGovernor.Contract.TransferOwnership(&_ConfigGovernor.TransactOpts, newOwner)
}

// UpdateOperators is a paid mutator transaction binding the contract method 0xc0d8d832.
//
// Solidity: function updateOperators(address[] newOperators, uint256 newThreshold, uint256 operatorNonce_, bytes[] signatures) returns()
func (_ConfigGovernor *ConfigGovernorTransactor) UpdateOperators(opts *bind.TransactOpts, newOperators []common.Address, newThreshold *big.Int, operatorNonce_ *big.Int, signatures [][]byte) (*types.Transaction, error) {
	return _ConfigGovernor.contract.Transact(opts, "updateOperators", newOperators, newThreshold, operatorNonce_, signatures)
}

// UpdateOperators is a paid mutator transaction binding the contract method 0xc0d8d832.
//
// Solidity: function updateOperators(address[] newOperators, uint256 newThreshold, uint256 operatorNonce_, bytes[] signatures) returns()
func (_ConfigGovernor *ConfigGovernorSession) UpdateOperators(newOperators []common.Address, newThreshold *big.Int, operatorNonce_ *big.Int, signatures [][]byte) (*types.Transaction, error) {
	return _ConfigGovernor.Contract.UpdateOperators(&_ConfigGovernor.TransactOpts, newOperators, newThreshold, operatorNonce_, signatures)
}

// UpdateOperators is a paid mutator transaction binding the contract method 0xc0d8d832.
//
// Solidity: function updateOperators(address[] newOperators, uint256 newThreshold, uint256 operatorNonce_, bytes[] signatures) returns()
func (_ConfigGovernor *ConfigGovernorTransactorSession) UpdateOperators(newOperators []common.Address, newThreshold *big.Int, operatorNonce_ *big.Int, signatures [][]byte) (*types.Transaction, error) {
	return _ConfigGovernor.Contract.UpdateOperators(&_ConfigGovernor.TransactOpts, newOperators, newThreshold, operatorNonce_, signatures)
}

// ConfigGovernorConfigCommittedIterator is returned from FilterConfigCommitted and is used to iterate over the raw logs and unpacked data for ConfigCommitted events raised by the ConfigGovernor contract.
type ConfigGovernorConfigCommittedIterator struct {
	Event *ConfigGovernorConfigCommitted // Event containing the contract specifics and raw log

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
func (it *ConfigGovernorConfigCommittedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ConfigGovernorConfigCommitted)
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
		it.Event = new(ConfigGovernorConfigCommitted)
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
func (it *ConfigGovernorConfigCommittedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ConfigGovernorConfigCommittedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ConfigGovernorConfigCommitted represents a ConfigCommitted event raised by the ConfigGovernor contract.
type ConfigGovernorConfigCommitted struct {
	Key      [32]byte
	Checksum [32]byte
	NewEpoch uint64
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterConfigCommitted is a free log retrieval operation binding the contract event 0xc71b358da013b9c5fb3a85ec8abed4f2e9d97ed717fda612859e6f552a8fd3fd.
//
// Solidity: event ConfigCommitted(bytes32 indexed key, bytes32 checksum, uint64 newEpoch)
func (_ConfigGovernor *ConfigGovernorFilterer) FilterConfigCommitted(opts *bind.FilterOpts, key [][32]byte) (*ConfigGovernorConfigCommittedIterator, error) {

	var keyRule []interface{}
	for _, keyItem := range key {
		keyRule = append(keyRule, keyItem)
	}

	logs, sub, err := _ConfigGovernor.contract.FilterLogs(opts, "ConfigCommitted", keyRule)
	if err != nil {
		return nil, err
	}
	return &ConfigGovernorConfigCommittedIterator{contract: _ConfigGovernor.contract, event: "ConfigCommitted", logs: logs, sub: sub}, nil
}

// WatchConfigCommitted is a free log subscription operation binding the contract event 0xc71b358da013b9c5fb3a85ec8abed4f2e9d97ed717fda612859e6f552a8fd3fd.
//
// Solidity: event ConfigCommitted(bytes32 indexed key, bytes32 checksum, uint64 newEpoch)
func (_ConfigGovernor *ConfigGovernorFilterer) WatchConfigCommitted(opts *bind.WatchOpts, sink chan<- *ConfigGovernorConfigCommitted, key [][32]byte) (event.Subscription, error) {

	var keyRule []interface{}
	for _, keyItem := range key {
		keyRule = append(keyRule, keyItem)
	}

	logs, sub, err := _ConfigGovernor.contract.WatchLogs(opts, "ConfigCommitted", keyRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ConfigGovernorConfigCommitted)
				if err := _ConfigGovernor.contract.UnpackLog(event, "ConfigCommitted", log); err != nil {
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

// ParseConfigCommitted is a log parse operation binding the contract event 0xc71b358da013b9c5fb3a85ec8abed4f2e9d97ed717fda612859e6f552a8fd3fd.
//
// Solidity: event ConfigCommitted(bytes32 indexed key, bytes32 checksum, uint64 newEpoch)
func (_ConfigGovernor *ConfigGovernorFilterer) ParseConfigCommitted(log types.Log) (*ConfigGovernorConfigCommitted, error) {
	event := new(ConfigGovernorConfigCommitted)
	if err := _ConfigGovernor.contract.UnpackLog(event, "ConfigCommitted", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ConfigGovernorOperatorsUpdatedIterator is returned from FilterOperatorsUpdated and is used to iterate over the raw logs and unpacked data for OperatorsUpdated events raised by the ConfigGovernor contract.
type ConfigGovernorOperatorsUpdatedIterator struct {
	Event *ConfigGovernorOperatorsUpdated // Event containing the contract specifics and raw log

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
func (it *ConfigGovernorOperatorsUpdatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ConfigGovernorOperatorsUpdated)
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
		it.Event = new(ConfigGovernorOperatorsUpdated)
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
func (it *ConfigGovernorOperatorsUpdatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ConfigGovernorOperatorsUpdatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ConfigGovernorOperatorsUpdated represents a OperatorsUpdated event raised by the ConfigGovernor contract.
type ConfigGovernorOperatorsUpdated struct {
	NewOperators []common.Address
	NewThreshold *big.Int
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterOperatorsUpdated is a free log retrieval operation binding the contract event 0xfbfe9d8242d8f40f67fc06928a0be4790057ee8f0e228f9d84789828124be0c7.
//
// Solidity: event OperatorsUpdated(address[] newOperators, uint256 newThreshold)
func (_ConfigGovernor *ConfigGovernorFilterer) FilterOperatorsUpdated(opts *bind.FilterOpts) (*ConfigGovernorOperatorsUpdatedIterator, error) {

	logs, sub, err := _ConfigGovernor.contract.FilterLogs(opts, "OperatorsUpdated")
	if err != nil {
		return nil, err
	}
	return &ConfigGovernorOperatorsUpdatedIterator{contract: _ConfigGovernor.contract, event: "OperatorsUpdated", logs: logs, sub: sub}, nil
}

// WatchOperatorsUpdated is a free log subscription operation binding the contract event 0xfbfe9d8242d8f40f67fc06928a0be4790057ee8f0e228f9d84789828124be0c7.
//
// Solidity: event OperatorsUpdated(address[] newOperators, uint256 newThreshold)
func (_ConfigGovernor *ConfigGovernorFilterer) WatchOperatorsUpdated(opts *bind.WatchOpts, sink chan<- *ConfigGovernorOperatorsUpdated) (event.Subscription, error) {

	logs, sub, err := _ConfigGovernor.contract.WatchLogs(opts, "OperatorsUpdated")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ConfigGovernorOperatorsUpdated)
				if err := _ConfigGovernor.contract.UnpackLog(event, "OperatorsUpdated", log); err != nil {
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

// ParseOperatorsUpdated is a log parse operation binding the contract event 0xfbfe9d8242d8f40f67fc06928a0be4790057ee8f0e228f9d84789828124be0c7.
//
// Solidity: event OperatorsUpdated(address[] newOperators, uint256 newThreshold)
func (_ConfigGovernor *ConfigGovernorFilterer) ParseOperatorsUpdated(log types.Log) (*ConfigGovernorOperatorsUpdated, error) {
	event := new(ConfigGovernorOperatorsUpdated)
	if err := _ConfigGovernor.contract.UnpackLog(event, "OperatorsUpdated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ConfigGovernorOwnershipTransferStartedIterator is returned from FilterOwnershipTransferStarted and is used to iterate over the raw logs and unpacked data for OwnershipTransferStarted events raised by the ConfigGovernor contract.
type ConfigGovernorOwnershipTransferStartedIterator struct {
	Event *ConfigGovernorOwnershipTransferStarted // Event containing the contract specifics and raw log

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
func (it *ConfigGovernorOwnershipTransferStartedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ConfigGovernorOwnershipTransferStarted)
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
		it.Event = new(ConfigGovernorOwnershipTransferStarted)
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
func (it *ConfigGovernorOwnershipTransferStartedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ConfigGovernorOwnershipTransferStartedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ConfigGovernorOwnershipTransferStarted represents a OwnershipTransferStarted event raised by the ConfigGovernor contract.
type ConfigGovernorOwnershipTransferStarted struct {
	PreviousOwner common.Address
	NewOwner      common.Address
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterOwnershipTransferStarted is a free log retrieval operation binding the contract event 0x38d16b8cac22d99fc7c124b9cd0de2d3fa1faef420bfe791d8c362d765e22700.
//
// Solidity: event OwnershipTransferStarted(address indexed previousOwner, address indexed newOwner)
func (_ConfigGovernor *ConfigGovernorFilterer) FilterOwnershipTransferStarted(opts *bind.FilterOpts, previousOwner []common.Address, newOwner []common.Address) (*ConfigGovernorOwnershipTransferStartedIterator, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _ConfigGovernor.contract.FilterLogs(opts, "OwnershipTransferStarted", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return &ConfigGovernorOwnershipTransferStartedIterator{contract: _ConfigGovernor.contract, event: "OwnershipTransferStarted", logs: logs, sub: sub}, nil
}

// WatchOwnershipTransferStarted is a free log subscription operation binding the contract event 0x38d16b8cac22d99fc7c124b9cd0de2d3fa1faef420bfe791d8c362d765e22700.
//
// Solidity: event OwnershipTransferStarted(address indexed previousOwner, address indexed newOwner)
func (_ConfigGovernor *ConfigGovernorFilterer) WatchOwnershipTransferStarted(opts *bind.WatchOpts, sink chan<- *ConfigGovernorOwnershipTransferStarted, previousOwner []common.Address, newOwner []common.Address) (event.Subscription, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _ConfigGovernor.contract.WatchLogs(opts, "OwnershipTransferStarted", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ConfigGovernorOwnershipTransferStarted)
				if err := _ConfigGovernor.contract.UnpackLog(event, "OwnershipTransferStarted", log); err != nil {
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
func (_ConfigGovernor *ConfigGovernorFilterer) ParseOwnershipTransferStarted(log types.Log) (*ConfigGovernorOwnershipTransferStarted, error) {
	event := new(ConfigGovernorOwnershipTransferStarted)
	if err := _ConfigGovernor.contract.UnpackLog(event, "OwnershipTransferStarted", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ConfigGovernorOwnershipTransferredIterator is returned from FilterOwnershipTransferred and is used to iterate over the raw logs and unpacked data for OwnershipTransferred events raised by the ConfigGovernor contract.
type ConfigGovernorOwnershipTransferredIterator struct {
	Event *ConfigGovernorOwnershipTransferred // Event containing the contract specifics and raw log

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
func (it *ConfigGovernorOwnershipTransferredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ConfigGovernorOwnershipTransferred)
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
		it.Event = new(ConfigGovernorOwnershipTransferred)
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
func (it *ConfigGovernorOwnershipTransferredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ConfigGovernorOwnershipTransferredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ConfigGovernorOwnershipTransferred represents a OwnershipTransferred event raised by the ConfigGovernor contract.
type ConfigGovernorOwnershipTransferred struct {
	PreviousOwner common.Address
	NewOwner      common.Address
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterOwnershipTransferred is a free log retrieval operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_ConfigGovernor *ConfigGovernorFilterer) FilterOwnershipTransferred(opts *bind.FilterOpts, previousOwner []common.Address, newOwner []common.Address) (*ConfigGovernorOwnershipTransferredIterator, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _ConfigGovernor.contract.FilterLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return &ConfigGovernorOwnershipTransferredIterator{contract: _ConfigGovernor.contract, event: "OwnershipTransferred", logs: logs, sub: sub}, nil
}

// WatchOwnershipTransferred is a free log subscription operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_ConfigGovernor *ConfigGovernorFilterer) WatchOwnershipTransferred(opts *bind.WatchOpts, sink chan<- *ConfigGovernorOwnershipTransferred, previousOwner []common.Address, newOwner []common.Address) (event.Subscription, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _ConfigGovernor.contract.WatchLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ConfigGovernorOwnershipTransferred)
				if err := _ConfigGovernor.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
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
func (_ConfigGovernor *ConfigGovernorFilterer) ParseOwnershipTransferred(log types.Log) (*ConfigGovernorOwnershipTransferred, error) {
	event := new(ConfigGovernorOwnershipTransferred)
	if err := _ConfigGovernor.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

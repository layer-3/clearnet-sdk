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

// CustodyMetaData contains all meta data concerning the Custody contract.
var CustodyMetaData = &bind.MetaData{
	ABI: "[{\"type\":\"constructor\",\"inputs\":[{\"name\":\"initialSigners\",\"type\":\"address[]\",\"internalType\":\"address[]\"},{\"name\":\"initialThreshold\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"receive\",\"stateMutability\":\"payable\"},{\"type\":\"function\",\"name\":\"deposit\",\"inputs\":[{\"name\":\"account\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"asset\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"amount\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"depositReference\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[],\"stateMutability\":\"payable\"},{\"type\":\"function\",\"name\":\"execute\",\"inputs\":[{\"name\":\"to\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"asset\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"amount\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"withdrawalId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"signatures\",\"type\":\"bytes[]\",\"internalType\":\"bytes[]\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"executed\",\"inputs\":[{\"name\":\"withdrawalId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"isSigner\",\"inputs\":[{\"name\":\"addr\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"signerNonce\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"signers\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address[]\",\"internalType\":\"address[]\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"threshold\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"updateSigners\",\"inputs\":[{\"name\":\"newSigners\",\"type\":\"address[]\",\"internalType\":\"address[]\"},{\"name\":\"newThreshold\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"signatures\",\"type\":\"bytes[]\",\"internalType\":\"bytes[]\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"event\",\"name\":\"Deposited\",\"inputs\":[{\"name\":\"account\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"depositReference\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"},{\"name\":\"depositor\",\"type\":\"address\",\"indexed\":false,\"internalType\":\"address\"},{\"name\":\"asset\",\"type\":\"address\",\"indexed\":false,\"internalType\":\"address\"},{\"name\":\"amount\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"Executed\",\"inputs\":[{\"name\":\"withdrawalId\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"},{\"name\":\"to\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"asset\",\"type\":\"address\",\"indexed\":false,\"internalType\":\"address\"},{\"name\":\"amount\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"SignersUpdated\",\"inputs\":[{\"name\":\"newSigners\",\"type\":\"address[]\",\"indexed\":false,\"internalType\":\"address[]\"},{\"name\":\"newThreshold\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"error\",\"name\":\"ECDSAInvalidSignature\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"ECDSAInvalidSignatureLength\",\"inputs\":[{\"name\":\"length\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]},{\"type\":\"error\",\"name\":\"ECDSAInvalidSignatureS\",\"inputs\":[{\"name\":\"s\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}]},{\"type\":\"error\",\"name\":\"ReentrancyGuardReentrantCall\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"SafeERC20FailedOperation\",\"inputs\":[{\"name\":\"token\",\"type\":\"address\",\"internalType\":\"address\"}]}]",
	Bin: "0x60806040523461039c5761135c80380380610019816103a0565b928339810160408282031261039c5781516001600160401b03811161039c5782019181601f8401121561039c578251926001600160401b03841161029a578360051b9260206100698186016103a0565b8096815201916020839582010191821161039c57602001915b81831061037c575050506020015160017f9b779b17422d0df92223018b32b4d1fa46e071723d6817e2486d003becc55f0055801561033757808351106102f35760038351106102ae575f5b83518110156101fc576001600160a01b036100e882866103c5565b5116156101b75780610126575b6001906001600160a01b0361010a82876103c5565b51165f528160205260405f208260ff19825416179055016100cd565b6001600160a01b0361013882866103c5565b51165f1982018281116101a3576001600160a01b039061015890876103c5565b5116106100f557606460405162461bcd60e51b815260206004820152602060248201527f5369676e657273206d75737420626520736f7274656420617363656e64696e676044820152fd5b634e487b7160e01b5f52601160045260245ffd5b60405162461bcd60e51b815260206004820152601360248201527f5a65726f2061646472657373207369676e6572000000000000000000000000006044820152606490fd5b509151906001600160401b03821161029a5768010000000000000000821161029a576004548260045580831061026f575b5060045f5260205f205f5b8381106102525784600255604051610f6e90816103ee8239f35b82516001600160a01b031681830155602090920191600101610238565b60045f52828060205f20019103905f5b82811061028d57505061022d565b5f8282015560010161027f565b634e487b7160e01b5f52604160045260245ffd5b60405162461bcd60e51b815260206004820152601760248201527f4e656564206174206c656173742033207369676e6572730000000000000000006044820152606490fd5b606460405162461bcd60e51b815260206004820152602060248201527f4e6f7420656e6f756768207369676e65727320666f72207468726573686f6c646044820152fd5b60405162461bcd60e51b815260206004820152601a60248201527f5468726573686f6c64206d75737420626520706f7369746976650000000000006044820152606490fd5b82516001600160a01b038116810361039c57815260209283019201610082565b5f80fd5b6040519190601f01601f191682016001600160401b0381118382101761029a57604052565b80518210156103d95760209160051b010190565b634e487b7160e01b5f52603260045260245ffdfe608080604052600436101561001c575b50361561001a575f80fd5b005b5f3560e01c9081630ce8d62214610a58575080630e2411ac14610631578063191d0a491461038f57806342cde4e81461037257806346f0975a146102bd5780637df73e2714610280578063a9fcfb33146102525763c98444f714610080575f61000f565b608036600319011261024e57610094610aa3565b61009c610ab9565b604435916100a8610dba565b6001600160a01b031690811561021a576100c3831515610b7b565b6001600160a01b0316918261016057803403610126575b60405192338452602084015260408301527f29856f6638b9b9b8d4e50e7b837b6bfad87b2ce76577304d1b178e02d6d9eb02606060643593a360015f516020610f4e5f395f51905f5255005b60405162461bcd60e51b815260206004820152601260248201527108aa89040ecc2d8eaca40dad2e6dac2e8c6d60731b6044820152606490fd5b346101d5576040516323b872dd60e01b5f5233600452306024528160445260205f60648180885af19060015f51148216156101b4575b6040525f6060526100da5782635274afe760e01b5f5260045260245ffd5b9060018115166101cc57843b15153d15161690610196565b503d5f823e3d90fd5b60405162461bcd60e51b815260206004820152601b60248201527f4554482073656e742077697468204552433230206465706f73697400000000006044820152606490fd5b60405162461bcd60e51b815260206004820152600c60248201526b16995c9bc81858d8dbdd5b9d60a21b6044820152606490fd5b5f80fd5b3461024e57602036600319011261024e576004355f525f602052602060ff60405f2054166040519015158152f35b3461024e57602036600319011261024e576001600160a01b036102a1610aa3565b165f526001602052602060ff60405f2054166040519015158152f35b3461024e575f36600319011261024e576040518060206004549283815201809260045f525f516020610f2e5f395f51905f52905f5b8181106103535750505081610308910382610b21565b604051918291602083019060208452518091526040830191905f5b818110610331575050500390f35b82516001600160a01b0316845285945060209384019390920191600101610323565b82546001600160a01b03168452602090930192600192830192016102f2565b3461024e575f36600319011261024e576020600254604051908152f35b3461024e5760a036600319011261024e576103a8610aa3565b6103b0610ab9565b6064359060443560843567ffffffffffffffff811161024e576103d7903690600401610a72565b90946103e1610dba565b845f525f60205260ff60405f2054166105f9576001600160a01b0381169586156105c357839261046091610416851515610b7b565b6040519660208801904682523060408a01528a60608a015260018060a01b0316978860808201528660a08201528960c082015260c0815261045860e082610b21565b519020610bd1565b845f525f60205260405f20600160ff1982541617905583155f14610537575f80809381935af13d15610532573d61049681610bb5565b906104a46040519283610b21565b81525f60203d92013e5b156104f7577fe57dd573634102b6cae74aab341f709f6fc3ae2bdc0a35f9a47a85f45b677a21916040915b82519182526020820152a360015f516020610f4e5f395f51905f5255005b60405162461bcd60e51b8152602060048201526013602482015272115512081d1c985b9cd9995c8819985a5b1959606a1b6044820152606490fd5b6104ae565b505060405163a9059cbb60e01b5f52846004528160245260205f60448180875af19060015f51148216156105ab575b60405215610598577fe57dd573634102b6cae74aab341f709f6fc3ae2bdc0a35f9a47a85f45b677a21916040916104d9565b50635274afe760e01b5f5260045260245ffd5b9060018115166101cc57833b15153d15161690610566565b60405162461bcd60e51b815260206004820152600e60248201526d16995c9bc81c9958da5c1a595b9d60921b6044820152606490fd5b60405162461bcd60e51b815260206004820152601060248201526f105b1c9958591e48195e1958dd5d195960821b6044820152606490fd5b3461024e57606036600319011261024e5760043567ffffffffffffffff811161024e57610662903690600401610a72565b906024359160443567ffffffffffffffff811161024e57610687903690600401610a72565b908415610a13578483106109cf576003831061098a5761072160019260405160208101906106c9816106bb8b8a8c87610acf565b03601f198101835282610b21565b5190209260035493604051602081019146835230604083015260a06060830152600d60c08301526c7570646174655369676e65727360981b60e083015260808201528560a082015260e0815261045861010082610b21565b016003555f5b600454811015610769575f516020610f2e5f395f51905f528101546001600160a01b03165f908152600160208190526040909120805460ff1916905501610727565b50905f5b82811061085a575067ffffffffffffffff821161084657680100000000000000008211610846578160045481600455808210610818575b50508060045f525f5b8381106107f05750506107eb837feb4dc7fab86d67670d7a4d7443a38860da1aa053f26529c8f41cc68e5d6a93369460025560405193849384610acf565b0390a1005b60019060206107fe84610b67565b930192815f516020610f2e5f395f51905f520155016107ad565b035f5b81811061082a578391506107a4565b5f8482015f516020610f2e5f395f51905f52015560010161081b565b634e487b7160e01b5f52604160045260245ffd5b6001600160a01b03610875610870838686610b43565b610b67565b161561094f57806108b5575b6001906001600160a01b0361089a610870838787610b43565b165f528160205260405f208260ff198254161790550161076d565b6108c3610870828585610b43565b5f19820182811161093b576001600160a01b03906108e690610870908787610b43565b166001600160a01b039091161161088157606460405162461bcd60e51b815260206004820152602060248201527f5369676e657273206d75737420626520736f7274656420617363656e64696e676044820152fd5b634e487b7160e01b5f52601160045260245ffd5b60405162461bcd60e51b81526020600482015260136024820152722d32b9379030b2323932b9b99039b4b3b732b960691b6044820152606490fd5b60405162461bcd60e51b815260206004820152601760248201527f4e656564206174206c656173742033207369676e6572730000000000000000006044820152606490fd5b606460405162461bcd60e51b815260206004820152602060248201527f4e6f7420656e6f756768207369676e65727320666f72207468726573686f6c646044820152fd5b60405162461bcd60e51b815260206004820152601a60248201527f5468726573686f6c64206d75737420626520706f7369746976650000000000006044820152606490fd5b3461024e575f36600319011261024e576020906003548152f35b9181601f8401121561024e5782359167ffffffffffffffff831161024e576020808501948460051b01011161024e57565b600435906001600160a01b038216820361024e57565b602435906001600160a01b038216820361024e57565b6040808252810183905293929160608501905f905b808210610af657505060209150930152565b909183356001600160a01b038116919082900361024e57908152602093840193019160010190610ae4565b90601f8019910116810190811067ffffffffffffffff82111761084657604052565b9190811015610b535760051b0190565b634e487b7160e01b5f52603260045260245ffd5b356001600160a01b038116810361024e5790565b15610b8257565b60405162461bcd60e51b815260206004820152600b60248201526a16995c9bc8185b5bdd5b9d60aa1b6044820152606490fd5b67ffffffffffffffff811161084657601f01601f191660200190565b9060025490818410610d83575f948592835b86881015610d2f578760051b840135601e198536030181121561024e5784019081359167ffffffffffffffff831161024e576020810190833603821361024e57610c2c84610bb5565b90610c3a6040519283610b21565b848252602085369201011161024e575f602085610c6c96610c6395838601378301015288610df2565b90939193610e2c565b6001600160a01b038281169116811115610cde575f52600160205260ff60405f20541615610caa57935f19811461093b576001978801970193610be3565b60405162461bcd60e51b815260206004820152600c60248201526b2737ba10309039b4b3b732b960a11b6044820152606490fd5b60405162461bcd60e51b815260206004820152602360248201527f5369676e617475726573206e6f74206f726465726564206f72206475706c696360448201526261746560e81b6064820152608490fd5b509450945050905010610d3e57565b60405162461bcd60e51b815260206004820152601d60248201527f496e73756666696369656e742076616c6964207369676e6174757265730000006044820152606490fd5b60405162461bcd60e51b815260206004820152600f60248201526e10995b1bddc81d1a1c995cda1bdb19608a1b6044820152606490fd5b60025f516020610f4e5f395f51905f525414610de35760025f516020610f4e5f395f51905f5255565b633ee5aeb560e01b5f5260045ffd5b8151919060418303610e2257610e1b9250602082015190606060408401519301515f1a90610ea0565b9192909190565b50505f9160029190565b6004811015610e8c5780610e3e575050565b60018103610e555763f645eedf60e01b5f5260045ffd5b60028103610e70575063fce698f760e01b5f5260045260245ffd5b600314610e7a5750565b6335e2f38360e21b5f5260045260245ffd5b634e487b7160e01b5f52602160045260245ffd5b91907f7fffffffffffffffffffffffffffffff5d576e7357a4501ddfe92f46681b20a08411610f22579160209360809260ff5f9560405194855216868401526040830152606082015282805260015afa15610f17575f516001600160a01b03811615610f0d57905f905f90565b505f906001905f90565b6040513d5f823e3d90fd5b5050505f916003919056fe8a35acfbc15ff81a39ae7d344fd709f28e8600b4aa8c65c6b64bfe7fe36bd19b9b779b17422d0df92223018b32b4d1fa46e071723d6817e2486d003becc55f00",
}

// CustodyABI is the input ABI used to generate the binding from.
// Deprecated: Use CustodyMetaData.ABI instead.
var CustodyABI = CustodyMetaData.ABI

// CustodyBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use CustodyMetaData.Bin instead.
var CustodyBin = CustodyMetaData.Bin

// DeployCustody deploys a new Ethereum contract, binding an instance of Custody to it.
func DeployCustody(auth *bind.TransactOpts, backend bind.ContractBackend, initialSigners []common.Address, initialThreshold *big.Int) (common.Address, *types.Transaction, *Custody, error) {
	parsed, err := CustodyMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(CustodyBin), backend, initialSigners, initialThreshold)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &Custody{CustodyCaller: CustodyCaller{contract: contract}, CustodyTransactor: CustodyTransactor{contract: contract}, CustodyFilterer: CustodyFilterer{contract: contract}}, nil
}

// Custody is an auto generated Go binding around an Ethereum contract.
type Custody struct {
	CustodyCaller     // Read-only binding to the contract
	CustodyTransactor // Write-only binding to the contract
	CustodyFilterer   // Log filterer for contract events
}

// CustodyCaller is an auto generated read-only Go binding around an Ethereum contract.
type CustodyCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// CustodyTransactor is an auto generated write-only Go binding around an Ethereum contract.
type CustodyTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// CustodyFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type CustodyFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// CustodySession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type CustodySession struct {
	Contract     *Custody          // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// CustodyCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type CustodyCallerSession struct {
	Contract *CustodyCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts  // Call options to use throughout this session
}

// CustodyTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type CustodyTransactorSession struct {
	Contract     *CustodyTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts  // Transaction auth options to use throughout this session
}

// CustodyRaw is an auto generated low-level Go binding around an Ethereum contract.
type CustodyRaw struct {
	Contract *Custody // Generic contract binding to access the raw methods on
}

// CustodyCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type CustodyCallerRaw struct {
	Contract *CustodyCaller // Generic read-only contract binding to access the raw methods on
}

// CustodyTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type CustodyTransactorRaw struct {
	Contract *CustodyTransactor // Generic write-only contract binding to access the raw methods on
}

// NewCustody creates a new instance of Custody, bound to a specific deployed contract.
func NewCustody(address common.Address, backend bind.ContractBackend) (*Custody, error) {
	contract, err := bindCustody(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Custody{CustodyCaller: CustodyCaller{contract: contract}, CustodyTransactor: CustodyTransactor{contract: contract}, CustodyFilterer: CustodyFilterer{contract: contract}}, nil
}

// NewCustodyCaller creates a new read-only instance of Custody, bound to a specific deployed contract.
func NewCustodyCaller(address common.Address, caller bind.ContractCaller) (*CustodyCaller, error) {
	contract, err := bindCustody(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &CustodyCaller{contract: contract}, nil
}

// NewCustodyTransactor creates a new write-only instance of Custody, bound to a specific deployed contract.
func NewCustodyTransactor(address common.Address, transactor bind.ContractTransactor) (*CustodyTransactor, error) {
	contract, err := bindCustody(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &CustodyTransactor{contract: contract}, nil
}

// NewCustodyFilterer creates a new log filterer instance of Custody, bound to a specific deployed contract.
func NewCustodyFilterer(address common.Address, filterer bind.ContractFilterer) (*CustodyFilterer, error) {
	contract, err := bindCustody(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &CustodyFilterer{contract: contract}, nil
}

// bindCustody binds a generic wrapper to an already deployed contract.
func bindCustody(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := CustodyMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Custody *CustodyRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Custody.Contract.CustodyCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Custody *CustodyRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Custody.Contract.CustodyTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Custody *CustodyRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Custody.Contract.CustodyTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Custody *CustodyCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Custody.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Custody *CustodyTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Custody.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Custody *CustodyTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Custody.Contract.contract.Transact(opts, method, params...)
}

// Executed is a free data retrieval call binding the contract method 0xa9fcfb33.
//
// Solidity: function executed(bytes32 withdrawalId) view returns(bool)
func (_Custody *CustodyCaller) Executed(opts *bind.CallOpts, withdrawalId [32]byte) (bool, error) {
	var out []interface{}
	err := _Custody.contract.Call(opts, &out, "executed", withdrawalId)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// Executed is a free data retrieval call binding the contract method 0xa9fcfb33.
//
// Solidity: function executed(bytes32 withdrawalId) view returns(bool)
func (_Custody *CustodySession) Executed(withdrawalId [32]byte) (bool, error) {
	return _Custody.Contract.Executed(&_Custody.CallOpts, withdrawalId)
}

// Executed is a free data retrieval call binding the contract method 0xa9fcfb33.
//
// Solidity: function executed(bytes32 withdrawalId) view returns(bool)
func (_Custody *CustodyCallerSession) Executed(withdrawalId [32]byte) (bool, error) {
	return _Custody.Contract.Executed(&_Custody.CallOpts, withdrawalId)
}

// IsSigner is a free data retrieval call binding the contract method 0x7df73e27.
//
// Solidity: function isSigner(address addr) view returns(bool)
func (_Custody *CustodyCaller) IsSigner(opts *bind.CallOpts, addr common.Address) (bool, error) {
	var out []interface{}
	err := _Custody.contract.Call(opts, &out, "isSigner", addr)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsSigner is a free data retrieval call binding the contract method 0x7df73e27.
//
// Solidity: function isSigner(address addr) view returns(bool)
func (_Custody *CustodySession) IsSigner(addr common.Address) (bool, error) {
	return _Custody.Contract.IsSigner(&_Custody.CallOpts, addr)
}

// IsSigner is a free data retrieval call binding the contract method 0x7df73e27.
//
// Solidity: function isSigner(address addr) view returns(bool)
func (_Custody *CustodyCallerSession) IsSigner(addr common.Address) (bool, error) {
	return _Custody.Contract.IsSigner(&_Custody.CallOpts, addr)
}

// SignerNonce is a free data retrieval call binding the contract method 0x0ce8d622.
//
// Solidity: function signerNonce() view returns(uint256)
func (_Custody *CustodyCaller) SignerNonce(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Custody.contract.Call(opts, &out, "signerNonce")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// SignerNonce is a free data retrieval call binding the contract method 0x0ce8d622.
//
// Solidity: function signerNonce() view returns(uint256)
func (_Custody *CustodySession) SignerNonce() (*big.Int, error) {
	return _Custody.Contract.SignerNonce(&_Custody.CallOpts)
}

// SignerNonce is a free data retrieval call binding the contract method 0x0ce8d622.
//
// Solidity: function signerNonce() view returns(uint256)
func (_Custody *CustodyCallerSession) SignerNonce() (*big.Int, error) {
	return _Custody.Contract.SignerNonce(&_Custody.CallOpts)
}

// Signers is a free data retrieval call binding the contract method 0x46f0975a.
//
// Solidity: function signers() view returns(address[])
func (_Custody *CustodyCaller) Signers(opts *bind.CallOpts) ([]common.Address, error) {
	var out []interface{}
	err := _Custody.contract.Call(opts, &out, "signers")

	if err != nil {
		return *new([]common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new([]common.Address)).(*[]common.Address)

	return out0, err

}

// Signers is a free data retrieval call binding the contract method 0x46f0975a.
//
// Solidity: function signers() view returns(address[])
func (_Custody *CustodySession) Signers() ([]common.Address, error) {
	return _Custody.Contract.Signers(&_Custody.CallOpts)
}

// Signers is a free data retrieval call binding the contract method 0x46f0975a.
//
// Solidity: function signers() view returns(address[])
func (_Custody *CustodyCallerSession) Signers() ([]common.Address, error) {
	return _Custody.Contract.Signers(&_Custody.CallOpts)
}

// Threshold is a free data retrieval call binding the contract method 0x42cde4e8.
//
// Solidity: function threshold() view returns(uint256)
func (_Custody *CustodyCaller) Threshold(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Custody.contract.Call(opts, &out, "threshold")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Threshold is a free data retrieval call binding the contract method 0x42cde4e8.
//
// Solidity: function threshold() view returns(uint256)
func (_Custody *CustodySession) Threshold() (*big.Int, error) {
	return _Custody.Contract.Threshold(&_Custody.CallOpts)
}

// Threshold is a free data retrieval call binding the contract method 0x42cde4e8.
//
// Solidity: function threshold() view returns(uint256)
func (_Custody *CustodyCallerSession) Threshold() (*big.Int, error) {
	return _Custody.Contract.Threshold(&_Custody.CallOpts)
}

// Deposit is a paid mutator transaction binding the contract method 0xc98444f7.
//
// Solidity: function deposit(address account, address asset, uint256 amount, bytes32 depositReference) payable returns()
func (_Custody *CustodyTransactor) Deposit(opts *bind.TransactOpts, account common.Address, asset common.Address, amount *big.Int, depositReference [32]byte) (*types.Transaction, error) {
	return _Custody.contract.Transact(opts, "deposit", account, asset, amount, depositReference)
}

// Deposit is a paid mutator transaction binding the contract method 0xc98444f7.
//
// Solidity: function deposit(address account, address asset, uint256 amount, bytes32 depositReference) payable returns()
func (_Custody *CustodySession) Deposit(account common.Address, asset common.Address, amount *big.Int, depositReference [32]byte) (*types.Transaction, error) {
	return _Custody.Contract.Deposit(&_Custody.TransactOpts, account, asset, amount, depositReference)
}

// Deposit is a paid mutator transaction binding the contract method 0xc98444f7.
//
// Solidity: function deposit(address account, address asset, uint256 amount, bytes32 depositReference) payable returns()
func (_Custody *CustodyTransactorSession) Deposit(account common.Address, asset common.Address, amount *big.Int, depositReference [32]byte) (*types.Transaction, error) {
	return _Custody.Contract.Deposit(&_Custody.TransactOpts, account, asset, amount, depositReference)
}

// Execute is a paid mutator transaction binding the contract method 0x191d0a49.
//
// Solidity: function execute(address to, address asset, uint256 amount, bytes32 withdrawalId, bytes[] signatures) returns()
func (_Custody *CustodyTransactor) Execute(opts *bind.TransactOpts, to common.Address, asset common.Address, amount *big.Int, withdrawalId [32]byte, signatures [][]byte) (*types.Transaction, error) {
	return _Custody.contract.Transact(opts, "execute", to, asset, amount, withdrawalId, signatures)
}

// Execute is a paid mutator transaction binding the contract method 0x191d0a49.
//
// Solidity: function execute(address to, address asset, uint256 amount, bytes32 withdrawalId, bytes[] signatures) returns()
func (_Custody *CustodySession) Execute(to common.Address, asset common.Address, amount *big.Int, withdrawalId [32]byte, signatures [][]byte) (*types.Transaction, error) {
	return _Custody.Contract.Execute(&_Custody.TransactOpts, to, asset, amount, withdrawalId, signatures)
}

// Execute is a paid mutator transaction binding the contract method 0x191d0a49.
//
// Solidity: function execute(address to, address asset, uint256 amount, bytes32 withdrawalId, bytes[] signatures) returns()
func (_Custody *CustodyTransactorSession) Execute(to common.Address, asset common.Address, amount *big.Int, withdrawalId [32]byte, signatures [][]byte) (*types.Transaction, error) {
	return _Custody.Contract.Execute(&_Custody.TransactOpts, to, asset, amount, withdrawalId, signatures)
}

// UpdateSigners is a paid mutator transaction binding the contract method 0x0e2411ac.
//
// Solidity: function updateSigners(address[] newSigners, uint256 newThreshold, bytes[] signatures) returns()
func (_Custody *CustodyTransactor) UpdateSigners(opts *bind.TransactOpts, newSigners []common.Address, newThreshold *big.Int, signatures [][]byte) (*types.Transaction, error) {
	return _Custody.contract.Transact(opts, "updateSigners", newSigners, newThreshold, signatures)
}

// UpdateSigners is a paid mutator transaction binding the contract method 0x0e2411ac.
//
// Solidity: function updateSigners(address[] newSigners, uint256 newThreshold, bytes[] signatures) returns()
func (_Custody *CustodySession) UpdateSigners(newSigners []common.Address, newThreshold *big.Int, signatures [][]byte) (*types.Transaction, error) {
	return _Custody.Contract.UpdateSigners(&_Custody.TransactOpts, newSigners, newThreshold, signatures)
}

// UpdateSigners is a paid mutator transaction binding the contract method 0x0e2411ac.
//
// Solidity: function updateSigners(address[] newSigners, uint256 newThreshold, bytes[] signatures) returns()
func (_Custody *CustodyTransactorSession) UpdateSigners(newSigners []common.Address, newThreshold *big.Int, signatures [][]byte) (*types.Transaction, error) {
	return _Custody.Contract.UpdateSigners(&_Custody.TransactOpts, newSigners, newThreshold, signatures)
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_Custody *CustodyTransactor) Receive(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Custody.contract.RawTransact(opts, nil) // calldata is disallowed for receive function
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_Custody *CustodySession) Receive() (*types.Transaction, error) {
	return _Custody.Contract.Receive(&_Custody.TransactOpts)
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_Custody *CustodyTransactorSession) Receive() (*types.Transaction, error) {
	return _Custody.Contract.Receive(&_Custody.TransactOpts)
}

// CustodyDepositedIterator is returned from FilterDeposited and is used to iterate over the raw logs and unpacked data for Deposited events raised by the Custody contract.
type CustodyDepositedIterator struct {
	Event *CustodyDeposited // Event containing the contract specifics and raw log

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
func (it *CustodyDepositedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(CustodyDeposited)
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
		it.Event = new(CustodyDeposited)
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
func (it *CustodyDepositedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *CustodyDepositedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// CustodyDeposited represents a Deposited event raised by the Custody contract.
type CustodyDeposited struct {
	Account          common.Address
	DepositReference [32]byte
	Depositor        common.Address
	Asset            common.Address
	Amount           *big.Int
	Raw              types.Log // Blockchain specific contextual infos
}

// FilterDeposited is a free log retrieval operation binding the contract event 0x29856f6638b9b9b8d4e50e7b837b6bfad87b2ce76577304d1b178e02d6d9eb02.
//
// Solidity: event Deposited(address indexed account, bytes32 indexed depositReference, address depositor, address asset, uint256 amount)
func (_Custody *CustodyFilterer) FilterDeposited(opts *bind.FilterOpts, account []common.Address, depositReference [][32]byte) (*CustodyDepositedIterator, error) {

	var accountRule []interface{}
	for _, accountItem := range account {
		accountRule = append(accountRule, accountItem)
	}
	var depositReferenceRule []interface{}
	for _, depositReferenceItem := range depositReference {
		depositReferenceRule = append(depositReferenceRule, depositReferenceItem)
	}

	logs, sub, err := _Custody.contract.FilterLogs(opts, "Deposited", accountRule, depositReferenceRule)
	if err != nil {
		return nil, err
	}
	return &CustodyDepositedIterator{contract: _Custody.contract, event: "Deposited", logs: logs, sub: sub}, nil
}

// WatchDeposited is a free log subscription operation binding the contract event 0x29856f6638b9b9b8d4e50e7b837b6bfad87b2ce76577304d1b178e02d6d9eb02.
//
// Solidity: event Deposited(address indexed account, bytes32 indexed depositReference, address depositor, address asset, uint256 amount)
func (_Custody *CustodyFilterer) WatchDeposited(opts *bind.WatchOpts, sink chan<- *CustodyDeposited, account []common.Address, depositReference [][32]byte) (event.Subscription, error) {

	var accountRule []interface{}
	for _, accountItem := range account {
		accountRule = append(accountRule, accountItem)
	}
	var depositReferenceRule []interface{}
	for _, depositReferenceItem := range depositReference {
		depositReferenceRule = append(depositReferenceRule, depositReferenceItem)
	}

	logs, sub, err := _Custody.contract.WatchLogs(opts, "Deposited", accountRule, depositReferenceRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(CustodyDeposited)
				if err := _Custody.contract.UnpackLog(event, "Deposited", log); err != nil {
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

// ParseDeposited is a log parse operation binding the contract event 0x29856f6638b9b9b8d4e50e7b837b6bfad87b2ce76577304d1b178e02d6d9eb02.
//
// Solidity: event Deposited(address indexed account, bytes32 indexed depositReference, address depositor, address asset, uint256 amount)
func (_Custody *CustodyFilterer) ParseDeposited(log types.Log) (*CustodyDeposited, error) {
	event := new(CustodyDeposited)
	if err := _Custody.contract.UnpackLog(event, "Deposited", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// CustodyExecutedIterator is returned from FilterExecuted and is used to iterate over the raw logs and unpacked data for Executed events raised by the Custody contract.
type CustodyExecutedIterator struct {
	Event *CustodyExecuted // Event containing the contract specifics and raw log

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
func (it *CustodyExecutedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(CustodyExecuted)
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
		it.Event = new(CustodyExecuted)
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
func (it *CustodyExecutedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *CustodyExecutedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// CustodyExecuted represents a Executed event raised by the Custody contract.
type CustodyExecuted struct {
	WithdrawalId [32]byte
	To           common.Address
	Asset        common.Address
	Amount       *big.Int
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterExecuted is a free log retrieval operation binding the contract event 0xe57dd573634102b6cae74aab341f709f6fc3ae2bdc0a35f9a47a85f45b677a21.
//
// Solidity: event Executed(bytes32 indexed withdrawalId, address indexed to, address asset, uint256 amount)
func (_Custody *CustodyFilterer) FilterExecuted(opts *bind.FilterOpts, withdrawalId [][32]byte, to []common.Address) (*CustodyExecutedIterator, error) {

	var withdrawalIdRule []interface{}
	for _, withdrawalIdItem := range withdrawalId {
		withdrawalIdRule = append(withdrawalIdRule, withdrawalIdItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _Custody.contract.FilterLogs(opts, "Executed", withdrawalIdRule, toRule)
	if err != nil {
		return nil, err
	}
	return &CustodyExecutedIterator{contract: _Custody.contract, event: "Executed", logs: logs, sub: sub}, nil
}

// WatchExecuted is a free log subscription operation binding the contract event 0xe57dd573634102b6cae74aab341f709f6fc3ae2bdc0a35f9a47a85f45b677a21.
//
// Solidity: event Executed(bytes32 indexed withdrawalId, address indexed to, address asset, uint256 amount)
func (_Custody *CustodyFilterer) WatchExecuted(opts *bind.WatchOpts, sink chan<- *CustodyExecuted, withdrawalId [][32]byte, to []common.Address) (event.Subscription, error) {

	var withdrawalIdRule []interface{}
	for _, withdrawalIdItem := range withdrawalId {
		withdrawalIdRule = append(withdrawalIdRule, withdrawalIdItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _Custody.contract.WatchLogs(opts, "Executed", withdrawalIdRule, toRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(CustodyExecuted)
				if err := _Custody.contract.UnpackLog(event, "Executed", log); err != nil {
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

// ParseExecuted is a log parse operation binding the contract event 0xe57dd573634102b6cae74aab341f709f6fc3ae2bdc0a35f9a47a85f45b677a21.
//
// Solidity: event Executed(bytes32 indexed withdrawalId, address indexed to, address asset, uint256 amount)
func (_Custody *CustodyFilterer) ParseExecuted(log types.Log) (*CustodyExecuted, error) {
	event := new(CustodyExecuted)
	if err := _Custody.contract.UnpackLog(event, "Executed", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// CustodySignersUpdatedIterator is returned from FilterSignersUpdated and is used to iterate over the raw logs and unpacked data for SignersUpdated events raised by the Custody contract.
type CustodySignersUpdatedIterator struct {
	Event *CustodySignersUpdated // Event containing the contract specifics and raw log

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
func (it *CustodySignersUpdatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(CustodySignersUpdated)
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
		it.Event = new(CustodySignersUpdated)
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
func (it *CustodySignersUpdatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *CustodySignersUpdatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// CustodySignersUpdated represents a SignersUpdated event raised by the Custody contract.
type CustodySignersUpdated struct {
	NewSigners   []common.Address
	NewThreshold *big.Int
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterSignersUpdated is a free log retrieval operation binding the contract event 0xeb4dc7fab86d67670d7a4d7443a38860da1aa053f26529c8f41cc68e5d6a9336.
//
// Solidity: event SignersUpdated(address[] newSigners, uint256 newThreshold)
func (_Custody *CustodyFilterer) FilterSignersUpdated(opts *bind.FilterOpts) (*CustodySignersUpdatedIterator, error) {

	logs, sub, err := _Custody.contract.FilterLogs(opts, "SignersUpdated")
	if err != nil {
		return nil, err
	}
	return &CustodySignersUpdatedIterator{contract: _Custody.contract, event: "SignersUpdated", logs: logs, sub: sub}, nil
}

// WatchSignersUpdated is a free log subscription operation binding the contract event 0xeb4dc7fab86d67670d7a4d7443a38860da1aa053f26529c8f41cc68e5d6a9336.
//
// Solidity: event SignersUpdated(address[] newSigners, uint256 newThreshold)
func (_Custody *CustodyFilterer) WatchSignersUpdated(opts *bind.WatchOpts, sink chan<- *CustodySignersUpdated) (event.Subscription, error) {

	logs, sub, err := _Custody.contract.WatchLogs(opts, "SignersUpdated")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(CustodySignersUpdated)
				if err := _Custody.contract.UnpackLog(event, "SignersUpdated", log); err != nil {
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

// ParseSignersUpdated is a log parse operation binding the contract event 0xeb4dc7fab86d67670d7a4d7443a38860da1aa053f26529c8f41cc68e5d6a9336.
//
// Solidity: event SignersUpdated(address[] newSigners, uint256 newThreshold)
func (_Custody *CustodyFilterer) ParseSignersUpdated(log types.Log) (*CustodySignersUpdated, error) {
	event := new(CustodySignersUpdated)
	if err := _Custody.contract.UnpackLog(event, "SignersUpdated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

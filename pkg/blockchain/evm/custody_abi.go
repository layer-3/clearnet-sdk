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
	ABI: "[{\"type\":\"constructor\",\"inputs\":[{\"name\":\"initialSigners\",\"type\":\"address[]\",\"internalType\":\"address[]\"},{\"name\":\"initialThreshold\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"receive\",\"stateMutability\":\"payable\"},{\"type\":\"function\",\"name\":\"WITHDRAWAL_EXECUTION_WINDOW\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"deposit\",\"inputs\":[{\"name\":\"account\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"asset\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"amount\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"depositReference\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[],\"stateMutability\":\"payable\"},{\"type\":\"function\",\"name\":\"eip712Domain\",\"inputs\":[],\"outputs\":[{\"name\":\"fields\",\"type\":\"bytes1\",\"internalType\":\"bytes1\"},{\"name\":\"name\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"version\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"chainId\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"verifyingContract\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"salt\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"extensions\",\"type\":\"uint256[]\",\"internalType\":\"uint256[]\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"execute\",\"inputs\":[{\"name\":\"to\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"asset\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"amount\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"withdrawalId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"finalizedAt\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"authorizationRotationNonce\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"signatures\",\"type\":\"bytes[]\",\"internalType\":\"bytes[]\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"executed\",\"inputs\":[{\"name\":\"withdrawalId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"isSigner\",\"inputs\":[{\"name\":\"addr\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"rotationNonce\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"signers\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address[]\",\"internalType\":\"address[]\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"threshold\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"updateSigners\",\"inputs\":[{\"name\":\"newSigners\",\"type\":\"address[]\",\"internalType\":\"address[]\"},{\"name\":\"newThreshold\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"signatures\",\"type\":\"bytes[]\",\"internalType\":\"bytes[]\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"event\",\"name\":\"Deposited\",\"inputs\":[{\"name\":\"account\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"depositReference\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"},{\"name\":\"depositor\",\"type\":\"address\",\"indexed\":false,\"internalType\":\"address\"},{\"name\":\"asset\",\"type\":\"address\",\"indexed\":false,\"internalType\":\"address\"},{\"name\":\"amount\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"EIP712DomainChanged\",\"inputs\":[],\"anonymous\":false},{\"type\":\"event\",\"name\":\"Executed\",\"inputs\":[{\"name\":\"withdrawalId\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"},{\"name\":\"to\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"asset\",\"type\":\"address\",\"indexed\":false,\"internalType\":\"address\"},{\"name\":\"amount\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"SignersUpdated\",\"inputs\":[{\"name\":\"newSigners\",\"type\":\"address[]\",\"indexed\":false,\"internalType\":\"address[]\"},{\"name\":\"newThreshold\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"error\",\"name\":\"ECDSAInvalidSignature\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"ECDSAInvalidSignatureLength\",\"inputs\":[{\"name\":\"length\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]},{\"type\":\"error\",\"name\":\"ECDSAInvalidSignatureS\",\"inputs\":[{\"name\":\"s\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}]},{\"type\":\"error\",\"name\":\"InvalidShortString\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"ReentrancyGuardReentrantCall\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"SafeERC20FailedOperation\",\"inputs\":[{\"name\":\"token\",\"type\":\"address\",\"internalType\":\"address\"}]},{\"type\":\"error\",\"name\":\"StringTooLong\",\"inputs\":[{\"name\":\"str\",\"type\":\"string\",\"internalType\":\"string\"}]}]",
	Bin: "0x610160806040523461052e57611e0f803803809161001d8285610532565b833981019060408183031261052e5780516001600160401b03811161052e5781019082601f8301121561052e578151916001600160401b0383116103e7578260051b93604051936100716020870186610532565b845260208401916020839682010191821161052e57602001915b81831061050e57505050602001516040516100a7604082610532565b600d815260208101906c59656c6c6f77437573746f647960981b8252604051916100d2604084610532565b600183526020830191603160f81b835260017f9b779b17422d0df92223018b32b4d1fa46e071723d6817e2486d003becc55f005561010f81610598565b6101205261011c8461072d565b61014052519020918260e05251902080610100524660a0526040519060208201927f8b73c3c69bb8fe3d512ecc4cf759cc79239f7b179b0ffacaa9a75d522b39400f8452604083015260608201524660808201523060a082015260a0815261018560c082610532565b5190206080523060c05280156104c9578082511061048557600382511061044057815160011c8111156103fb575f5b82518110156102ba576001600160a01b036101cf8285610555565b51161561027557806101e4575b6001016101b4565b6001600160a01b036101f68285610555565b51165f198201828111610261576001600160a01b03906102169086610555565b5116106101dc57606460405162461bcd60e51b815260206004820152602060248201527f5369676e657273206d75737420626520736f7274656420617363656e64696e676044820152fd5b634e487b7160e01b5f52601160045260245ffd5b60405162461bcd60e51b815260206004820152601360248201527f5a65726f2061646472657373207369676e6572000000000000000000000000006044820152606490fd5b5091905f5b81518110156102fa576001906001600160a01b036102dd8285610555565b51165f52600360205260405f208260ff19825416179055016102bf565b5051906001600160401b0382116103e7576801000000000000000082116103e757600654826006558083106103ae575b5060065f5260205f205f5b83811061039157846004556040516115aa9081610865823960805181611155015260a05181611212015260c0518161111f015260e051816111a4015261010051816111ca015261012051816102c2015261014051816102eb0152f35b82516001600160a01b031681830155602090920191600101610335565b60065f526103e1908390037ff652222313e28459528d920b65115c16c04f3efc82aaedc97be59f3f377c0d3f840161057d565b5f61032a565b634e487b7160e01b5f52604160045260245ffd5b60405162461bcd60e51b815260206004820152601a60248201527f5468726573686f6c64206d757374206265206d616a6f726974790000000000006044820152606490fd5b60405162461bcd60e51b815260206004820152601760248201527f4e656564206174206c656173742033207369676e6572730000000000000000006044820152606490fd5b606460405162461bcd60e51b815260206004820152602060248201527f4e6f7420656e6f756768207369676e65727320666f72207468726573686f6c646044820152fd5b60405162461bcd60e51b815260206004820152601a60248201527f5468726573686f6c64206d75737420626520706f7369746976650000000000006044820152606490fd5b82516001600160a01b038116810361052e5781526020928301920161008b565b5f80fd5b601f909101601f19168101906001600160401b038211908210176103e757604052565b80518210156105695760209160051b010190565b634e487b7160e01b5f52603260045260245ffd5b5f5b82811061058b57505050565b5f8282015560010161057f565b908151602081105f14610612575090601f8151116105d25760208151910151602082106105c3571790565b5f198260200360031b1b161790565b604460209160405192839163305a27a960e01b83528160048401528051918291826024860152018484015e5f828201840152601f01601f19168101030190fd5b6001600160401b0381116103e7575f54600181811c91168015610723575b602082101461070f57601f81116106de575b50602092601f821160011461067f57928192935f92610674575b50508160011b915f199060031b1c1916175f5560ff90565b015190505f8061065c565b601f198216935f8052805f20915f5b8681106106c657508360019596106106ae575b505050811b015f5560ff90565b01515f1960f88460031b161c191690555f80806106a1565b9192602060018192868501518155019401920161068e565b8181111561064257610709905f8052601f60205f209181850160051c9182910160051c03910161057d565b5f610642565b634e487b7160e01b5f52602260045260245ffd5b90607f1690610630565b908151602081105f14610758575090601f8151116105d25760208151910151602082106105c3571790565b6001600160401b0381116103e757600154600181811c9116801561085a575b602082101461070f57601f8111610828575b50602092601f82116001146107c757928192935f926107bc575b50508160011b915f199060031b1c19161760015560ff90565b015190505f806107a3565b601f1982169360015f52805f20915f5b86811061081057508360019596106107f8575b505050811b0160015560ff90565b01515f1960f88460031b161c191690555f80806107ea565b919260206001819286850151815501940192016107d7565b81811115610789576108549060015f52601f60205f209181850160051c9182910160051c03910161057d565b5f610789565b90607f169061077756fe6080604052600436101561001a575b3615610018575f80fd5b005b5f3560e01c80630e2411ac14610841578063289381ff1461082557806342cde4e81461080857806346f0975a146107535780634f0b83a6146103df5780637df73e27146103a257806384b0196e146102aa578063946030711461028d578063a9fcfb331461025e5763c98444f70361000e57608036600319011261025a576100a0610d5e565b6100a8610d74565b604435916100b46110e4565b6001600160a01b0316908115610226576100cf831515610e57565b6001600160a01b0316918261016c57803403610132575b60405192338452602084015260408301527f29856f6638b9b9b8d4e50e7b837b6bfad87b2ce76577304d1b178e02d6d9eb02606060643593a360015f51602061158a5f395f51905f5255005b60405162461bcd60e51b815260206004820152601260248201527108aa89040ecc2d8eaca40dad2e6dac2e8c6d60731b6044820152606490fd5b346101e1576040516323b872dd60e01b5f5233600452306024528160445260205f60648180885af19060015f51148216156101c0575b6040525f6060526100e65782635274afe760e01b5f5260045260245ffd5b9060018115166101d857843b15153d151616906101a2565b503d5f823e3d90fd5b60405162461bcd60e51b815260206004820152601b60248201527f4554482073656e742077697468204552433230206465706f73697400000000006044820152606490fd5b60405162461bcd60e51b815260206004820152600c60248201526b16995c9bc81858d8dbdd5b9d60a21b6044820152606490fd5b5f80fd5b3461025a57602036600319011261025a576004355f526002602052602060ff60405f2054166040519015158152f35b3461025a575f36600319011261025a576020600554604051908152f35b3461025a575f36600319011261025a576103466102e67f0000000000000000000000000000000000000000000000000000000000000000611238565b61030f7f000000000000000000000000000000000000000000000000000000000000000061135e565b6020610354604051926103228385610dc2565b5f84525f368137604051958695600f60f81b875260e08588015260e0870190610d9e565b908582036040870152610d9e565b4660608501523060808501525f60a085015283810360c08501528180845192838152019301915f5b82811061038b57505050500390f35b83518552869550938101939281019260010161037c565b3461025a57602036600319011261025a576001600160a01b036103c3610d5e565b165f526003602052602060ff60405f2054166040519015158152f35b3461025a5760e036600319011261025a576103f8610d5e565b610400610d74565b9060443590606435916084359160a4359060c43567ffffffffffffffff811161025a57610431903690600401610d2d565b61043c9791976110e4565b865f52600260205260ff60405f20541661071b576001600160a01b0383169788156106e55761046c861515610e57565b60055485036106a957610e108701808811610695574211610666576104ff6105049387966040519060208201927f0e3df6662f35c96c24968304bfabb07b61043bb33a6dd6c2286d4bf96d1aab8284528d604084015260018060a01b03169a8b60608401528960808401528c60a084015260c083015260e082015260e081526104f761010082610dc2565b519020610eb9565b610efb565b5f858152600260205260409020805460ff19166001179055836105da575f80809381935af13d156105d5573d61053981610edf565b906105476040519283610dc2565b81525f60203d92013e5b1561059a577fe57dd573634102b6cae74aab341f709f6fc3ae2bdc0a35f9a47a85f45b677a21916040915b82519182526020820152a360015f51602061158a5f395f51905f5255005b60405162461bcd60e51b8152602060048201526013602482015272115512081d1c985b9cd9995c8819985a5b1959606a1b6044820152606490fd5b610551565b505060405163a9059cbb60e01b5f52846004528160245260205f60448180875af19060015f511482161561064e575b6040521561063b577fe57dd573634102b6cae74aab341f709f6fc3ae2bdc0a35f9a47a85f45b677a219160409161057c565b50635274afe760e01b5f5260045260245ffd5b9060018115166101d857833b15153d15161690610609565b60405162461bcd60e51b8152602060048201526007602482015266115e1c1a5c995960ca1b6044820152606490fd5b634e487b7160e01b5f52601160045260245ffd5b60405162461bcd60e51b81526020600482015260146024820152735374616c6520726f746174696f6e206e6f6e636560601b6044820152606490fd5b60405162461bcd60e51b815260206004820152600e60248201526d16995c9bc81c9958da5c1a595b9d60921b6044820152606490fd5b60405162461bcd60e51b815260206004820152601060248201526f105b1c9958591e48195e1958dd5d195960821b6044820152606490fd5b3461025a575f36600319011261025a576040518060206006549283815201809260065f525f51602061156a5f395f51905f52905f5b8181106107e9575050508161079e910382610dc2565b604051918291602083019060208452518091526040830191905f5b8181106107c7575050500390f35b82516001600160a01b03168452859450602093840193909201916001016107b9565b82546001600160a01b0316845260209093019260019283019201610788565b3461025a575f36600319011261025a576020600454604051908152f35b3461025a575f36600319011261025a576020604051610e108152f35b3461025a57606036600319011261025a5760043567ffffffffffffffff811161025a57610872903690600401610d2d565b906024359060443567ffffffffffffffff811161025a57610897903690600401610d2d565b9390926108a5368385610de4565b938115610ce85781855110610ca4576003855110610c5f57845160011c821115610c1a575f5b85518110156109b3576001600160a01b036108e68288610e91565b51161561097857806108fb575b6001016108cb565b6001600160a01b0361090d8288610e91565b51165f198201828111610695576001600160a01b039061092d9089610e91565b5116106108f357606460405162461bcd60e51b815260206004820152602060248201527f5369676e657273206d75737420626520736f7274656420617363656e64696e676044820152fd5b60405162461bcd60e51b81526020600482015260136024820152722d32b9379030b2323932b9b99039b4b3b732b960691b6044820152606490fd5b5085600554916109c4368688610de4565b60405160208101918260208251919201905f5b818110610bfb57505050600195949282610a01610a4f96946104ff9403601f198101835282610dc2565b51902060405160208101917f41b5acd9db878ee7df5936598a98288cc0a088122870112af58f2186fa65a07483526040820152886060820152866080820152608081526104f760a082610dc2565b016005555f5b600654811015610a96575f51602061156a5f395f51905f528101546001600160a01b03165f908152600360205260409020805460ff19169055600101610a55565b50905f5b81811015610ad8576001906001600160a01b03610abc600583901b8701610e43565b165f52600360205260405f208260ff1982541617905501610a9a565b509067ffffffffffffffff8211610be757680100000000000000008211610be7578160065481600655808210610bb9575b50508260065f525f5b838110610b915750508060045560405191806040840160408552526060830193905f5b818110610b6b577feb4dc7fab86d67670d7a4d7443a38860da1aa053f26529c8f41cc68e5d6a93368580888760208301520390a1005b909194602080600192838060a01b03610b838a610d8a565b168152019601929101610b35565b6001906020610b9f84610e43565b930192815f51602061156a5f395f51905f52015501610b12565b035f5b818110610bcb57839150610b09565b5f8482015f51602061156a5f395f51905f520155600101610bbc565b634e487b7160e01b5f52604160045260245ffd5b82516001600160a01b03168452602093840193909201916001016109d7565b60405162461bcd60e51b815260206004820152601a60248201527f5468726573686f6c64206d757374206265206d616a6f726974790000000000006044820152606490fd5b60405162461bcd60e51b815260206004820152601760248201527f4e656564206174206c656173742033207369676e6572730000000000000000006044820152606490fd5b606460405162461bcd60e51b815260206004820152602060248201527f4e6f7420656e6f756768207369676e65727320666f72207468726573686f6c646044820152fd5b60405162461bcd60e51b815260206004820152601a60248201527f5468726573686f6c64206d75737420626520706f7369746976650000000000006044820152606490fd5b9181601f8401121561025a5782359167ffffffffffffffff831161025a576020808501948460051b01011161025a57565b600435906001600160a01b038216820361025a57565b602435906001600160a01b038216820361025a57565b35906001600160a01b038216820361025a57565b805180835260209291819084018484015e5f828201840152601f01601f1916010190565b90601f8019910116810190811067ffffffffffffffff821117610be757604052565b9291909267ffffffffffffffff8411610be7578360051b906020808301610e0e6040519182610dc2565b809681520191810192831161025a57905b828210610e2b57505050565b60208091610e3884610d8a565b815201910190610e1f565b356001600160a01b038116810361025a5790565b15610e5e57565b60405162461bcd60e51b815260206004820152600b60248201526a16995c9bc8185b5bdd5b9d60aa1b6044820152606490fd5b8051821015610ea55760209160051b010190565b634e487b7160e01b5f52603260045260245ffd5b604290610ec461111c565b906040519161190160f01b8352600283015260228201522090565b67ffffffffffffffff8111610be757601f01601f191660200190565b90600454908184106110ad575f948592835b86881015611059578760051b840135601e198536030181121561025a5784019081359167ffffffffffffffff831161025a576020810190833603821361025a57610f5684610edf565b90610f646040519283610dc2565b848252602085369201011161025a575f602085610f9696610f8d9583860137830101528861142e565b90939193611468565b6001600160a01b038281169116811115611008575f52600360205260ff60405f20541615610fd457935f198114610695576001978801970193610f0d565b60405162461bcd60e51b815260206004820152600c60248201526b2737ba10309039b4b3b732b960a11b6044820152606490fd5b60405162461bcd60e51b815260206004820152602360248201527f5369676e617475726573206e6f74206f726465726564206f72206475706c696360448201526261746560e81b6064820152608490fd5b50945094505090501061106857565b60405162461bcd60e51b815260206004820152601d60248201527f496e73756666696369656e742076616c6964207369676e6174757265730000006044820152606490fd5b60405162461bcd60e51b815260206004820152600f60248201526e10995b1bddc81d1a1c995cda1bdb19608a1b6044820152606490fd5b60025f51602061158a5f395f51905f52541461110d5760025f51602061158a5f395f51905f5255565b633ee5aeb560e01b5f5260045ffd5b307f00000000000000000000000000000000000000000000000000000000000000006001600160a01b0316148061120f575b15611177577f000000000000000000000000000000000000000000000000000000000000000090565b60405160208101907f8b73c3c69bb8fe3d512ecc4cf759cc79239f7b179b0ffacaa9a75d522b39400f82527f000000000000000000000000000000000000000000000000000000000000000060408201527f000000000000000000000000000000000000000000000000000000000000000060608201524660808201523060a082015260a0815261120960c082610dc2565b51902090565b507f0000000000000000000000000000000000000000000000000000000000000000461461114e565b60ff811461127e5760ff811690601f821161126f576040519161125c604084610dc2565b6020808452838101919036833783525290565b632cd44ac360e21b5f5260045ffd5b506040515f5f548060011c9160018216918215611354575b60208410831461134057838552849290811561132157506001146112c4575b6112c192500382610dc2565b90565b505f80805290917f290decd9548b62a8d60345a988386fc84ba6bc95484008f6362f93160ef3e5635b8183106113055750509060206112c1928201016112b5565b60209193508060019154838588010152019101909183926112ed565b602092506112c194915060ff191682840152151560051b8201016112b5565b634e487b7160e01b5f52602260045260245ffd5b92607f1692611296565b60ff81146113825760ff811690601f821161126f576040519161125c604084610dc2565b506040515f6001548060011c9160018216918215611424575b60208410831461134057838552849290811561132157506001146113c5576112c192500382610dc2565b5060015f90815290917fb10e2d527612073b26eecdfd717e6a320cf44b4afac2b0732d9fcbe2b7fa0cf65b8183106114085750509060206112c1928201016112b5565b60209193508060019154838588010152019101909183926113f0565b92607f169261139b565b815191906041830361145e576114579250602082015190606060408401519301515f1a906114dc565b9192909190565b50505f9160029190565b60048110156114c8578061147a575050565b600181036114915763f645eedf60e01b5f5260045ffd5b600281036114ac575063fce698f760e01b5f5260045260245ffd5b6003146114b65750565b6335e2f38360e21b5f5260045260245ffd5b634e487b7160e01b5f52602160045260245ffd5b91907f7fffffffffffffffffffffffffffffff5d576e7357a4501ddfe92f46681b20a0841161155e579160209360809260ff5f9560405194855216868401526040830152606082015282805260015afa15611553575f516001600160a01b0381161561154957905f905f90565b505f906001905f90565b6040513d5f823e3d90fd5b5050505f916003919056fef652222313e28459528d920b65115c16c04f3efc82aaedc97be59f3f377c0d3f9b779b17422d0df92223018b32b4d1fa46e071723d6817e2486d003becc55f00",
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

// WITHDRAWALEXECUTIONWINDOW is a free data retrieval call binding the contract method 0x289381ff.
//
// Solidity: function WITHDRAWAL_EXECUTION_WINDOW() view returns(uint256)
func (_Custody *CustodyCaller) WITHDRAWALEXECUTIONWINDOW(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Custody.contract.Call(opts, &out, "WITHDRAWAL_EXECUTION_WINDOW")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// WITHDRAWALEXECUTIONWINDOW is a free data retrieval call binding the contract method 0x289381ff.
//
// Solidity: function WITHDRAWAL_EXECUTION_WINDOW() view returns(uint256)
func (_Custody *CustodySession) WITHDRAWALEXECUTIONWINDOW() (*big.Int, error) {
	return _Custody.Contract.WITHDRAWALEXECUTIONWINDOW(&_Custody.CallOpts)
}

// WITHDRAWALEXECUTIONWINDOW is a free data retrieval call binding the contract method 0x289381ff.
//
// Solidity: function WITHDRAWAL_EXECUTION_WINDOW() view returns(uint256)
func (_Custody *CustodyCallerSession) WITHDRAWALEXECUTIONWINDOW() (*big.Int, error) {
	return _Custody.Contract.WITHDRAWALEXECUTIONWINDOW(&_Custody.CallOpts)
}

// Eip712Domain is a free data retrieval call binding the contract method 0x84b0196e.
//
// Solidity: function eip712Domain() view returns(bytes1 fields, string name, string version, uint256 chainId, address verifyingContract, bytes32 salt, uint256[] extensions)
func (_Custody *CustodyCaller) Eip712Domain(opts *bind.CallOpts) (struct {
	Fields            [1]byte
	Name              string
	Version           string
	ChainId           *big.Int
	VerifyingContract common.Address
	Salt              [32]byte
	Extensions        []*big.Int
}, error) {
	var out []interface{}
	err := _Custody.contract.Call(opts, &out, "eip712Domain")

	outstruct := new(struct {
		Fields            [1]byte
		Name              string
		Version           string
		ChainId           *big.Int
		VerifyingContract common.Address
		Salt              [32]byte
		Extensions        []*big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Fields = *abi.ConvertType(out[0], new([1]byte)).(*[1]byte)
	outstruct.Name = *abi.ConvertType(out[1], new(string)).(*string)
	outstruct.Version = *abi.ConvertType(out[2], new(string)).(*string)
	outstruct.ChainId = *abi.ConvertType(out[3], new(*big.Int)).(**big.Int)
	outstruct.VerifyingContract = *abi.ConvertType(out[4], new(common.Address)).(*common.Address)
	outstruct.Salt = *abi.ConvertType(out[5], new([32]byte)).(*[32]byte)
	outstruct.Extensions = *abi.ConvertType(out[6], new([]*big.Int)).(*[]*big.Int)

	return *outstruct, err

}

// Eip712Domain is a free data retrieval call binding the contract method 0x84b0196e.
//
// Solidity: function eip712Domain() view returns(bytes1 fields, string name, string version, uint256 chainId, address verifyingContract, bytes32 salt, uint256[] extensions)
func (_Custody *CustodySession) Eip712Domain() (struct {
	Fields            [1]byte
	Name              string
	Version           string
	ChainId           *big.Int
	VerifyingContract common.Address
	Salt              [32]byte
	Extensions        []*big.Int
}, error) {
	return _Custody.Contract.Eip712Domain(&_Custody.CallOpts)
}

// Eip712Domain is a free data retrieval call binding the contract method 0x84b0196e.
//
// Solidity: function eip712Domain() view returns(bytes1 fields, string name, string version, uint256 chainId, address verifyingContract, bytes32 salt, uint256[] extensions)
func (_Custody *CustodyCallerSession) Eip712Domain() (struct {
	Fields            [1]byte
	Name              string
	Version           string
	ChainId           *big.Int
	VerifyingContract common.Address
	Salt              [32]byte
	Extensions        []*big.Int
}, error) {
	return _Custody.Contract.Eip712Domain(&_Custody.CallOpts)
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

// RotationNonce is a free data retrieval call binding the contract method 0x94603071.
//
// Solidity: function rotationNonce() view returns(uint256)
func (_Custody *CustodyCaller) RotationNonce(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Custody.contract.Call(opts, &out, "rotationNonce")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// RotationNonce is a free data retrieval call binding the contract method 0x94603071.
//
// Solidity: function rotationNonce() view returns(uint256)
func (_Custody *CustodySession) RotationNonce() (*big.Int, error) {
	return _Custody.Contract.RotationNonce(&_Custody.CallOpts)
}

// RotationNonce is a free data retrieval call binding the contract method 0x94603071.
//
// Solidity: function rotationNonce() view returns(uint256)
func (_Custody *CustodyCallerSession) RotationNonce() (*big.Int, error) {
	return _Custody.Contract.RotationNonce(&_Custody.CallOpts)
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

// Execute is a paid mutator transaction binding the contract method 0x4f0b83a6.
//
// Solidity: function execute(address to, address asset, uint256 amount, bytes32 withdrawalId, uint256 finalizedAt, uint256 authorizationRotationNonce, bytes[] signatures) returns()
func (_Custody *CustodyTransactor) Execute(opts *bind.TransactOpts, to common.Address, asset common.Address, amount *big.Int, withdrawalId [32]byte, finalizedAt *big.Int, authorizationRotationNonce *big.Int, signatures [][]byte) (*types.Transaction, error) {
	return _Custody.contract.Transact(opts, "execute", to, asset, amount, withdrawalId, finalizedAt, authorizationRotationNonce, signatures)
}

// Execute is a paid mutator transaction binding the contract method 0x4f0b83a6.
//
// Solidity: function execute(address to, address asset, uint256 amount, bytes32 withdrawalId, uint256 finalizedAt, uint256 authorizationRotationNonce, bytes[] signatures) returns()
func (_Custody *CustodySession) Execute(to common.Address, asset common.Address, amount *big.Int, withdrawalId [32]byte, finalizedAt *big.Int, authorizationRotationNonce *big.Int, signatures [][]byte) (*types.Transaction, error) {
	return _Custody.Contract.Execute(&_Custody.TransactOpts, to, asset, amount, withdrawalId, finalizedAt, authorizationRotationNonce, signatures)
}

// Execute is a paid mutator transaction binding the contract method 0x4f0b83a6.
//
// Solidity: function execute(address to, address asset, uint256 amount, bytes32 withdrawalId, uint256 finalizedAt, uint256 authorizationRotationNonce, bytes[] signatures) returns()
func (_Custody *CustodyTransactorSession) Execute(to common.Address, asset common.Address, amount *big.Int, withdrawalId [32]byte, finalizedAt *big.Int, authorizationRotationNonce *big.Int, signatures [][]byte) (*types.Transaction, error) {
	return _Custody.Contract.Execute(&_Custody.TransactOpts, to, asset, amount, withdrawalId, finalizedAt, authorizationRotationNonce, signatures)
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

// CustodyEIP712DomainChangedIterator is returned from FilterEIP712DomainChanged and is used to iterate over the raw logs and unpacked data for EIP712DomainChanged events raised by the Custody contract.
type CustodyEIP712DomainChangedIterator struct {
	Event *CustodyEIP712DomainChanged // Event containing the contract specifics and raw log

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
func (it *CustodyEIP712DomainChangedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(CustodyEIP712DomainChanged)
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
		it.Event = new(CustodyEIP712DomainChanged)
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
func (it *CustodyEIP712DomainChangedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *CustodyEIP712DomainChangedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// CustodyEIP712DomainChanged represents a EIP712DomainChanged event raised by the Custody contract.
type CustodyEIP712DomainChanged struct {
	Raw types.Log // Blockchain specific contextual infos
}

// FilterEIP712DomainChanged is a free log retrieval operation binding the contract event 0x0a6387c9ea3628b88a633bb4f3b151770f70085117a15f9bf3787cda53f13d31.
//
// Solidity: event EIP712DomainChanged()
func (_Custody *CustodyFilterer) FilterEIP712DomainChanged(opts *bind.FilterOpts) (*CustodyEIP712DomainChangedIterator, error) {

	logs, sub, err := _Custody.contract.FilterLogs(opts, "EIP712DomainChanged")
	if err != nil {
		return nil, err
	}
	return &CustodyEIP712DomainChangedIterator{contract: _Custody.contract, event: "EIP712DomainChanged", logs: logs, sub: sub}, nil
}

// WatchEIP712DomainChanged is a free log subscription operation binding the contract event 0x0a6387c9ea3628b88a633bb4f3b151770f70085117a15f9bf3787cda53f13d31.
//
// Solidity: event EIP712DomainChanged()
func (_Custody *CustodyFilterer) WatchEIP712DomainChanged(opts *bind.WatchOpts, sink chan<- *CustodyEIP712DomainChanged) (event.Subscription, error) {

	logs, sub, err := _Custody.contract.WatchLogs(opts, "EIP712DomainChanged")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(CustodyEIP712DomainChanged)
				if err := _Custody.contract.UnpackLog(event, "EIP712DomainChanged", log); err != nil {
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

// ParseEIP712DomainChanged is a log parse operation binding the contract event 0x0a6387c9ea3628b88a633bb4f3b151770f70085117a15f9bf3787cda53f13d31.
//
// Solidity: event EIP712DomainChanged()
func (_Custody *CustodyFilterer) ParseEIP712DomainChanged(log types.Log) (*CustodyEIP712DomainChanged, error) {
	event := new(CustodyEIP712DomainChanged)
	if err := _Custody.contract.UnpackLog(event, "EIP712DomainChanged", log); err != nil {
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

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

// YellowTokenMetaData contains all meta data concerning the YellowToken contract.
var YellowTokenMetaData = &bind.MetaData{
	ABI: "[{\"type\":\"constructor\",\"inputs\":[{\"name\":\"treasury\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"DOMAIN_SEPARATOR\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"SUPPLY_CAP\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"allowance\",\"inputs\":[{\"name\":\"owner\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"spender\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"approve\",\"inputs\":[{\"name\":\"spender\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"value\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"balanceOf\",\"inputs\":[{\"name\":\"account\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"decimals\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint8\",\"internalType\":\"uint8\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"eip712Domain\",\"inputs\":[],\"outputs\":[{\"name\":\"fields\",\"type\":\"bytes1\",\"internalType\":\"bytes1\"},{\"name\":\"name\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"version\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"chainId\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"verifyingContract\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"salt\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"extensions\",\"type\":\"uint256[]\",\"internalType\":\"uint256[]\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"name\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"string\",\"internalType\":\"string\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"nonces\",\"inputs\":[{\"name\":\"owner\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"permit\",\"inputs\":[{\"name\":\"owner\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"spender\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"value\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"deadline\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"v\",\"type\":\"uint8\",\"internalType\":\"uint8\"},{\"name\":\"r\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"s\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"symbol\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"string\",\"internalType\":\"string\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"totalSupply\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"transfer\",\"inputs\":[{\"name\":\"to\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"value\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"transferFrom\",\"inputs\":[{\"name\":\"from\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"to\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"value\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"event\",\"name\":\"Approval\",\"inputs\":[{\"name\":\"owner\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"spender\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"value\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"EIP712DomainChanged\",\"inputs\":[],\"anonymous\":false},{\"type\":\"event\",\"name\":\"Transfer\",\"inputs\":[{\"name\":\"from\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"to\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"value\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"error\",\"name\":\"ECDSAInvalidSignature\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"ECDSAInvalidSignatureLength\",\"inputs\":[{\"name\":\"length\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]},{\"type\":\"error\",\"name\":\"ECDSAInvalidSignatureS\",\"inputs\":[{\"name\":\"s\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}]},{\"type\":\"error\",\"name\":\"ERC20InsufficientAllowance\",\"inputs\":[{\"name\":\"spender\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"allowance\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"needed\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]},{\"type\":\"error\",\"name\":\"ERC20InsufficientBalance\",\"inputs\":[{\"name\":\"sender\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"balance\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"needed\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]},{\"type\":\"error\",\"name\":\"ERC20InvalidApprover\",\"inputs\":[{\"name\":\"approver\",\"type\":\"address\",\"internalType\":\"address\"}]},{\"type\":\"error\",\"name\":\"ERC20InvalidReceiver\",\"inputs\":[{\"name\":\"receiver\",\"type\":\"address\",\"internalType\":\"address\"}]},{\"type\":\"error\",\"name\":\"ERC20InvalidSender\",\"inputs\":[{\"name\":\"sender\",\"type\":\"address\",\"internalType\":\"address\"}]},{\"type\":\"error\",\"name\":\"ERC20InvalidSpender\",\"inputs\":[{\"name\":\"spender\",\"type\":\"address\",\"internalType\":\"address\"}]},{\"type\":\"error\",\"name\":\"ERC2612ExpiredSignature\",\"inputs\":[{\"name\":\"deadline\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]},{\"type\":\"error\",\"name\":\"ERC2612InvalidSigner\",\"inputs\":[{\"name\":\"signer\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"owner\",\"type\":\"address\",\"internalType\":\"address\"}]},{\"type\":\"error\",\"name\":\"InvalidAccountNonce\",\"inputs\":[{\"name\":\"account\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"currentNonce\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]},{\"type\":\"error\",\"name\":\"InvalidAddress\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"InvalidShortString\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"StringTooLong\",\"inputs\":[{\"name\":\"str\",\"type\":\"string\",\"internalType\":\"string\"}]}]",
	Bin: "0x61016080604052346105045760208161140980380380916100208285610508565b83398101031261050457516001600160a01b038116908190036105045760405161004b604082610508565b60068152602081016559656c6c6f7760d01b81526040519061006e604083610508565b600682526559656c6c6f7760d01b602083015260405192610090604085610508565b600684526559454c4c4f5760d01b6020850152604051936100b2604086610508565b60018552603160f81b60208601908152845190946001600160401b0382116103ff5760035490600182811c921680156104fa575b60208310146103e15781601f849311610484575b50602090601f831160011461041e575f92610413575b50508160011b915f199060031b1c1916176003555b8051906001600160401b0382116103ff5760045490600182811c921680156103f5575b60208310146103e15781601f84931161036b575b50602090601f8311600114610305575f926102fa575b50508160011b915f199060031b1c1916176004555b6101908161052b565b6101205261019d846106be565b61014052519020918260e05251902080610100524660a0526040519060208201927f8b73c3c69bb8fe3d512ecc4cf759cc79239f7b179b0ffacaa9a75d522b39400f8452604083015260608201524660808201523060a082015260a0815261020660c082610508565b5190206080523060c05280156102eb576002546b204fce5e3e2502611000000081018091116102d757600255805f525f60205260405f206b204fce5e3e2502611000000081540190555f7fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef60206040516b204fce5e3e250261100000008152a3604051610c069081610803823960805181610925015260a051816109e2015260c051816108ef015260e051816109740152610100518161099a0152610120518161037a015261014051816103a30152f35b634e487b7160e01b5f52601160045260245ffd5b63e6c4247b60e01b5f5260045ffd5b015190505f80610172565b60045f9081528281209350601f198516905b818110610353575090846001959493921061033b575b505050811b01600455610187565b01515f1960f88460031b161c191690555f808061032d565b92936020600181928786015181550195019301610317565b8281111561015c5760045f52909150601f830160051c7f8a35acfbc15ff81a39ae7d344fd709f28e8600b4aa8c65c6b64bfe7fe36bd19b602085106103d9575b849392601f0160051c82900391015f5b8281106103c957505061015c565b5f818301558594506001016103bb565b5f91506103ab565b634e487b7160e01b5f52602260045260245ffd5b91607f1691610148565b634e487b7160e01b5f52604160045260245ffd5b015190505f80610110565b60035f9081528281209350601f198516905b81811061046c5750908460019594939210610454575b505050811b01600355610125565b01515f1960f88460031b161c191690555f8080610446565b92936020600181928786015181550195019301610430565b828111156100fa5760035f52909150601f830160051c7fc2575a0e9e593c00f959f8c92f12db2869c3395a3b0502d05e2516446f71f85b602085106104f2575b849392601f0160051c82900391015f5b8281106104e25750506100fa565b5f818301558594506001016104d4565b5f91506104c4565b91607f16916100e6565b5f80fd5b601f909101601f19168101906001600160401b038211908210176103ff57604052565b908151602081105f146105a5575090601f815111610565576020815191015160208210610556571790565b5f198260200360031b1b161790565b604460209160405192839163305a27a960e01b83528160048401528051918291826024860152018484015e5f828201840152601f01601f19168101030190fd5b6001600160401b0381116103ff57600554600181811c911680156106b4575b60208210146103e157601f8111610675575b50602092601f821160011461061457928192935f92610609575b50508160011b915f199060031b1c19161760055560ff90565b015190505f806105f0565b601f1982169360055f52805f20915f5b86811061065d5750836001959610610645575b505050811b0160055560ff90565b01515f1960f88460031b161c191690555f8080610637565b91926020600181928685015181550194019201610624565b818111156105d65760055f5260205f20601f80840160051c809201920160051c03905f5b8281106106a75750506105d6565b5f82820155600101610699565b90607f16906105c4565b908151602081105f146106e9575090601f815111610565576020815191015160208210610556571790565b6001600160401b0381116103ff57600654600181811c911680156107f8575b60208210146103e157601f81116107b9575b50602092601f821160011461075857928192935f9261074d575b50508160011b915f199060031b1c19161760065560ff90565b015190505f80610734565b601f1982169360065f52805f20915f5b8681106107a15750836001959610610789575b505050811b0160065560ff90565b01515f1960f88460031b161c191690555f808061077b565b91926020600181928685015181550194019201610768565b8181111561071a5760065f5260205f20601f80840160051c809201920160051c03905f5b8281106107eb57505061071a565b5f828201556001016107dd565b90607f169061070856fe6080806040526004361015610012575f80fd5b5f3560e01c90816306fdde031461064e57508063095ea7b3146106285780630cfccc831461060257806318160ddd146105e557806323b872dd14610506578063313ce567146104eb5780633644e515146104c957806370a08231146104925780637ecebe001461045a57806384b0196e1461036257806395d89b4114610280578063a9059cbb1461024f578063d505accf1461010a5763dd62ed3e146100b6575f80fd5b34610106576040366003190112610106576100cf610714565b6100d761072a565b6001600160a01b039182165f908152600160209081526040808320949093168252928352819020549051908152f35b5f80fd5b346101065760e036600319011261010657610123610714565b61012b61072a565b604435906064359260843560ff811681036101065784421161023c576101ff6102089160018060a01b03841696875f52600760205260405f20908154916001830190556040519060208201927f6e71edae12b1b97f4d1f60370fef10105fa2faae0126114a169c64845d6126c984528a604084015260018060a01b038916606084015289608084015260a083015260c082015260c081526101cd60e0826107f9565b5190206101d86108ec565b906040519161190160f01b83526002830152602282015260c43591604260a4359220610b05565b90929192610b92565b6001600160a01b031684810361022557506102239350610a08565b005b84906325c0072360e11b5f5260045260245260445ffd5b8463313c898160e11b5f5260045260245ffd5b346101065760403660031901126101065761027561026b610714565b602435903361082f565b602060405160018152f35b34610106575f366003190112610106576040515f6004546102a081610740565b808452906001811690811561033e57506001146102e0575b6102dc836102c8818503826107f9565b6040519182916020835260208301906106f0565b0390f35b60045f9081527f8a35acfbc15ff81a39ae7d344fd709f28e8600b4aa8c65c6b64bfe7fe36bd19b939250905b808210610324575090915081016020016102c86102b8565b91926001816020925483858801015201910190929161030c565b60ff191660208086019190915291151560051b840190910191506102c890506102b8565b34610106575f366003190112610106576103fe61039e7f0000000000000000000000000000000000000000000000000000000000000000610a6b565b6103c77f0000000000000000000000000000000000000000000000000000000000000000610ace565b602061040c604051926103da83856107f9565b5f84525f368137604051958695600f60f81b875260e08588015260e08701906106f0565b9085820360408701526106f0565b4660608501523060808501525f60a085015283810360c08501528180845192838152019301915f5b82811061044357505050500390f35b835185528695509381019392810192600101610434565b34610106576020366003190112610106576001600160a01b0361047b610714565b165f526007602052602060405f2054604051908152f35b34610106576020366003190112610106576001600160a01b036104b3610714565b165f525f602052602060405f2054604051908152f35b34610106575f3660031901126101065760206104e36108ec565b604051908152f35b34610106575f36600319011261010657602060405160128152f35b346101065760603660031901126101065761051f610714565b61052761072a565b6001600160a01b0382165f818152600160209081526040808320338452909152902054909260443592915f198110610565575b50610275935061082f565b8381106105ca5784156105b75733156105a457610275945f52600160205260405f2060018060a01b0333165f526020528360405f20910390558461055a565b634a1406b160e11b5f525f60045260245ffd5b63e602df0560e01b5f525f60045260245ffd5b8390637dc7a0d960e11b5f523360045260245260445260645ffd5b34610106575f366003190112610106576020600254604051908152f35b34610106575f3660031901126101065760206040516b204fce5e3e250261100000008152f35b3461010657604036600319011261010657610275610644610714565b6024359033610a08565b34610106575f366003190112610106575f60035461066b81610740565b808452906001811690811561033e5750600114610692576102dc836102c8818503826107f9565b60035f9081527fc2575a0e9e593c00f959f8c92f12db2869c3395a3b0502d05e2516446f71f85b939250905b8082106106d6575090915081016020016102c86102b8565b9192600181602092548385880101520191019092916106be565b805180835260209291819084018484015e5f828201840152601f01601f1916010190565b600435906001600160a01b038216820361010657565b602435906001600160a01b038216820361010657565b90600182811c9216801561076e575b602083101461075a57565b634e487b7160e01b5f52602260045260245ffd5b91607f169161074f565b5f929181549161078783610740565b80835292600181169081156107dc57506001146107a357505050565b5f9081526020812093945091925b8383106107c2575060209250010190565b6001816020929493945483858701015201910191906107b1565b915050602093945060ff929192191683830152151560051b010190565b90601f8019910116810190811067ffffffffffffffff82111761081b57604052565b634e487b7160e01b5f52604160045260245ffd5b6001600160a01b03169081156108d9576001600160a01b03169182156108c657815f525f60205260405f20548181106108ad57817fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef92602092855f525f84520360405f2055845f525f825260405f20818154019055604051908152a3565b8263391434e360e21b5f5260045260245260445260645ffd5b63ec442f0560e01b5f525f60045260245ffd5b634b637e8f60e11b5f525f60045260245ffd5b307f00000000000000000000000000000000000000000000000000000000000000006001600160a01b031614806109df575b15610947577f000000000000000000000000000000000000000000000000000000000000000090565b60405160208101907f8b73c3c69bb8fe3d512ecc4cf759cc79239f7b179b0ffacaa9a75d522b39400f82527f000000000000000000000000000000000000000000000000000000000000000060408201527f000000000000000000000000000000000000000000000000000000000000000060608201524660808201523060a082015260a081526109d960c0826107f9565b51902090565b507f0000000000000000000000000000000000000000000000000000000000000000461461091e565b6001600160a01b03169081156105b7576001600160a01b03169182156105a45760207f8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b92591835f526001825260405f20855f5282528060405f2055604051908152a3565b60ff8114610ab15760ff811690601f8211610aa25760405191610a8f6040846107f9565b6020808452838101919036833783525290565b632cd44ac360e21b5f5260045ffd5b50604051610acb81610ac4816005610778565b03826107f9565b90565b60ff8114610af25760ff811690601f8211610aa25760405191610a8f6040846107f9565b50604051610acb81610ac4816006610778565b91907f7fffffffffffffffffffffffffffffff5d576e7357a4501ddfe92f46681b20a08411610b87579160209360809260ff5f9560405194855216868401526040830152606082015282805260015afa15610b7c575f516001600160a01b03811615610b7257905f905f90565b505f906001905f90565b6040513d5f823e3d90fd5b5050505f9160039190565b6004811015610bf25780610ba4575050565b60018103610bbb5763f645eedf60e01b5f5260045ffd5b60028103610bd6575063fce698f760e01b5f5260045260245ffd5b600314610be05750565b6335e2f38360e21b5f5260045260245ffd5b634e487b7160e01b5f52602160045260245ffd",
}

// YellowTokenABI is the input ABI used to generate the binding from.
// Deprecated: Use YellowTokenMetaData.ABI instead.
var YellowTokenABI = YellowTokenMetaData.ABI

// YellowTokenBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use YellowTokenMetaData.Bin instead.
var YellowTokenBin = YellowTokenMetaData.Bin

// DeployYellowToken deploys a new Ethereum contract, binding an instance of YellowToken to it.
func DeployYellowToken(auth *bind.TransactOpts, backend bind.ContractBackend, treasury common.Address) (common.Address, *types.Transaction, *YellowToken, error) {
	parsed, err := YellowTokenMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(YellowTokenBin), backend, treasury)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &YellowToken{YellowTokenCaller: YellowTokenCaller{contract: contract}, YellowTokenTransactor: YellowTokenTransactor{contract: contract}, YellowTokenFilterer: YellowTokenFilterer{contract: contract}}, nil
}

// YellowToken is an auto generated Go binding around an Ethereum contract.
type YellowToken struct {
	YellowTokenCaller     // Read-only binding to the contract
	YellowTokenTransactor // Write-only binding to the contract
	YellowTokenFilterer   // Log filterer for contract events
}

// YellowTokenCaller is an auto generated read-only Go binding around an Ethereum contract.
type YellowTokenCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// YellowTokenTransactor is an auto generated write-only Go binding around an Ethereum contract.
type YellowTokenTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// YellowTokenFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type YellowTokenFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// YellowTokenSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type YellowTokenSession struct {
	Contract     *YellowToken      // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// YellowTokenCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type YellowTokenCallerSession struct {
	Contract *YellowTokenCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts      // Call options to use throughout this session
}

// YellowTokenTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type YellowTokenTransactorSession struct {
	Contract     *YellowTokenTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts      // Transaction auth options to use throughout this session
}

// YellowTokenRaw is an auto generated low-level Go binding around an Ethereum contract.
type YellowTokenRaw struct {
	Contract *YellowToken // Generic contract binding to access the raw methods on
}

// YellowTokenCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type YellowTokenCallerRaw struct {
	Contract *YellowTokenCaller // Generic read-only contract binding to access the raw methods on
}

// YellowTokenTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type YellowTokenTransactorRaw struct {
	Contract *YellowTokenTransactor // Generic write-only contract binding to access the raw methods on
}

// NewYellowToken creates a new instance of YellowToken, bound to a specific deployed contract.
func NewYellowToken(address common.Address, backend bind.ContractBackend) (*YellowToken, error) {
	contract, err := bindYellowToken(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &YellowToken{YellowTokenCaller: YellowTokenCaller{contract: contract}, YellowTokenTransactor: YellowTokenTransactor{contract: contract}, YellowTokenFilterer: YellowTokenFilterer{contract: contract}}, nil
}

// NewYellowTokenCaller creates a new read-only instance of YellowToken, bound to a specific deployed contract.
func NewYellowTokenCaller(address common.Address, caller bind.ContractCaller) (*YellowTokenCaller, error) {
	contract, err := bindYellowToken(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &YellowTokenCaller{contract: contract}, nil
}

// NewYellowTokenTransactor creates a new write-only instance of YellowToken, bound to a specific deployed contract.
func NewYellowTokenTransactor(address common.Address, transactor bind.ContractTransactor) (*YellowTokenTransactor, error) {
	contract, err := bindYellowToken(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &YellowTokenTransactor{contract: contract}, nil
}

// NewYellowTokenFilterer creates a new log filterer instance of YellowToken, bound to a specific deployed contract.
func NewYellowTokenFilterer(address common.Address, filterer bind.ContractFilterer) (*YellowTokenFilterer, error) {
	contract, err := bindYellowToken(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &YellowTokenFilterer{contract: contract}, nil
}

// bindYellowToken binds a generic wrapper to an already deployed contract.
func bindYellowToken(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := YellowTokenMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_YellowToken *YellowTokenRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _YellowToken.Contract.YellowTokenCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_YellowToken *YellowTokenRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _YellowToken.Contract.YellowTokenTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_YellowToken *YellowTokenRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _YellowToken.Contract.YellowTokenTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_YellowToken *YellowTokenCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _YellowToken.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_YellowToken *YellowTokenTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _YellowToken.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_YellowToken *YellowTokenTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _YellowToken.Contract.contract.Transact(opts, method, params...)
}

// DOMAINSEPARATOR is a free data retrieval call binding the contract method 0x3644e515.
//
// Solidity: function DOMAIN_SEPARATOR() view returns(bytes32)
func (_YellowToken *YellowTokenCaller) DOMAINSEPARATOR(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _YellowToken.contract.Call(opts, &out, "DOMAIN_SEPARATOR")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// DOMAINSEPARATOR is a free data retrieval call binding the contract method 0x3644e515.
//
// Solidity: function DOMAIN_SEPARATOR() view returns(bytes32)
func (_YellowToken *YellowTokenSession) DOMAINSEPARATOR() ([32]byte, error) {
	return _YellowToken.Contract.DOMAINSEPARATOR(&_YellowToken.CallOpts)
}

// DOMAINSEPARATOR is a free data retrieval call binding the contract method 0x3644e515.
//
// Solidity: function DOMAIN_SEPARATOR() view returns(bytes32)
func (_YellowToken *YellowTokenCallerSession) DOMAINSEPARATOR() ([32]byte, error) {
	return _YellowToken.Contract.DOMAINSEPARATOR(&_YellowToken.CallOpts)
}

// SUPPLYCAP is a free data retrieval call binding the contract method 0x0cfccc83.
//
// Solidity: function SUPPLY_CAP() view returns(uint256)
func (_YellowToken *YellowTokenCaller) SUPPLYCAP(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _YellowToken.contract.Call(opts, &out, "SUPPLY_CAP")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// SUPPLYCAP is a free data retrieval call binding the contract method 0x0cfccc83.
//
// Solidity: function SUPPLY_CAP() view returns(uint256)
func (_YellowToken *YellowTokenSession) SUPPLYCAP() (*big.Int, error) {
	return _YellowToken.Contract.SUPPLYCAP(&_YellowToken.CallOpts)
}

// SUPPLYCAP is a free data retrieval call binding the contract method 0x0cfccc83.
//
// Solidity: function SUPPLY_CAP() view returns(uint256)
func (_YellowToken *YellowTokenCallerSession) SUPPLYCAP() (*big.Int, error) {
	return _YellowToken.Contract.SUPPLYCAP(&_YellowToken.CallOpts)
}

// Allowance is a free data retrieval call binding the contract method 0xdd62ed3e.
//
// Solidity: function allowance(address owner, address spender) view returns(uint256)
func (_YellowToken *YellowTokenCaller) Allowance(opts *bind.CallOpts, owner common.Address, spender common.Address) (*big.Int, error) {
	var out []interface{}
	err := _YellowToken.contract.Call(opts, &out, "allowance", owner, spender)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Allowance is a free data retrieval call binding the contract method 0xdd62ed3e.
//
// Solidity: function allowance(address owner, address spender) view returns(uint256)
func (_YellowToken *YellowTokenSession) Allowance(owner common.Address, spender common.Address) (*big.Int, error) {
	return _YellowToken.Contract.Allowance(&_YellowToken.CallOpts, owner, spender)
}

// Allowance is a free data retrieval call binding the contract method 0xdd62ed3e.
//
// Solidity: function allowance(address owner, address spender) view returns(uint256)
func (_YellowToken *YellowTokenCallerSession) Allowance(owner common.Address, spender common.Address) (*big.Int, error) {
	return _YellowToken.Contract.Allowance(&_YellowToken.CallOpts, owner, spender)
}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address account) view returns(uint256)
func (_YellowToken *YellowTokenCaller) BalanceOf(opts *bind.CallOpts, account common.Address) (*big.Int, error) {
	var out []interface{}
	err := _YellowToken.contract.Call(opts, &out, "balanceOf", account)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address account) view returns(uint256)
func (_YellowToken *YellowTokenSession) BalanceOf(account common.Address) (*big.Int, error) {
	return _YellowToken.Contract.BalanceOf(&_YellowToken.CallOpts, account)
}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address account) view returns(uint256)
func (_YellowToken *YellowTokenCallerSession) BalanceOf(account common.Address) (*big.Int, error) {
	return _YellowToken.Contract.BalanceOf(&_YellowToken.CallOpts, account)
}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() view returns(uint8)
func (_YellowToken *YellowTokenCaller) Decimals(opts *bind.CallOpts) (uint8, error) {
	var out []interface{}
	err := _YellowToken.contract.Call(opts, &out, "decimals")

	if err != nil {
		return *new(uint8), err
	}

	out0 := *abi.ConvertType(out[0], new(uint8)).(*uint8)

	return out0, err

}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() view returns(uint8)
func (_YellowToken *YellowTokenSession) Decimals() (uint8, error) {
	return _YellowToken.Contract.Decimals(&_YellowToken.CallOpts)
}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() view returns(uint8)
func (_YellowToken *YellowTokenCallerSession) Decimals() (uint8, error) {
	return _YellowToken.Contract.Decimals(&_YellowToken.CallOpts)
}

// Eip712Domain is a free data retrieval call binding the contract method 0x84b0196e.
//
// Solidity: function eip712Domain() view returns(bytes1 fields, string name, string version, uint256 chainId, address verifyingContract, bytes32 salt, uint256[] extensions)
func (_YellowToken *YellowTokenCaller) Eip712Domain(opts *bind.CallOpts) (struct {
	Fields            [1]byte
	Name              string
	Version           string
	ChainId           *big.Int
	VerifyingContract common.Address
	Salt              [32]byte
	Extensions        []*big.Int
}, error) {
	var out []interface{}
	err := _YellowToken.contract.Call(opts, &out, "eip712Domain")

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
func (_YellowToken *YellowTokenSession) Eip712Domain() (struct {
	Fields            [1]byte
	Name              string
	Version           string
	ChainId           *big.Int
	VerifyingContract common.Address
	Salt              [32]byte
	Extensions        []*big.Int
}, error) {
	return _YellowToken.Contract.Eip712Domain(&_YellowToken.CallOpts)
}

// Eip712Domain is a free data retrieval call binding the contract method 0x84b0196e.
//
// Solidity: function eip712Domain() view returns(bytes1 fields, string name, string version, uint256 chainId, address verifyingContract, bytes32 salt, uint256[] extensions)
func (_YellowToken *YellowTokenCallerSession) Eip712Domain() (struct {
	Fields            [1]byte
	Name              string
	Version           string
	ChainId           *big.Int
	VerifyingContract common.Address
	Salt              [32]byte
	Extensions        []*big.Int
}, error) {
	return _YellowToken.Contract.Eip712Domain(&_YellowToken.CallOpts)
}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_YellowToken *YellowTokenCaller) Name(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _YellowToken.contract.Call(opts, &out, "name")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_YellowToken *YellowTokenSession) Name() (string, error) {
	return _YellowToken.Contract.Name(&_YellowToken.CallOpts)
}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_YellowToken *YellowTokenCallerSession) Name() (string, error) {
	return _YellowToken.Contract.Name(&_YellowToken.CallOpts)
}

// Nonces is a free data retrieval call binding the contract method 0x7ecebe00.
//
// Solidity: function nonces(address owner) view returns(uint256)
func (_YellowToken *YellowTokenCaller) Nonces(opts *bind.CallOpts, owner common.Address) (*big.Int, error) {
	var out []interface{}
	err := _YellowToken.contract.Call(opts, &out, "nonces", owner)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Nonces is a free data retrieval call binding the contract method 0x7ecebe00.
//
// Solidity: function nonces(address owner) view returns(uint256)
func (_YellowToken *YellowTokenSession) Nonces(owner common.Address) (*big.Int, error) {
	return _YellowToken.Contract.Nonces(&_YellowToken.CallOpts, owner)
}

// Nonces is a free data retrieval call binding the contract method 0x7ecebe00.
//
// Solidity: function nonces(address owner) view returns(uint256)
func (_YellowToken *YellowTokenCallerSession) Nonces(owner common.Address) (*big.Int, error) {
	return _YellowToken.Contract.Nonces(&_YellowToken.CallOpts, owner)
}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() view returns(string)
func (_YellowToken *YellowTokenCaller) Symbol(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _YellowToken.contract.Call(opts, &out, "symbol")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() view returns(string)
func (_YellowToken *YellowTokenSession) Symbol() (string, error) {
	return _YellowToken.Contract.Symbol(&_YellowToken.CallOpts)
}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() view returns(string)
func (_YellowToken *YellowTokenCallerSession) Symbol() (string, error) {
	return _YellowToken.Contract.Symbol(&_YellowToken.CallOpts)
}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() view returns(uint256)
func (_YellowToken *YellowTokenCaller) TotalSupply(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _YellowToken.contract.Call(opts, &out, "totalSupply")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() view returns(uint256)
func (_YellowToken *YellowTokenSession) TotalSupply() (*big.Int, error) {
	return _YellowToken.Contract.TotalSupply(&_YellowToken.CallOpts)
}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() view returns(uint256)
func (_YellowToken *YellowTokenCallerSession) TotalSupply() (*big.Int, error) {
	return _YellowToken.Contract.TotalSupply(&_YellowToken.CallOpts)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address spender, uint256 value) returns(bool)
func (_YellowToken *YellowTokenTransactor) Approve(opts *bind.TransactOpts, spender common.Address, value *big.Int) (*types.Transaction, error) {
	return _YellowToken.contract.Transact(opts, "approve", spender, value)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address spender, uint256 value) returns(bool)
func (_YellowToken *YellowTokenSession) Approve(spender common.Address, value *big.Int) (*types.Transaction, error) {
	return _YellowToken.Contract.Approve(&_YellowToken.TransactOpts, spender, value)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address spender, uint256 value) returns(bool)
func (_YellowToken *YellowTokenTransactorSession) Approve(spender common.Address, value *big.Int) (*types.Transaction, error) {
	return _YellowToken.Contract.Approve(&_YellowToken.TransactOpts, spender, value)
}

// Permit is a paid mutator transaction binding the contract method 0xd505accf.
//
// Solidity: function permit(address owner, address spender, uint256 value, uint256 deadline, uint8 v, bytes32 r, bytes32 s) returns()
func (_YellowToken *YellowTokenTransactor) Permit(opts *bind.TransactOpts, owner common.Address, spender common.Address, value *big.Int, deadline *big.Int, v uint8, r [32]byte, s [32]byte) (*types.Transaction, error) {
	return _YellowToken.contract.Transact(opts, "permit", owner, spender, value, deadline, v, r, s)
}

// Permit is a paid mutator transaction binding the contract method 0xd505accf.
//
// Solidity: function permit(address owner, address spender, uint256 value, uint256 deadline, uint8 v, bytes32 r, bytes32 s) returns()
func (_YellowToken *YellowTokenSession) Permit(owner common.Address, spender common.Address, value *big.Int, deadline *big.Int, v uint8, r [32]byte, s [32]byte) (*types.Transaction, error) {
	return _YellowToken.Contract.Permit(&_YellowToken.TransactOpts, owner, spender, value, deadline, v, r, s)
}

// Permit is a paid mutator transaction binding the contract method 0xd505accf.
//
// Solidity: function permit(address owner, address spender, uint256 value, uint256 deadline, uint8 v, bytes32 r, bytes32 s) returns()
func (_YellowToken *YellowTokenTransactorSession) Permit(owner common.Address, spender common.Address, value *big.Int, deadline *big.Int, v uint8, r [32]byte, s [32]byte) (*types.Transaction, error) {
	return _YellowToken.Contract.Permit(&_YellowToken.TransactOpts, owner, spender, value, deadline, v, r, s)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(address to, uint256 value) returns(bool)
func (_YellowToken *YellowTokenTransactor) Transfer(opts *bind.TransactOpts, to common.Address, value *big.Int) (*types.Transaction, error) {
	return _YellowToken.contract.Transact(opts, "transfer", to, value)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(address to, uint256 value) returns(bool)
func (_YellowToken *YellowTokenSession) Transfer(to common.Address, value *big.Int) (*types.Transaction, error) {
	return _YellowToken.Contract.Transfer(&_YellowToken.TransactOpts, to, value)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(address to, uint256 value) returns(bool)
func (_YellowToken *YellowTokenTransactorSession) Transfer(to common.Address, value *big.Int) (*types.Transaction, error) {
	return _YellowToken.Contract.Transfer(&_YellowToken.TransactOpts, to, value)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address from, address to, uint256 value) returns(bool)
func (_YellowToken *YellowTokenTransactor) TransferFrom(opts *bind.TransactOpts, from common.Address, to common.Address, value *big.Int) (*types.Transaction, error) {
	return _YellowToken.contract.Transact(opts, "transferFrom", from, to, value)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address from, address to, uint256 value) returns(bool)
func (_YellowToken *YellowTokenSession) TransferFrom(from common.Address, to common.Address, value *big.Int) (*types.Transaction, error) {
	return _YellowToken.Contract.TransferFrom(&_YellowToken.TransactOpts, from, to, value)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address from, address to, uint256 value) returns(bool)
func (_YellowToken *YellowTokenTransactorSession) TransferFrom(from common.Address, to common.Address, value *big.Int) (*types.Transaction, error) {
	return _YellowToken.Contract.TransferFrom(&_YellowToken.TransactOpts, from, to, value)
}

// YellowTokenApprovalIterator is returned from FilterApproval and is used to iterate over the raw logs and unpacked data for Approval events raised by the YellowToken contract.
type YellowTokenApprovalIterator struct {
	Event *YellowTokenApproval // Event containing the contract specifics and raw log

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
func (it *YellowTokenApprovalIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(YellowTokenApproval)
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
		it.Event = new(YellowTokenApproval)
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
func (it *YellowTokenApprovalIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *YellowTokenApprovalIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// YellowTokenApproval represents a Approval event raised by the YellowToken contract.
type YellowTokenApproval struct {
	Owner   common.Address
	Spender common.Address
	Value   *big.Int
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterApproval is a free log retrieval operation binding the contract event 0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925.
//
// Solidity: event Approval(address indexed owner, address indexed spender, uint256 value)
func (_YellowToken *YellowTokenFilterer) FilterApproval(opts *bind.FilterOpts, owner []common.Address, spender []common.Address) (*YellowTokenApprovalIterator, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var spenderRule []interface{}
	for _, spenderItem := range spender {
		spenderRule = append(spenderRule, spenderItem)
	}

	logs, sub, err := _YellowToken.contract.FilterLogs(opts, "Approval", ownerRule, spenderRule)
	if err != nil {
		return nil, err
	}
	return &YellowTokenApprovalIterator{contract: _YellowToken.contract, event: "Approval", logs: logs, sub: sub}, nil
}

// WatchApproval is a free log subscription operation binding the contract event 0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925.
//
// Solidity: event Approval(address indexed owner, address indexed spender, uint256 value)
func (_YellowToken *YellowTokenFilterer) WatchApproval(opts *bind.WatchOpts, sink chan<- *YellowTokenApproval, owner []common.Address, spender []common.Address) (event.Subscription, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var spenderRule []interface{}
	for _, spenderItem := range spender {
		spenderRule = append(spenderRule, spenderItem)
	}

	logs, sub, err := _YellowToken.contract.WatchLogs(opts, "Approval", ownerRule, spenderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(YellowTokenApproval)
				if err := _YellowToken.contract.UnpackLog(event, "Approval", log); err != nil {
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
func (_YellowToken *YellowTokenFilterer) ParseApproval(log types.Log) (*YellowTokenApproval, error) {
	event := new(YellowTokenApproval)
	if err := _YellowToken.contract.UnpackLog(event, "Approval", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// YellowTokenEIP712DomainChangedIterator is returned from FilterEIP712DomainChanged and is used to iterate over the raw logs and unpacked data for EIP712DomainChanged events raised by the YellowToken contract.
type YellowTokenEIP712DomainChangedIterator struct {
	Event *YellowTokenEIP712DomainChanged // Event containing the contract specifics and raw log

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
func (it *YellowTokenEIP712DomainChangedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(YellowTokenEIP712DomainChanged)
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
		it.Event = new(YellowTokenEIP712DomainChanged)
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
func (it *YellowTokenEIP712DomainChangedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *YellowTokenEIP712DomainChangedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// YellowTokenEIP712DomainChanged represents a EIP712DomainChanged event raised by the YellowToken contract.
type YellowTokenEIP712DomainChanged struct {
	Raw types.Log // Blockchain specific contextual infos
}

// FilterEIP712DomainChanged is a free log retrieval operation binding the contract event 0x0a6387c9ea3628b88a633bb4f3b151770f70085117a15f9bf3787cda53f13d31.
//
// Solidity: event EIP712DomainChanged()
func (_YellowToken *YellowTokenFilterer) FilterEIP712DomainChanged(opts *bind.FilterOpts) (*YellowTokenEIP712DomainChangedIterator, error) {

	logs, sub, err := _YellowToken.contract.FilterLogs(opts, "EIP712DomainChanged")
	if err != nil {
		return nil, err
	}
	return &YellowTokenEIP712DomainChangedIterator{contract: _YellowToken.contract, event: "EIP712DomainChanged", logs: logs, sub: sub}, nil
}

// WatchEIP712DomainChanged is a free log subscription operation binding the contract event 0x0a6387c9ea3628b88a633bb4f3b151770f70085117a15f9bf3787cda53f13d31.
//
// Solidity: event EIP712DomainChanged()
func (_YellowToken *YellowTokenFilterer) WatchEIP712DomainChanged(opts *bind.WatchOpts, sink chan<- *YellowTokenEIP712DomainChanged) (event.Subscription, error) {

	logs, sub, err := _YellowToken.contract.WatchLogs(opts, "EIP712DomainChanged")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(YellowTokenEIP712DomainChanged)
				if err := _YellowToken.contract.UnpackLog(event, "EIP712DomainChanged", log); err != nil {
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
func (_YellowToken *YellowTokenFilterer) ParseEIP712DomainChanged(log types.Log) (*YellowTokenEIP712DomainChanged, error) {
	event := new(YellowTokenEIP712DomainChanged)
	if err := _YellowToken.contract.UnpackLog(event, "EIP712DomainChanged", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// YellowTokenTransferIterator is returned from FilterTransfer and is used to iterate over the raw logs and unpacked data for Transfer events raised by the YellowToken contract.
type YellowTokenTransferIterator struct {
	Event *YellowTokenTransfer // Event containing the contract specifics and raw log

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
func (it *YellowTokenTransferIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(YellowTokenTransfer)
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
		it.Event = new(YellowTokenTransfer)
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
func (it *YellowTokenTransferIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *YellowTokenTransferIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// YellowTokenTransfer represents a Transfer event raised by the YellowToken contract.
type YellowTokenTransfer struct {
	From  common.Address
	To    common.Address
	Value *big.Int
	Raw   types.Log // Blockchain specific contextual infos
}

// FilterTransfer is a free log retrieval operation binding the contract event 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef.
//
// Solidity: event Transfer(address indexed from, address indexed to, uint256 value)
func (_YellowToken *YellowTokenFilterer) FilterTransfer(opts *bind.FilterOpts, from []common.Address, to []common.Address) (*YellowTokenTransferIterator, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _YellowToken.contract.FilterLogs(opts, "Transfer", fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return &YellowTokenTransferIterator{contract: _YellowToken.contract, event: "Transfer", logs: logs, sub: sub}, nil
}

// WatchTransfer is a free log subscription operation binding the contract event 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef.
//
// Solidity: event Transfer(address indexed from, address indexed to, uint256 value)
func (_YellowToken *YellowTokenFilterer) WatchTransfer(opts *bind.WatchOpts, sink chan<- *YellowTokenTransfer, from []common.Address, to []common.Address) (event.Subscription, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _YellowToken.contract.WatchLogs(opts, "Transfer", fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(YellowTokenTransfer)
				if err := _YellowToken.contract.UnpackLog(event, "Transfer", log); err != nil {
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
func (_YellowToken *YellowTokenFilterer) ParseTransfer(log types.Log) (*YellowTokenTransfer, error) {
	event := new(YellowTokenTransfer)
	if err := _YellowToken.contract.UnpackLog(event, "Transfer", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

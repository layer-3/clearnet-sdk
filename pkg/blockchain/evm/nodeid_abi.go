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

// NodeIDTerms is an auto generated low-level Go binding around an user-defined struct.
type NodeIDTerms struct {
	MintedAt            uint64
	VestingPeriod       uint64
	MinActivationAmount *big.Int
}

// NodeIDMetaData contains all meta data concerning the NodeID contract.
var NodeIDMetaData = &bind.MetaData{
	ABI: "[{\"type\":\"constructor\",\"inputs\":[{\"name\":\"_owner\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"MAX_NODES\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"approve\",\"inputs\":[{\"name\":\"to\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"tokenId\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"availableSlots\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"balanceOf\",\"inputs\":[{\"name\":\"owner\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"baseTokenURI\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"string\",\"internalType\":\"string\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getApproved\",\"inputs\":[{\"name\":\"tokenId\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"isApprovedForAll\",\"inputs\":[{\"name\":\"owner\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"operator\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"mint\",\"inputs\":[{\"name\":\"to\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"minActivationAmount\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"vestingPeriod\",\"type\":\"uint64\",\"internalType\":\"uint64\"}],\"outputs\":[{\"name\":\"tokenId\",\"type\":\"uint32\",\"internalType\":\"uint32\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"mintBatch\",\"inputs\":[{\"name\":\"recipients\",\"type\":\"address[]\",\"internalType\":\"address[]\"},{\"name\":\"minActivationAmount\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"vestingPeriod\",\"type\":\"uint64\",\"internalType\":\"uint64\"}],\"outputs\":[{\"name\":\"firstTokenId\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"lastTokenId\",\"type\":\"uint32\",\"internalType\":\"uint32\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"mintToRegistry\",\"inputs\":[{\"name\":\"minActivationAmount\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"vestingPeriod\",\"type\":\"uint64\",\"internalType\":\"uint64\"}],\"outputs\":[{\"name\":\"tokenId\",\"type\":\"uint32\",\"internalType\":\"uint32\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"name\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"string\",\"internalType\":\"string\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"owner\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"ownerOf\",\"inputs\":[{\"name\":\"tokenId\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"registry\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"safeTransferFrom\",\"inputs\":[{\"name\":\"from\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"to\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"tokenId\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"safeTransferFrom\",\"inputs\":[{\"name\":\"from\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"to\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"tokenId\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"data\",\"type\":\"bytes\",\"internalType\":\"bytes\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"setApprovalForAll\",\"inputs\":[{\"name\":\"operator\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"approved\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"setBaseTokenURI\",\"inputs\":[{\"name\":\"baseURI\",\"type\":\"string\",\"internalType\":\"string\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"setRegistry\",\"inputs\":[{\"name\":\"newRegistry\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"supportsInterface\",\"inputs\":[{\"name\":\"interfaceId\",\"type\":\"bytes4\",\"internalType\":\"bytes4\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"symbol\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"string\",\"internalType\":\"string\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"termsOf\",\"inputs\":[{\"name\":\"tokenId\",\"type\":\"uint32\",\"internalType\":\"uint32\"}],\"outputs\":[{\"name\":\"\",\"type\":\"tuple\",\"internalType\":\"structNodeID.Terms\",\"components\":[{\"name\":\"mintedAt\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"vestingPeriod\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"minActivationAmount\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"tokenURI\",\"inputs\":[{\"name\":\"tokenId\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"string\",\"internalType\":\"string\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"transferFrom\",\"inputs\":[{\"name\":\"from\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"to\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"tokenId\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"transferOwnership\",\"inputs\":[{\"name\":\"newOwner\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"event\",\"name\":\"Approval\",\"inputs\":[{\"name\":\"owner\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"approved\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"tokenId\",\"type\":\"uint256\",\"indexed\":true,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"ApprovalForAll\",\"inputs\":[{\"name\":\"owner\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"operator\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"approved\",\"type\":\"bool\",\"indexed\":false,\"internalType\":\"bool\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"OwnershipTransferred\",\"inputs\":[{\"name\":\"oldOwner\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"newOwner\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"RegistryUpdated\",\"inputs\":[{\"name\":\"oldRegistry\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"newRegistry\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"SlotsMinted\",\"inputs\":[{\"name\":\"by\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"firstTokenId\",\"type\":\"uint32\",\"indexed\":true,\"internalType\":\"uint32\"},{\"name\":\"lastTokenId\",\"type\":\"uint32\",\"indexed\":true,\"internalType\":\"uint32\"},{\"name\":\"minActivationAmount\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"},{\"name\":\"vestingPeriod\",\"type\":\"uint64\",\"indexed\":false,\"internalType\":\"uint64\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"Transfer\",\"inputs\":[{\"name\":\"from\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"to\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"tokenId\",\"type\":\"uint256\",\"indexed\":true,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"error\",\"name\":\"AllSlotsMinted\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"DirectRegistryTransfer\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"ERC721IncorrectOwner\",\"inputs\":[{\"name\":\"sender\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"tokenId\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"owner\",\"type\":\"address\",\"internalType\":\"address\"}]},{\"type\":\"error\",\"name\":\"ERC721InsufficientApproval\",\"inputs\":[{\"name\":\"operator\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"tokenId\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]},{\"type\":\"error\",\"name\":\"ERC721InvalidApprover\",\"inputs\":[{\"name\":\"approver\",\"type\":\"address\",\"internalType\":\"address\"}]},{\"type\":\"error\",\"name\":\"ERC721InvalidOperator\",\"inputs\":[{\"name\":\"operator\",\"type\":\"address\",\"internalType\":\"address\"}]},{\"type\":\"error\",\"name\":\"ERC721InvalidOwner\",\"inputs\":[{\"name\":\"owner\",\"type\":\"address\",\"internalType\":\"address\"}]},{\"type\":\"error\",\"name\":\"ERC721InvalidReceiver\",\"inputs\":[{\"name\":\"receiver\",\"type\":\"address\",\"internalType\":\"address\"}]},{\"type\":\"error\",\"name\":\"ERC721InvalidSender\",\"inputs\":[{\"name\":\"sender\",\"type\":\"address\",\"internalType\":\"address\"}]},{\"type\":\"error\",\"name\":\"ERC721NonexistentToken\",\"inputs\":[{\"name\":\"tokenId\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]},{\"type\":\"error\",\"name\":\"NotOwner\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"NotRegistry\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"ZeroAddress\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"ZeroCount\",\"inputs\":[]}]",
	Bin: "0x60806040523461037a57611b496020813803918261001c8161037e565b93849283398101031261037a57516001600160a01b0381169081900361037a57610046604061037e565b90600e82526d10db19585c939bd9194814db1bdd60921b602083015261006c604061037e565b600681526510d394d313d560d21b602082015282519091906001600160401b038111610283575f54600181811c91168015610370575b602082101461026557601f8111610303575b506020601f82116001146102a257819293945f92610297575b50508160011b915f199060031b1c1916175f555b81516001600160401b03811161028357600154600181811c91168015610279575b602082101461026557601f81116101f7575b50602092601f821160011461019657928192935f9261018b575b50508160011b915f199060031b1c1916176001555b600163ffffffff196009541617600955801561017c57600680546001600160a01b0319169190911790556040516117a590816103a48239f35b63d92e233d60e01b5f5260045ffd5b015190505f8061012e565b601f1982169360015f52805f20915f5b8681106101df57508360019596106101c7575b505050811b01600155610143565b01515f1960f88460031b161c191690555f80806101b9565b919260206001819286850151815501940192016101a6565b818111156101145760015f52601f820160051c7fb10e2d527612073b26eecdfd717e6a320cf44b4afac2b0732d9fcbe2b7fa0cf66020841061025d575b81601f9101920160051c03905f5b828110610250575050610114565b5f82820155600101610242565b5f9150610234565b634e487b7160e01b5f52602260045260245ffd5b90607f1690610102565b634e487b7160e01b5f52604160045260245ffd5b015190505f806100cd565b601f198216905f8052805f20915f5b8181106102eb575095836001959697106102d3575b505050811b015f556100e1565b01515f1960f88460031b161c191690555f80806102c6565b9192602060018192868b0151815501940192016102b1565b818111156100b4575f8052601f820160051c7f290decd9548b62a8d60345a988386fc84ba6bc95484008f6362f93160ef3e56360208410610368575b81601f9101920160051c03905f5b82811061035b5750506100b4565b5f8282015560010161034d565b5f915061033f565b90607f16906100a2565b5f80fd5b6040519190601f01601f191682016001600160401b038111838210176102835760405256fe6080806040526004361015610012575f80fd5b5f3560e01c90816301ffc9a714610d3b5750806306fdde0314610c99578063081812fc14610c5d578063095ea7b314610b7357806323b872dd14610b5c57806330176e13146109f657806342842e0e146109cd5780636352211e1461099d57806370a082311461094c5780637b103999146109245780638b3d35ae1461088e5780638da5cb5b146108665780638eeda103146107cc5780638f16e1cd146107af57806395d89b411461070a578063a22cb4651461066f578063a91ee0dc146105fc578063b88d4fde14610573578063c87b56dd14610554578063d547cfb714610485578063e6fb38131461042a578063e8804a2b1461033f578063e985e9c5146102e8578063f2fde38b146102665763ff875f031461012f575f80fd5b3461024e57606036600319011261024e576004356001600160401b03811161024e573660238201121561024e578060040135906001600160401b038211610252578160051b9060208201926101876040519485610e61565b8352602460208401928201019036821161024e57602401915b81831061022e57604063ffffffff61021f60243582817f9902251bf2894876d6d1dc26c5e2005e75018334706538cb7bf283598aefc42b6101f38b6101e3610e30565b9586916101ee6114e5565b611515565b93169586931694859488519182913395839092916001600160401b036020916040840195845216910152565b0390a482519182526020820152f35b82356001600160a01b038116810361024e578152602092830192016101a0565b5f80fd5b634e487b7160e01b5f52604160045260245ffd5b3461024e57602036600319011261024e5761027f610dca565b6102876114e5565b6001600160a01b031680156102d957600680546001600160a01b0319811683179091556001600160a01b03167f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e05f80a3005b63d92e233d60e01b5f5260045ffd5b3461024e57604036600319011261024e57610301610dca565b610309610de0565b9060018060a01b03165f52600560205260405f209060018060a01b03165f52602052602060ff60405f2054166040519015158152f35b3461024e57604036600319011261024e576004356024356001600160401b038116810361024e576007546001600160a01b031633811480610421575b15610412576104097f9902251bf2894876d6d1dc26c5e2005e75018334706538cb7bf283598aefc42b9260209463ffffffff6103df83836040978851906103c28a83610e61565b60018252601f198a01368d8401376103d9826110d6565b52611515565b5016948593849386519182913395839092916001600160401b036020916040840195845216910152565b0390a451908152f35b633217675b60e21b5f5260045ffd5b5080151561037b565b3461024e575f36600319011261024e575f1963ffffffff600954160163ffffffff81116104715763ffffffff16620100000362010000811161047157602090604051908152f35b634e487b7160e01b5f52601160045260245ffd5b3461024e575f36600319011261024e576040515f6008546104a581610e9d565b808452906001811690811561053057506001146104e5575b6104e1836104cd81850382610e61565b604051918291602083526020830190610da6565b0390f35b60085f9081525f5160206117855f395f51905f52939250905b808210610516575090915081016020016104cd6104bd565b9192600181602092548385880101520191019092916104fe565b60ff191660208086019190915291151560051b840190910191506104cd90506104bd565b3461024e57602036600319011261024e576104e16104cd60043561124b565b3461024e57608036600319011261024e5761058c610dca565b610594610de0565b606435916001600160401b03831161024e573660238401121561024e578260040135916105c083610e82565b926105ce6040519485610e61565b808452366024828701011161024e576020815f9260246105fa980183880137850101526044359161110b565b005b3461024e57602036600319011261024e57610615610dca565b61061d6114e5565b6001600160a01b031680156102d957600780546001600160a01b0319811683179091556001600160a01b03167f482b97c53e48ffa324a976e2738053e9aff6eee04d8aac63b10e19411d869b825f80a3005b3461024e57604036600319011261024e57610688610dca565b6024359081151580920361024e576001600160a01b03169081156106f757335f52600560205260405f20825f5260205260405f2060ff1981541660ff83161790556040519081527f17307eab39ab6107e8899845ad3d59bd9653f200f220920489ca2b5937696c3160203392a3005b50630b61174360e31b5f5260045260245ffd5b3461024e575f36600319011261024e576040515f60015461072a81610e9d565b80845290600181169081156105305750600114610751576104e1836104cd81850382610e61565b60015f9081527fb10e2d527612073b26eecdfd717e6a320cf44b4afac2b0732d9fcbe2b7fa0cf6939250905b808210610795575090915081016020016104cd6104bd565b91926001816020925483858801015201910190929161077d565b3461024e575f36600319011261024e576020604051620100008152f35b3461024e57602036600319011261024e5760043563ffffffff811680910361024e575f604080516107fc81610e46565b8281528260208201520152610810816114b1565b505f52600a602052606060405f2060405161082a81610e46565b6001600160401b0382546040600183831695868652846020870194841c1684520154930192835260405193845251166020830152516040820152f35b3461024e575f36600319011261024e576006546040516001600160a01b039091168152602090f35b3461024e57606036600319011261024e5760207f9902251bf2894876d6d1dc26c5e2005e75018334706538cb7bf283598aefc42b6108ca610dca565b6104096024356108d8610e30565b906108e16114e5565b63ffffffff6103df83836040978851906108fb8a83610e61565b60018252601f198a01368d840137610912826110d6565b6001600160a01b039091169052611515565b3461024e575f36600319011261024e576007546040516001600160a01b039091168152602090f35b3461024e57602036600319011261024e576001600160a01b0361096d610dca565b16801561098a575f526003602052602060405f2054604051908152f35b6322718ad960e21b5f525f60045260245ffd5b3461024e57602036600319011261024e5760206109bb6004356114b1565b6040516001600160a01b039091168152f35b3461024e576105fa6109de36610df6565b90604051926109ee602085610e61565b5f845261110b565b3461024e57602036600319011261024e576004356001600160401b03811161024e573660238201121561024e5780600401356001600160401b03811161024e57366024828401011161024e57610a4a6114e5565b610a55600854610e9d565b601f8111610b05575b505f601f8211600114610a9a5781925f92610a8c575b50505f19600383901b1c191660019190911b17600855005b602492500101358280610a74565b601f198216925f5160206117855f395f51905f52915f5b858110610aea57508360019510610ace575b505050811b01600855005b01602401355f19600384901b60f8161c19169055828080610ac3565b90926020600181926024878701013581550194019101610ab1565b81811115610a5e57601f820160051c9060208310610b54575b601f82910160051c03905f5b828110610b38575050610a5e565b5f8282015f5160206117855f395f51905f520155600101610b2a565b5f9150610b1e565b3461024e576105fa610b6d36610df6565b91610ed5565b3461024e57604036600319011261024e57610b8c610dca565b602435610b98816114b1565b33151580610c4a575b80610c1d575b610c0a5781906001600160a01b0384811691167f8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b9255f80a45f90815260046020526040902080546001600160a01b0319166001600160a01b03909216919091179055005b63a9fbf51f60e01b5f523360045260245ffd5b506001600160a01b0381165f90815260056020908152604080832033845290915290205460ff1615610ba7565b506001600160a01b038116331415610ba1565b3461024e57602036600319011261024e57600435610c7a816114b1565b505f526004602052602060018060a01b0360405f205416604051908152f35b3461024e575f36600319011261024e576040515f5f54610cb881610e9d565b80845290600181169081156105305750600114610cdf576104e1836104cd81850382610e61565b5f8080527f290decd9548b62a8d60345a988386fc84ba6bc95484008f6362f93160ef3e563939250905b808210610d21575090915081016020016104cd6104bd565b919260018160209254838588010152019101909291610d09565b3461024e57602036600319011261024e576004359063ffffffff60e01b821680920361024e576020916380ac58cd60e01b8114908115610d95575b8115610d84575b5015158152f35b6301ffc9a760e01b14905083610d7d565b635b5e139f60e01b81149150610d76565b805180835260209291819084018484015e5f828201840152601f01601f1916010190565b600435906001600160a01b038216820361024e57565b602435906001600160a01b038216820361024e57565b606090600319011261024e576004356001600160a01b038116810361024e57906024356001600160a01b038116810361024e579060443590565b604435906001600160401b038216820361024e57565b606081019081106001600160401b0382111761025257604052565b90601f801991011681019081106001600160401b0382111761025257604052565b6001600160401b03811161025257601f01601f191660200190565b90600182811c92168015610ecb575b6020831014610eb757565b634e487b7160e01b5f52602260045260245ffd5b91607f1691610eac565b6001600160a01b03909116919082156110c3575f828152600260205260409020546001600160a01b03161515806110af575b8061109a575b61108b575f828152600260205260409020546001600160a01b031692829033151580610ff6575b5084610fc3575b805f52600360205260405f2060018154019055815f52600260205260405f20816bffffffffffffffffffffffff60a01b825416179055847fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef5f80a46001600160a01b0316808303610fab57505050565b6364283d7b60e01b5f5260045260245260445260645ffd5b5f82815260046020526040902080546001600160a01b0319169055845f52600360205260405f205f198154019055610f3b565b9091508061103a575b1561100c5782905f610f34565b828461102457637e27328960e01b5f5260045260245ffd5b63177e802f60e01b5f523360045260245260445ffd5b503384148015611069575b80610fff57505f838152600460205260409020546001600160a01b03163314610fff565b505f84815260056020908152604080832033845290915290205460ff16611045565b63588e7fef60e11b5f5260045ffd5b506007546001600160a01b0316331415610f0d565b506007546001600160a01b03168314610f07565b633250574960e11b5f525f60045260245ffd5b8051156110e35760200190565b634e487b7160e01b5f52603260045260245ffd5b80518210156110e35760209160051b010190565b9291611118818386610ed5565b813b611125575b50505050565b604051630a85bd0160e11b81523360048201526001600160a01b0394851660248201526044810191909152608060648201529216919060209082908190611170906084830190610da6565b03815f865af15f9181611206575b506111d357503d156111cc573d61119481610e82565b906111a26040519283610e61565b81523d5f602083013e5b805190816111c75782633250574960e11b5f5260045260245ffd5b602001fd5b60606111ac565b6001600160e01b03191663757a42ff60e11b016111f457505f80808061111f565b633250574960e11b5f5260045260245ffd5b9091506020813d602011611243575b8161122260209383610e61565b8101031261024e57516001600160e01b03198116810361024e57905f61117e565b3d9150611215565b611254816114b1565b506008549061126282610e9d565b1561149b5780815f9272184f03e93ff9f4daa797ed6e38ed64bf6a1f0160401b811015611475575b50806d04ee2d6d415b85acef8100000000600a92101561145a575b662386f26fc10000811015611446575b6305f5e100811015611435575b612710811015611426575b6064811015611418575b101561140e575b6001820190600a60216113096112f385610e82565b946113016040519687610e61565b808652610e82565b602085019590601f19013687378401015b5f1901916f181899199a1a9b1b9c1cb0b131b232b360811b8282061a835304801561134857600a909161131a565b50506040519283915f9161135b81610e9d565b90600181169081156113ea575060011461139d575b50926005929161139a94518092825e0164173539b7b760d91b815203601a19810184520182610e61565b90565b90915060085f525f5160206117855f395f51905f525f905b8282106113cc57505082016020019061139a611370565b60209192939450806001915483858a010152019101859392916113b5565b60ff19166020808701919091528215159092028501909101925061139a9050611370565b90600101906112de565b6064600291049301926112d7565b612710600491049301926112cd565b6305f5e100600891049301926112c2565b662386f26fc10000601091049301926112b5565b6d04ee2d6d415b85acef8100000000602091049301926112a5565b6040935072184f03e93ff9f4daa797ed6e38ed64bf6a1f0160401b90049050600a61128a565b50506040516114ab602082610e61565b5f815290565b5f818152600260205260409020546001600160a01b03169081156114d3575090565b637e27328960e01b5f5260045260245ffd5b6006546001600160a01b031633036114f957565b6330cd747160e01b5f5260045ffd5b9190820180921161047157565b9291928051156117755763ffffffff6009541691611534825184611508565b925f19840184811161047157620100008111611766579394426001600160401b031694905f5b8551811015611745576001600160a01b0361157582886110f7565b5116156102d95763ffffffff61158b8286611508565b16906001600160a01b0361159f82896110f7565b511680156110c3575f838152600260205260409020546001600160a01b0316151580611731575b8061171c575b61108b575f838152600260205260409020546001600160a01b0316801515918490836116e9575b5f818152600360209081526040808320805460010190558483526002909152812080546001600160a01b0319166001600160a01b03841617905583907fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef9080a4506116d657600191826040519161166983610e46565b8a83528c6001600160401b03602085019116815260408401918a83525f52600a6020526001600160401b0360405f209451166fffffffffffffffff00000000000000008554925160401b16916fffffffffffffffffffffffffffffffff191617178355519101550161155a565b6339e3563760e11b5f525f60045260245ffd5b5f82815260046020526040902080546001600160a01b0319169055825f52600360205260405f205f1981540190556115f3565b506007546001600160a01b03163314156115cc565b506007546001600160a01b031681146115c6565b5095919350955063ffffffff93508391501682196009541617600955921690565b6304710b1360e11b5f5260045ffd5b63011ee73b60e21b5f5260045ffdfef3f7a9fe364faab93b216da50a3214154f22a0a2b415b23a84c8169e8b636ee3",
}

// NodeIDABI is the input ABI used to generate the binding from.
// Deprecated: Use NodeIDMetaData.ABI instead.
var NodeIDABI = NodeIDMetaData.ABI

// NodeIDBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use NodeIDMetaData.Bin instead.
var NodeIDBin = NodeIDMetaData.Bin

// DeployNodeID deploys a new Ethereum contract, binding an instance of NodeID to it.
func DeployNodeID(auth *bind.TransactOpts, backend bind.ContractBackend, _owner common.Address) (common.Address, *types.Transaction, *NodeID, error) {
	parsed, err := NodeIDMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(NodeIDBin), backend, _owner)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &NodeID{NodeIDCaller: NodeIDCaller{contract: contract}, NodeIDTransactor: NodeIDTransactor{contract: contract}, NodeIDFilterer: NodeIDFilterer{contract: contract}}, nil
}

// NodeID is an auto generated Go binding around an Ethereum contract.
type NodeID struct {
	NodeIDCaller     // Read-only binding to the contract
	NodeIDTransactor // Write-only binding to the contract
	NodeIDFilterer   // Log filterer for contract events
}

// NodeIDCaller is an auto generated read-only Go binding around an Ethereum contract.
type NodeIDCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// NodeIDTransactor is an auto generated write-only Go binding around an Ethereum contract.
type NodeIDTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// NodeIDFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type NodeIDFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// NodeIDSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type NodeIDSession struct {
	Contract     *NodeID           // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// NodeIDCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type NodeIDCallerSession struct {
	Contract *NodeIDCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts // Call options to use throughout this session
}

// NodeIDTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type NodeIDTransactorSession struct {
	Contract     *NodeIDTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// NodeIDRaw is an auto generated low-level Go binding around an Ethereum contract.
type NodeIDRaw struct {
	Contract *NodeID // Generic contract binding to access the raw methods on
}

// NodeIDCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type NodeIDCallerRaw struct {
	Contract *NodeIDCaller // Generic read-only contract binding to access the raw methods on
}

// NodeIDTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type NodeIDTransactorRaw struct {
	Contract *NodeIDTransactor // Generic write-only contract binding to access the raw methods on
}

// NewNodeID creates a new instance of NodeID, bound to a specific deployed contract.
func NewNodeID(address common.Address, backend bind.ContractBackend) (*NodeID, error) {
	contract, err := bindNodeID(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &NodeID{NodeIDCaller: NodeIDCaller{contract: contract}, NodeIDTransactor: NodeIDTransactor{contract: contract}, NodeIDFilterer: NodeIDFilterer{contract: contract}}, nil
}

// NewNodeIDCaller creates a new read-only instance of NodeID, bound to a specific deployed contract.
func NewNodeIDCaller(address common.Address, caller bind.ContractCaller) (*NodeIDCaller, error) {
	contract, err := bindNodeID(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &NodeIDCaller{contract: contract}, nil
}

// NewNodeIDTransactor creates a new write-only instance of NodeID, bound to a specific deployed contract.
func NewNodeIDTransactor(address common.Address, transactor bind.ContractTransactor) (*NodeIDTransactor, error) {
	contract, err := bindNodeID(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &NodeIDTransactor{contract: contract}, nil
}

// NewNodeIDFilterer creates a new log filterer instance of NodeID, bound to a specific deployed contract.
func NewNodeIDFilterer(address common.Address, filterer bind.ContractFilterer) (*NodeIDFilterer, error) {
	contract, err := bindNodeID(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &NodeIDFilterer{contract: contract}, nil
}

// bindNodeID binds a generic wrapper to an already deployed contract.
func bindNodeID(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := NodeIDMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_NodeID *NodeIDRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _NodeID.Contract.NodeIDCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_NodeID *NodeIDRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _NodeID.Contract.NodeIDTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_NodeID *NodeIDRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _NodeID.Contract.NodeIDTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_NodeID *NodeIDCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _NodeID.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_NodeID *NodeIDTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _NodeID.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_NodeID *NodeIDTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _NodeID.Contract.contract.Transact(opts, method, params...)
}

// MAXNODES is a free data retrieval call binding the contract method 0x8f16e1cd.
//
// Solidity: function MAX_NODES() view returns(uint256)
func (_NodeID *NodeIDCaller) MAXNODES(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _NodeID.contract.Call(opts, &out, "MAX_NODES")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MAXNODES is a free data retrieval call binding the contract method 0x8f16e1cd.
//
// Solidity: function MAX_NODES() view returns(uint256)
func (_NodeID *NodeIDSession) MAXNODES() (*big.Int, error) {
	return _NodeID.Contract.MAXNODES(&_NodeID.CallOpts)
}

// MAXNODES is a free data retrieval call binding the contract method 0x8f16e1cd.
//
// Solidity: function MAX_NODES() view returns(uint256)
func (_NodeID *NodeIDCallerSession) MAXNODES() (*big.Int, error) {
	return _NodeID.Contract.MAXNODES(&_NodeID.CallOpts)
}

// AvailableSlots is a free data retrieval call binding the contract method 0xe6fb3813.
//
// Solidity: function availableSlots() view returns(uint256)
func (_NodeID *NodeIDCaller) AvailableSlots(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _NodeID.contract.Call(opts, &out, "availableSlots")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// AvailableSlots is a free data retrieval call binding the contract method 0xe6fb3813.
//
// Solidity: function availableSlots() view returns(uint256)
func (_NodeID *NodeIDSession) AvailableSlots() (*big.Int, error) {
	return _NodeID.Contract.AvailableSlots(&_NodeID.CallOpts)
}

// AvailableSlots is a free data retrieval call binding the contract method 0xe6fb3813.
//
// Solidity: function availableSlots() view returns(uint256)
func (_NodeID *NodeIDCallerSession) AvailableSlots() (*big.Int, error) {
	return _NodeID.Contract.AvailableSlots(&_NodeID.CallOpts)
}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address owner) view returns(uint256)
func (_NodeID *NodeIDCaller) BalanceOf(opts *bind.CallOpts, owner common.Address) (*big.Int, error) {
	var out []interface{}
	err := _NodeID.contract.Call(opts, &out, "balanceOf", owner)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address owner) view returns(uint256)
func (_NodeID *NodeIDSession) BalanceOf(owner common.Address) (*big.Int, error) {
	return _NodeID.Contract.BalanceOf(&_NodeID.CallOpts, owner)
}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address owner) view returns(uint256)
func (_NodeID *NodeIDCallerSession) BalanceOf(owner common.Address) (*big.Int, error) {
	return _NodeID.Contract.BalanceOf(&_NodeID.CallOpts, owner)
}

// BaseTokenURI is a free data retrieval call binding the contract method 0xd547cfb7.
//
// Solidity: function baseTokenURI() view returns(string)
func (_NodeID *NodeIDCaller) BaseTokenURI(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _NodeID.contract.Call(opts, &out, "baseTokenURI")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// BaseTokenURI is a free data retrieval call binding the contract method 0xd547cfb7.
//
// Solidity: function baseTokenURI() view returns(string)
func (_NodeID *NodeIDSession) BaseTokenURI() (string, error) {
	return _NodeID.Contract.BaseTokenURI(&_NodeID.CallOpts)
}

// BaseTokenURI is a free data retrieval call binding the contract method 0xd547cfb7.
//
// Solidity: function baseTokenURI() view returns(string)
func (_NodeID *NodeIDCallerSession) BaseTokenURI() (string, error) {
	return _NodeID.Contract.BaseTokenURI(&_NodeID.CallOpts)
}

// GetApproved is a free data retrieval call binding the contract method 0x081812fc.
//
// Solidity: function getApproved(uint256 tokenId) view returns(address)
func (_NodeID *NodeIDCaller) GetApproved(opts *bind.CallOpts, tokenId *big.Int) (common.Address, error) {
	var out []interface{}
	err := _NodeID.contract.Call(opts, &out, "getApproved", tokenId)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetApproved is a free data retrieval call binding the contract method 0x081812fc.
//
// Solidity: function getApproved(uint256 tokenId) view returns(address)
func (_NodeID *NodeIDSession) GetApproved(tokenId *big.Int) (common.Address, error) {
	return _NodeID.Contract.GetApproved(&_NodeID.CallOpts, tokenId)
}

// GetApproved is a free data retrieval call binding the contract method 0x081812fc.
//
// Solidity: function getApproved(uint256 tokenId) view returns(address)
func (_NodeID *NodeIDCallerSession) GetApproved(tokenId *big.Int) (common.Address, error) {
	return _NodeID.Contract.GetApproved(&_NodeID.CallOpts, tokenId)
}

// IsApprovedForAll is a free data retrieval call binding the contract method 0xe985e9c5.
//
// Solidity: function isApprovedForAll(address owner, address operator) view returns(bool)
func (_NodeID *NodeIDCaller) IsApprovedForAll(opts *bind.CallOpts, owner common.Address, operator common.Address) (bool, error) {
	var out []interface{}
	err := _NodeID.contract.Call(opts, &out, "isApprovedForAll", owner, operator)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsApprovedForAll is a free data retrieval call binding the contract method 0xe985e9c5.
//
// Solidity: function isApprovedForAll(address owner, address operator) view returns(bool)
func (_NodeID *NodeIDSession) IsApprovedForAll(owner common.Address, operator common.Address) (bool, error) {
	return _NodeID.Contract.IsApprovedForAll(&_NodeID.CallOpts, owner, operator)
}

// IsApprovedForAll is a free data retrieval call binding the contract method 0xe985e9c5.
//
// Solidity: function isApprovedForAll(address owner, address operator) view returns(bool)
func (_NodeID *NodeIDCallerSession) IsApprovedForAll(owner common.Address, operator common.Address) (bool, error) {
	return _NodeID.Contract.IsApprovedForAll(&_NodeID.CallOpts, owner, operator)
}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_NodeID *NodeIDCaller) Name(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _NodeID.contract.Call(opts, &out, "name")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_NodeID *NodeIDSession) Name() (string, error) {
	return _NodeID.Contract.Name(&_NodeID.CallOpts)
}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_NodeID *NodeIDCallerSession) Name() (string, error) {
	return _NodeID.Contract.Name(&_NodeID.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_NodeID *NodeIDCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _NodeID.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_NodeID *NodeIDSession) Owner() (common.Address, error) {
	return _NodeID.Contract.Owner(&_NodeID.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_NodeID *NodeIDCallerSession) Owner() (common.Address, error) {
	return _NodeID.Contract.Owner(&_NodeID.CallOpts)
}

// OwnerOf is a free data retrieval call binding the contract method 0x6352211e.
//
// Solidity: function ownerOf(uint256 tokenId) view returns(address)
func (_NodeID *NodeIDCaller) OwnerOf(opts *bind.CallOpts, tokenId *big.Int) (common.Address, error) {
	var out []interface{}
	err := _NodeID.contract.Call(opts, &out, "ownerOf", tokenId)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// OwnerOf is a free data retrieval call binding the contract method 0x6352211e.
//
// Solidity: function ownerOf(uint256 tokenId) view returns(address)
func (_NodeID *NodeIDSession) OwnerOf(tokenId *big.Int) (common.Address, error) {
	return _NodeID.Contract.OwnerOf(&_NodeID.CallOpts, tokenId)
}

// OwnerOf is a free data retrieval call binding the contract method 0x6352211e.
//
// Solidity: function ownerOf(uint256 tokenId) view returns(address)
func (_NodeID *NodeIDCallerSession) OwnerOf(tokenId *big.Int) (common.Address, error) {
	return _NodeID.Contract.OwnerOf(&_NodeID.CallOpts, tokenId)
}

// Registry is a free data retrieval call binding the contract method 0x7b103999.
//
// Solidity: function registry() view returns(address)
func (_NodeID *NodeIDCaller) Registry(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _NodeID.contract.Call(opts, &out, "registry")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Registry is a free data retrieval call binding the contract method 0x7b103999.
//
// Solidity: function registry() view returns(address)
func (_NodeID *NodeIDSession) Registry() (common.Address, error) {
	return _NodeID.Contract.Registry(&_NodeID.CallOpts)
}

// Registry is a free data retrieval call binding the contract method 0x7b103999.
//
// Solidity: function registry() view returns(address)
func (_NodeID *NodeIDCallerSession) Registry() (common.Address, error) {
	return _NodeID.Contract.Registry(&_NodeID.CallOpts)
}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 interfaceId) view returns(bool)
func (_NodeID *NodeIDCaller) SupportsInterface(opts *bind.CallOpts, interfaceId [4]byte) (bool, error) {
	var out []interface{}
	err := _NodeID.contract.Call(opts, &out, "supportsInterface", interfaceId)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 interfaceId) view returns(bool)
func (_NodeID *NodeIDSession) SupportsInterface(interfaceId [4]byte) (bool, error) {
	return _NodeID.Contract.SupportsInterface(&_NodeID.CallOpts, interfaceId)
}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 interfaceId) view returns(bool)
func (_NodeID *NodeIDCallerSession) SupportsInterface(interfaceId [4]byte) (bool, error) {
	return _NodeID.Contract.SupportsInterface(&_NodeID.CallOpts, interfaceId)
}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() view returns(string)
func (_NodeID *NodeIDCaller) Symbol(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _NodeID.contract.Call(opts, &out, "symbol")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() view returns(string)
func (_NodeID *NodeIDSession) Symbol() (string, error) {
	return _NodeID.Contract.Symbol(&_NodeID.CallOpts)
}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() view returns(string)
func (_NodeID *NodeIDCallerSession) Symbol() (string, error) {
	return _NodeID.Contract.Symbol(&_NodeID.CallOpts)
}

// TermsOf is a free data retrieval call binding the contract method 0x8eeda103.
//
// Solidity: function termsOf(uint32 tokenId) view returns((uint64,uint64,uint256))
func (_NodeID *NodeIDCaller) TermsOf(opts *bind.CallOpts, tokenId uint32) (NodeIDTerms, error) {
	var out []interface{}
	err := _NodeID.contract.Call(opts, &out, "termsOf", tokenId)

	if err != nil {
		return *new(NodeIDTerms), err
	}

	out0 := *abi.ConvertType(out[0], new(NodeIDTerms)).(*NodeIDTerms)

	return out0, err

}

// TermsOf is a free data retrieval call binding the contract method 0x8eeda103.
//
// Solidity: function termsOf(uint32 tokenId) view returns((uint64,uint64,uint256))
func (_NodeID *NodeIDSession) TermsOf(tokenId uint32) (NodeIDTerms, error) {
	return _NodeID.Contract.TermsOf(&_NodeID.CallOpts, tokenId)
}

// TermsOf is a free data retrieval call binding the contract method 0x8eeda103.
//
// Solidity: function termsOf(uint32 tokenId) view returns((uint64,uint64,uint256))
func (_NodeID *NodeIDCallerSession) TermsOf(tokenId uint32) (NodeIDTerms, error) {
	return _NodeID.Contract.TermsOf(&_NodeID.CallOpts, tokenId)
}

// TokenURI is a free data retrieval call binding the contract method 0xc87b56dd.
//
// Solidity: function tokenURI(uint256 tokenId) view returns(string)
func (_NodeID *NodeIDCaller) TokenURI(opts *bind.CallOpts, tokenId *big.Int) (string, error) {
	var out []interface{}
	err := _NodeID.contract.Call(opts, &out, "tokenURI", tokenId)

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// TokenURI is a free data retrieval call binding the contract method 0xc87b56dd.
//
// Solidity: function tokenURI(uint256 tokenId) view returns(string)
func (_NodeID *NodeIDSession) TokenURI(tokenId *big.Int) (string, error) {
	return _NodeID.Contract.TokenURI(&_NodeID.CallOpts, tokenId)
}

// TokenURI is a free data retrieval call binding the contract method 0xc87b56dd.
//
// Solidity: function tokenURI(uint256 tokenId) view returns(string)
func (_NodeID *NodeIDCallerSession) TokenURI(tokenId *big.Int) (string, error) {
	return _NodeID.Contract.TokenURI(&_NodeID.CallOpts, tokenId)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address to, uint256 tokenId) returns()
func (_NodeID *NodeIDTransactor) Approve(opts *bind.TransactOpts, to common.Address, tokenId *big.Int) (*types.Transaction, error) {
	return _NodeID.contract.Transact(opts, "approve", to, tokenId)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address to, uint256 tokenId) returns()
func (_NodeID *NodeIDSession) Approve(to common.Address, tokenId *big.Int) (*types.Transaction, error) {
	return _NodeID.Contract.Approve(&_NodeID.TransactOpts, to, tokenId)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address to, uint256 tokenId) returns()
func (_NodeID *NodeIDTransactorSession) Approve(to common.Address, tokenId *big.Int) (*types.Transaction, error) {
	return _NodeID.Contract.Approve(&_NodeID.TransactOpts, to, tokenId)
}

// Mint is a paid mutator transaction binding the contract method 0x8b3d35ae.
//
// Solidity: function mint(address to, uint256 minActivationAmount, uint64 vestingPeriod) returns(uint32 tokenId)
func (_NodeID *NodeIDTransactor) Mint(opts *bind.TransactOpts, to common.Address, minActivationAmount *big.Int, vestingPeriod uint64) (*types.Transaction, error) {
	return _NodeID.contract.Transact(opts, "mint", to, minActivationAmount, vestingPeriod)
}

// Mint is a paid mutator transaction binding the contract method 0x8b3d35ae.
//
// Solidity: function mint(address to, uint256 minActivationAmount, uint64 vestingPeriod) returns(uint32 tokenId)
func (_NodeID *NodeIDSession) Mint(to common.Address, minActivationAmount *big.Int, vestingPeriod uint64) (*types.Transaction, error) {
	return _NodeID.Contract.Mint(&_NodeID.TransactOpts, to, minActivationAmount, vestingPeriod)
}

// Mint is a paid mutator transaction binding the contract method 0x8b3d35ae.
//
// Solidity: function mint(address to, uint256 minActivationAmount, uint64 vestingPeriod) returns(uint32 tokenId)
func (_NodeID *NodeIDTransactorSession) Mint(to common.Address, minActivationAmount *big.Int, vestingPeriod uint64) (*types.Transaction, error) {
	return _NodeID.Contract.Mint(&_NodeID.TransactOpts, to, minActivationAmount, vestingPeriod)
}

// MintBatch is a paid mutator transaction binding the contract method 0xff875f03.
//
// Solidity: function mintBatch(address[] recipients, uint256 minActivationAmount, uint64 vestingPeriod) returns(uint32 firstTokenId, uint32 lastTokenId)
func (_NodeID *NodeIDTransactor) MintBatch(opts *bind.TransactOpts, recipients []common.Address, minActivationAmount *big.Int, vestingPeriod uint64) (*types.Transaction, error) {
	return _NodeID.contract.Transact(opts, "mintBatch", recipients, minActivationAmount, vestingPeriod)
}

// MintBatch is a paid mutator transaction binding the contract method 0xff875f03.
//
// Solidity: function mintBatch(address[] recipients, uint256 minActivationAmount, uint64 vestingPeriod) returns(uint32 firstTokenId, uint32 lastTokenId)
func (_NodeID *NodeIDSession) MintBatch(recipients []common.Address, minActivationAmount *big.Int, vestingPeriod uint64) (*types.Transaction, error) {
	return _NodeID.Contract.MintBatch(&_NodeID.TransactOpts, recipients, minActivationAmount, vestingPeriod)
}

// MintBatch is a paid mutator transaction binding the contract method 0xff875f03.
//
// Solidity: function mintBatch(address[] recipients, uint256 minActivationAmount, uint64 vestingPeriod) returns(uint32 firstTokenId, uint32 lastTokenId)
func (_NodeID *NodeIDTransactorSession) MintBatch(recipients []common.Address, minActivationAmount *big.Int, vestingPeriod uint64) (*types.Transaction, error) {
	return _NodeID.Contract.MintBatch(&_NodeID.TransactOpts, recipients, minActivationAmount, vestingPeriod)
}

// MintToRegistry is a paid mutator transaction binding the contract method 0xe8804a2b.
//
// Solidity: function mintToRegistry(uint256 minActivationAmount, uint64 vestingPeriod) returns(uint32 tokenId)
func (_NodeID *NodeIDTransactor) MintToRegistry(opts *bind.TransactOpts, minActivationAmount *big.Int, vestingPeriod uint64) (*types.Transaction, error) {
	return _NodeID.contract.Transact(opts, "mintToRegistry", minActivationAmount, vestingPeriod)
}

// MintToRegistry is a paid mutator transaction binding the contract method 0xe8804a2b.
//
// Solidity: function mintToRegistry(uint256 minActivationAmount, uint64 vestingPeriod) returns(uint32 tokenId)
func (_NodeID *NodeIDSession) MintToRegistry(minActivationAmount *big.Int, vestingPeriod uint64) (*types.Transaction, error) {
	return _NodeID.Contract.MintToRegistry(&_NodeID.TransactOpts, minActivationAmount, vestingPeriod)
}

// MintToRegistry is a paid mutator transaction binding the contract method 0xe8804a2b.
//
// Solidity: function mintToRegistry(uint256 minActivationAmount, uint64 vestingPeriod) returns(uint32 tokenId)
func (_NodeID *NodeIDTransactorSession) MintToRegistry(minActivationAmount *big.Int, vestingPeriod uint64) (*types.Transaction, error) {
	return _NodeID.Contract.MintToRegistry(&_NodeID.TransactOpts, minActivationAmount, vestingPeriod)
}

// SafeTransferFrom is a paid mutator transaction binding the contract method 0x42842e0e.
//
// Solidity: function safeTransferFrom(address from, address to, uint256 tokenId) returns()
func (_NodeID *NodeIDTransactor) SafeTransferFrom(opts *bind.TransactOpts, from common.Address, to common.Address, tokenId *big.Int) (*types.Transaction, error) {
	return _NodeID.contract.Transact(opts, "safeTransferFrom", from, to, tokenId)
}

// SafeTransferFrom is a paid mutator transaction binding the contract method 0x42842e0e.
//
// Solidity: function safeTransferFrom(address from, address to, uint256 tokenId) returns()
func (_NodeID *NodeIDSession) SafeTransferFrom(from common.Address, to common.Address, tokenId *big.Int) (*types.Transaction, error) {
	return _NodeID.Contract.SafeTransferFrom(&_NodeID.TransactOpts, from, to, tokenId)
}

// SafeTransferFrom is a paid mutator transaction binding the contract method 0x42842e0e.
//
// Solidity: function safeTransferFrom(address from, address to, uint256 tokenId) returns()
func (_NodeID *NodeIDTransactorSession) SafeTransferFrom(from common.Address, to common.Address, tokenId *big.Int) (*types.Transaction, error) {
	return _NodeID.Contract.SafeTransferFrom(&_NodeID.TransactOpts, from, to, tokenId)
}

// SafeTransferFrom0 is a paid mutator transaction binding the contract method 0xb88d4fde.
//
// Solidity: function safeTransferFrom(address from, address to, uint256 tokenId, bytes data) returns()
func (_NodeID *NodeIDTransactor) SafeTransferFrom0(opts *bind.TransactOpts, from common.Address, to common.Address, tokenId *big.Int, data []byte) (*types.Transaction, error) {
	return _NodeID.contract.Transact(opts, "safeTransferFrom0", from, to, tokenId, data)
}

// SafeTransferFrom0 is a paid mutator transaction binding the contract method 0xb88d4fde.
//
// Solidity: function safeTransferFrom(address from, address to, uint256 tokenId, bytes data) returns()
func (_NodeID *NodeIDSession) SafeTransferFrom0(from common.Address, to common.Address, tokenId *big.Int, data []byte) (*types.Transaction, error) {
	return _NodeID.Contract.SafeTransferFrom0(&_NodeID.TransactOpts, from, to, tokenId, data)
}

// SafeTransferFrom0 is a paid mutator transaction binding the contract method 0xb88d4fde.
//
// Solidity: function safeTransferFrom(address from, address to, uint256 tokenId, bytes data) returns()
func (_NodeID *NodeIDTransactorSession) SafeTransferFrom0(from common.Address, to common.Address, tokenId *big.Int, data []byte) (*types.Transaction, error) {
	return _NodeID.Contract.SafeTransferFrom0(&_NodeID.TransactOpts, from, to, tokenId, data)
}

// SetApprovalForAll is a paid mutator transaction binding the contract method 0xa22cb465.
//
// Solidity: function setApprovalForAll(address operator, bool approved) returns()
func (_NodeID *NodeIDTransactor) SetApprovalForAll(opts *bind.TransactOpts, operator common.Address, approved bool) (*types.Transaction, error) {
	return _NodeID.contract.Transact(opts, "setApprovalForAll", operator, approved)
}

// SetApprovalForAll is a paid mutator transaction binding the contract method 0xa22cb465.
//
// Solidity: function setApprovalForAll(address operator, bool approved) returns()
func (_NodeID *NodeIDSession) SetApprovalForAll(operator common.Address, approved bool) (*types.Transaction, error) {
	return _NodeID.Contract.SetApprovalForAll(&_NodeID.TransactOpts, operator, approved)
}

// SetApprovalForAll is a paid mutator transaction binding the contract method 0xa22cb465.
//
// Solidity: function setApprovalForAll(address operator, bool approved) returns()
func (_NodeID *NodeIDTransactorSession) SetApprovalForAll(operator common.Address, approved bool) (*types.Transaction, error) {
	return _NodeID.Contract.SetApprovalForAll(&_NodeID.TransactOpts, operator, approved)
}

// SetBaseTokenURI is a paid mutator transaction binding the contract method 0x30176e13.
//
// Solidity: function setBaseTokenURI(string baseURI) returns()
func (_NodeID *NodeIDTransactor) SetBaseTokenURI(opts *bind.TransactOpts, baseURI string) (*types.Transaction, error) {
	return _NodeID.contract.Transact(opts, "setBaseTokenURI", baseURI)
}

// SetBaseTokenURI is a paid mutator transaction binding the contract method 0x30176e13.
//
// Solidity: function setBaseTokenURI(string baseURI) returns()
func (_NodeID *NodeIDSession) SetBaseTokenURI(baseURI string) (*types.Transaction, error) {
	return _NodeID.Contract.SetBaseTokenURI(&_NodeID.TransactOpts, baseURI)
}

// SetBaseTokenURI is a paid mutator transaction binding the contract method 0x30176e13.
//
// Solidity: function setBaseTokenURI(string baseURI) returns()
func (_NodeID *NodeIDTransactorSession) SetBaseTokenURI(baseURI string) (*types.Transaction, error) {
	return _NodeID.Contract.SetBaseTokenURI(&_NodeID.TransactOpts, baseURI)
}

// SetRegistry is a paid mutator transaction binding the contract method 0xa91ee0dc.
//
// Solidity: function setRegistry(address newRegistry) returns()
func (_NodeID *NodeIDTransactor) SetRegistry(opts *bind.TransactOpts, newRegistry common.Address) (*types.Transaction, error) {
	return _NodeID.contract.Transact(opts, "setRegistry", newRegistry)
}

// SetRegistry is a paid mutator transaction binding the contract method 0xa91ee0dc.
//
// Solidity: function setRegistry(address newRegistry) returns()
func (_NodeID *NodeIDSession) SetRegistry(newRegistry common.Address) (*types.Transaction, error) {
	return _NodeID.Contract.SetRegistry(&_NodeID.TransactOpts, newRegistry)
}

// SetRegistry is a paid mutator transaction binding the contract method 0xa91ee0dc.
//
// Solidity: function setRegistry(address newRegistry) returns()
func (_NodeID *NodeIDTransactorSession) SetRegistry(newRegistry common.Address) (*types.Transaction, error) {
	return _NodeID.Contract.SetRegistry(&_NodeID.TransactOpts, newRegistry)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address from, address to, uint256 tokenId) returns()
func (_NodeID *NodeIDTransactor) TransferFrom(opts *bind.TransactOpts, from common.Address, to common.Address, tokenId *big.Int) (*types.Transaction, error) {
	return _NodeID.contract.Transact(opts, "transferFrom", from, to, tokenId)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address from, address to, uint256 tokenId) returns()
func (_NodeID *NodeIDSession) TransferFrom(from common.Address, to common.Address, tokenId *big.Int) (*types.Transaction, error) {
	return _NodeID.Contract.TransferFrom(&_NodeID.TransactOpts, from, to, tokenId)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address from, address to, uint256 tokenId) returns()
func (_NodeID *NodeIDTransactorSession) TransferFrom(from common.Address, to common.Address, tokenId *big.Int) (*types.Transaction, error) {
	return _NodeID.Contract.TransferFrom(&_NodeID.TransactOpts, from, to, tokenId)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_NodeID *NodeIDTransactor) TransferOwnership(opts *bind.TransactOpts, newOwner common.Address) (*types.Transaction, error) {
	return _NodeID.contract.Transact(opts, "transferOwnership", newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_NodeID *NodeIDSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _NodeID.Contract.TransferOwnership(&_NodeID.TransactOpts, newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_NodeID *NodeIDTransactorSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _NodeID.Contract.TransferOwnership(&_NodeID.TransactOpts, newOwner)
}

// NodeIDApprovalIterator is returned from FilterApproval and is used to iterate over the raw logs and unpacked data for Approval events raised by the NodeID contract.
type NodeIDApprovalIterator struct {
	Event *NodeIDApproval // Event containing the contract specifics and raw log

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
func (it *NodeIDApprovalIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(NodeIDApproval)
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
		it.Event = new(NodeIDApproval)
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
func (it *NodeIDApprovalIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *NodeIDApprovalIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// NodeIDApproval represents a Approval event raised by the NodeID contract.
type NodeIDApproval struct {
	Owner    common.Address
	Approved common.Address
	TokenId  *big.Int
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterApproval is a free log retrieval operation binding the contract event 0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925.
//
// Solidity: event Approval(address indexed owner, address indexed approved, uint256 indexed tokenId)
func (_NodeID *NodeIDFilterer) FilterApproval(opts *bind.FilterOpts, owner []common.Address, approved []common.Address, tokenId []*big.Int) (*NodeIDApprovalIterator, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var approvedRule []interface{}
	for _, approvedItem := range approved {
		approvedRule = append(approvedRule, approvedItem)
	}
	var tokenIdRule []interface{}
	for _, tokenIdItem := range tokenId {
		tokenIdRule = append(tokenIdRule, tokenIdItem)
	}

	logs, sub, err := _NodeID.contract.FilterLogs(opts, "Approval", ownerRule, approvedRule, tokenIdRule)
	if err != nil {
		return nil, err
	}
	return &NodeIDApprovalIterator{contract: _NodeID.contract, event: "Approval", logs: logs, sub: sub}, nil
}

// WatchApproval is a free log subscription operation binding the contract event 0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925.
//
// Solidity: event Approval(address indexed owner, address indexed approved, uint256 indexed tokenId)
func (_NodeID *NodeIDFilterer) WatchApproval(opts *bind.WatchOpts, sink chan<- *NodeIDApproval, owner []common.Address, approved []common.Address, tokenId []*big.Int) (event.Subscription, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var approvedRule []interface{}
	for _, approvedItem := range approved {
		approvedRule = append(approvedRule, approvedItem)
	}
	var tokenIdRule []interface{}
	for _, tokenIdItem := range tokenId {
		tokenIdRule = append(tokenIdRule, tokenIdItem)
	}

	logs, sub, err := _NodeID.contract.WatchLogs(opts, "Approval", ownerRule, approvedRule, tokenIdRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(NodeIDApproval)
				if err := _NodeID.contract.UnpackLog(event, "Approval", log); err != nil {
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
// Solidity: event Approval(address indexed owner, address indexed approved, uint256 indexed tokenId)
func (_NodeID *NodeIDFilterer) ParseApproval(log types.Log) (*NodeIDApproval, error) {
	event := new(NodeIDApproval)
	if err := _NodeID.contract.UnpackLog(event, "Approval", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// NodeIDApprovalForAllIterator is returned from FilterApprovalForAll and is used to iterate over the raw logs and unpacked data for ApprovalForAll events raised by the NodeID contract.
type NodeIDApprovalForAllIterator struct {
	Event *NodeIDApprovalForAll // Event containing the contract specifics and raw log

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
func (it *NodeIDApprovalForAllIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(NodeIDApprovalForAll)
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
		it.Event = new(NodeIDApprovalForAll)
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
func (it *NodeIDApprovalForAllIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *NodeIDApprovalForAllIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// NodeIDApprovalForAll represents a ApprovalForAll event raised by the NodeID contract.
type NodeIDApprovalForAll struct {
	Owner    common.Address
	Operator common.Address
	Approved bool
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterApprovalForAll is a free log retrieval operation binding the contract event 0x17307eab39ab6107e8899845ad3d59bd9653f200f220920489ca2b5937696c31.
//
// Solidity: event ApprovalForAll(address indexed owner, address indexed operator, bool approved)
func (_NodeID *NodeIDFilterer) FilterApprovalForAll(opts *bind.FilterOpts, owner []common.Address, operator []common.Address) (*NodeIDApprovalForAllIterator, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var operatorRule []interface{}
	for _, operatorItem := range operator {
		operatorRule = append(operatorRule, operatorItem)
	}

	logs, sub, err := _NodeID.contract.FilterLogs(opts, "ApprovalForAll", ownerRule, operatorRule)
	if err != nil {
		return nil, err
	}
	return &NodeIDApprovalForAllIterator{contract: _NodeID.contract, event: "ApprovalForAll", logs: logs, sub: sub}, nil
}

// WatchApprovalForAll is a free log subscription operation binding the contract event 0x17307eab39ab6107e8899845ad3d59bd9653f200f220920489ca2b5937696c31.
//
// Solidity: event ApprovalForAll(address indexed owner, address indexed operator, bool approved)
func (_NodeID *NodeIDFilterer) WatchApprovalForAll(opts *bind.WatchOpts, sink chan<- *NodeIDApprovalForAll, owner []common.Address, operator []common.Address) (event.Subscription, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var operatorRule []interface{}
	for _, operatorItem := range operator {
		operatorRule = append(operatorRule, operatorItem)
	}

	logs, sub, err := _NodeID.contract.WatchLogs(opts, "ApprovalForAll", ownerRule, operatorRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(NodeIDApprovalForAll)
				if err := _NodeID.contract.UnpackLog(event, "ApprovalForAll", log); err != nil {
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

// ParseApprovalForAll is a log parse operation binding the contract event 0x17307eab39ab6107e8899845ad3d59bd9653f200f220920489ca2b5937696c31.
//
// Solidity: event ApprovalForAll(address indexed owner, address indexed operator, bool approved)
func (_NodeID *NodeIDFilterer) ParseApprovalForAll(log types.Log) (*NodeIDApprovalForAll, error) {
	event := new(NodeIDApprovalForAll)
	if err := _NodeID.contract.UnpackLog(event, "ApprovalForAll", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// NodeIDOwnershipTransferredIterator is returned from FilterOwnershipTransferred and is used to iterate over the raw logs and unpacked data for OwnershipTransferred events raised by the NodeID contract.
type NodeIDOwnershipTransferredIterator struct {
	Event *NodeIDOwnershipTransferred // Event containing the contract specifics and raw log

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
func (it *NodeIDOwnershipTransferredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(NodeIDOwnershipTransferred)
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
		it.Event = new(NodeIDOwnershipTransferred)
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
func (it *NodeIDOwnershipTransferredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *NodeIDOwnershipTransferredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// NodeIDOwnershipTransferred represents a OwnershipTransferred event raised by the NodeID contract.
type NodeIDOwnershipTransferred struct {
	OldOwner common.Address
	NewOwner common.Address
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterOwnershipTransferred is a free log retrieval operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed oldOwner, address indexed newOwner)
func (_NodeID *NodeIDFilterer) FilterOwnershipTransferred(opts *bind.FilterOpts, oldOwner []common.Address, newOwner []common.Address) (*NodeIDOwnershipTransferredIterator, error) {

	var oldOwnerRule []interface{}
	for _, oldOwnerItem := range oldOwner {
		oldOwnerRule = append(oldOwnerRule, oldOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _NodeID.contract.FilterLogs(opts, "OwnershipTransferred", oldOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return &NodeIDOwnershipTransferredIterator{contract: _NodeID.contract, event: "OwnershipTransferred", logs: logs, sub: sub}, nil
}

// WatchOwnershipTransferred is a free log subscription operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed oldOwner, address indexed newOwner)
func (_NodeID *NodeIDFilterer) WatchOwnershipTransferred(opts *bind.WatchOpts, sink chan<- *NodeIDOwnershipTransferred, oldOwner []common.Address, newOwner []common.Address) (event.Subscription, error) {

	var oldOwnerRule []interface{}
	for _, oldOwnerItem := range oldOwner {
		oldOwnerRule = append(oldOwnerRule, oldOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _NodeID.contract.WatchLogs(opts, "OwnershipTransferred", oldOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(NodeIDOwnershipTransferred)
				if err := _NodeID.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
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
// Solidity: event OwnershipTransferred(address indexed oldOwner, address indexed newOwner)
func (_NodeID *NodeIDFilterer) ParseOwnershipTransferred(log types.Log) (*NodeIDOwnershipTransferred, error) {
	event := new(NodeIDOwnershipTransferred)
	if err := _NodeID.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// NodeIDRegistryUpdatedIterator is returned from FilterRegistryUpdated and is used to iterate over the raw logs and unpacked data for RegistryUpdated events raised by the NodeID contract.
type NodeIDRegistryUpdatedIterator struct {
	Event *NodeIDRegistryUpdated // Event containing the contract specifics and raw log

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
func (it *NodeIDRegistryUpdatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(NodeIDRegistryUpdated)
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
		it.Event = new(NodeIDRegistryUpdated)
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
func (it *NodeIDRegistryUpdatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *NodeIDRegistryUpdatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// NodeIDRegistryUpdated represents a RegistryUpdated event raised by the NodeID contract.
type NodeIDRegistryUpdated struct {
	OldRegistry common.Address
	NewRegistry common.Address
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterRegistryUpdated is a free log retrieval operation binding the contract event 0x482b97c53e48ffa324a976e2738053e9aff6eee04d8aac63b10e19411d869b82.
//
// Solidity: event RegistryUpdated(address indexed oldRegistry, address indexed newRegistry)
func (_NodeID *NodeIDFilterer) FilterRegistryUpdated(opts *bind.FilterOpts, oldRegistry []common.Address, newRegistry []common.Address) (*NodeIDRegistryUpdatedIterator, error) {

	var oldRegistryRule []interface{}
	for _, oldRegistryItem := range oldRegistry {
		oldRegistryRule = append(oldRegistryRule, oldRegistryItem)
	}
	var newRegistryRule []interface{}
	for _, newRegistryItem := range newRegistry {
		newRegistryRule = append(newRegistryRule, newRegistryItem)
	}

	logs, sub, err := _NodeID.contract.FilterLogs(opts, "RegistryUpdated", oldRegistryRule, newRegistryRule)
	if err != nil {
		return nil, err
	}
	return &NodeIDRegistryUpdatedIterator{contract: _NodeID.contract, event: "RegistryUpdated", logs: logs, sub: sub}, nil
}

// WatchRegistryUpdated is a free log subscription operation binding the contract event 0x482b97c53e48ffa324a976e2738053e9aff6eee04d8aac63b10e19411d869b82.
//
// Solidity: event RegistryUpdated(address indexed oldRegistry, address indexed newRegistry)
func (_NodeID *NodeIDFilterer) WatchRegistryUpdated(opts *bind.WatchOpts, sink chan<- *NodeIDRegistryUpdated, oldRegistry []common.Address, newRegistry []common.Address) (event.Subscription, error) {

	var oldRegistryRule []interface{}
	for _, oldRegistryItem := range oldRegistry {
		oldRegistryRule = append(oldRegistryRule, oldRegistryItem)
	}
	var newRegistryRule []interface{}
	for _, newRegistryItem := range newRegistry {
		newRegistryRule = append(newRegistryRule, newRegistryItem)
	}

	logs, sub, err := _NodeID.contract.WatchLogs(opts, "RegistryUpdated", oldRegistryRule, newRegistryRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(NodeIDRegistryUpdated)
				if err := _NodeID.contract.UnpackLog(event, "RegistryUpdated", log); err != nil {
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

// ParseRegistryUpdated is a log parse operation binding the contract event 0x482b97c53e48ffa324a976e2738053e9aff6eee04d8aac63b10e19411d869b82.
//
// Solidity: event RegistryUpdated(address indexed oldRegistry, address indexed newRegistry)
func (_NodeID *NodeIDFilterer) ParseRegistryUpdated(log types.Log) (*NodeIDRegistryUpdated, error) {
	event := new(NodeIDRegistryUpdated)
	if err := _NodeID.contract.UnpackLog(event, "RegistryUpdated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// NodeIDSlotsMintedIterator is returned from FilterSlotsMinted and is used to iterate over the raw logs and unpacked data for SlotsMinted events raised by the NodeID contract.
type NodeIDSlotsMintedIterator struct {
	Event *NodeIDSlotsMinted // Event containing the contract specifics and raw log

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
func (it *NodeIDSlotsMintedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(NodeIDSlotsMinted)
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
		it.Event = new(NodeIDSlotsMinted)
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
func (it *NodeIDSlotsMintedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *NodeIDSlotsMintedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// NodeIDSlotsMinted represents a SlotsMinted event raised by the NodeID contract.
type NodeIDSlotsMinted struct {
	By                  common.Address
	FirstTokenId        uint32
	LastTokenId         uint32
	MinActivationAmount *big.Int
	VestingPeriod       uint64
	Raw                 types.Log // Blockchain specific contextual infos
}

// FilterSlotsMinted is a free log retrieval operation binding the contract event 0x9902251bf2894876d6d1dc26c5e2005e75018334706538cb7bf283598aefc42b.
//
// Solidity: event SlotsMinted(address indexed by, uint32 indexed firstTokenId, uint32 indexed lastTokenId, uint256 minActivationAmount, uint64 vestingPeriod)
func (_NodeID *NodeIDFilterer) FilterSlotsMinted(opts *bind.FilterOpts, by []common.Address, firstTokenId []uint32, lastTokenId []uint32) (*NodeIDSlotsMintedIterator, error) {

	var byRule []interface{}
	for _, byItem := range by {
		byRule = append(byRule, byItem)
	}
	var firstTokenIdRule []interface{}
	for _, firstTokenIdItem := range firstTokenId {
		firstTokenIdRule = append(firstTokenIdRule, firstTokenIdItem)
	}
	var lastTokenIdRule []interface{}
	for _, lastTokenIdItem := range lastTokenId {
		lastTokenIdRule = append(lastTokenIdRule, lastTokenIdItem)
	}

	logs, sub, err := _NodeID.contract.FilterLogs(opts, "SlotsMinted", byRule, firstTokenIdRule, lastTokenIdRule)
	if err != nil {
		return nil, err
	}
	return &NodeIDSlotsMintedIterator{contract: _NodeID.contract, event: "SlotsMinted", logs: logs, sub: sub}, nil
}

// WatchSlotsMinted is a free log subscription operation binding the contract event 0x9902251bf2894876d6d1dc26c5e2005e75018334706538cb7bf283598aefc42b.
//
// Solidity: event SlotsMinted(address indexed by, uint32 indexed firstTokenId, uint32 indexed lastTokenId, uint256 minActivationAmount, uint64 vestingPeriod)
func (_NodeID *NodeIDFilterer) WatchSlotsMinted(opts *bind.WatchOpts, sink chan<- *NodeIDSlotsMinted, by []common.Address, firstTokenId []uint32, lastTokenId []uint32) (event.Subscription, error) {

	var byRule []interface{}
	for _, byItem := range by {
		byRule = append(byRule, byItem)
	}
	var firstTokenIdRule []interface{}
	for _, firstTokenIdItem := range firstTokenId {
		firstTokenIdRule = append(firstTokenIdRule, firstTokenIdItem)
	}
	var lastTokenIdRule []interface{}
	for _, lastTokenIdItem := range lastTokenId {
		lastTokenIdRule = append(lastTokenIdRule, lastTokenIdItem)
	}

	logs, sub, err := _NodeID.contract.WatchLogs(opts, "SlotsMinted", byRule, firstTokenIdRule, lastTokenIdRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(NodeIDSlotsMinted)
				if err := _NodeID.contract.UnpackLog(event, "SlotsMinted", log); err != nil {
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

// ParseSlotsMinted is a log parse operation binding the contract event 0x9902251bf2894876d6d1dc26c5e2005e75018334706538cb7bf283598aefc42b.
//
// Solidity: event SlotsMinted(address indexed by, uint32 indexed firstTokenId, uint32 indexed lastTokenId, uint256 minActivationAmount, uint64 vestingPeriod)
func (_NodeID *NodeIDFilterer) ParseSlotsMinted(log types.Log) (*NodeIDSlotsMinted, error) {
	event := new(NodeIDSlotsMinted)
	if err := _NodeID.contract.UnpackLog(event, "SlotsMinted", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// NodeIDTransferIterator is returned from FilterTransfer and is used to iterate over the raw logs and unpacked data for Transfer events raised by the NodeID contract.
type NodeIDTransferIterator struct {
	Event *NodeIDTransfer // Event containing the contract specifics and raw log

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
func (it *NodeIDTransferIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(NodeIDTransfer)
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
		it.Event = new(NodeIDTransfer)
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
func (it *NodeIDTransferIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *NodeIDTransferIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// NodeIDTransfer represents a Transfer event raised by the NodeID contract.
type NodeIDTransfer struct {
	From    common.Address
	To      common.Address
	TokenId *big.Int
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterTransfer is a free log retrieval operation binding the contract event 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef.
//
// Solidity: event Transfer(address indexed from, address indexed to, uint256 indexed tokenId)
func (_NodeID *NodeIDFilterer) FilterTransfer(opts *bind.FilterOpts, from []common.Address, to []common.Address, tokenId []*big.Int) (*NodeIDTransferIterator, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}
	var tokenIdRule []interface{}
	for _, tokenIdItem := range tokenId {
		tokenIdRule = append(tokenIdRule, tokenIdItem)
	}

	logs, sub, err := _NodeID.contract.FilterLogs(opts, "Transfer", fromRule, toRule, tokenIdRule)
	if err != nil {
		return nil, err
	}
	return &NodeIDTransferIterator{contract: _NodeID.contract, event: "Transfer", logs: logs, sub: sub}, nil
}

// WatchTransfer is a free log subscription operation binding the contract event 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef.
//
// Solidity: event Transfer(address indexed from, address indexed to, uint256 indexed tokenId)
func (_NodeID *NodeIDFilterer) WatchTransfer(opts *bind.WatchOpts, sink chan<- *NodeIDTransfer, from []common.Address, to []common.Address, tokenId []*big.Int) (event.Subscription, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}
	var tokenIdRule []interface{}
	for _, tokenIdItem := range tokenId {
		tokenIdRule = append(tokenIdRule, tokenIdItem)
	}

	logs, sub, err := _NodeID.contract.WatchLogs(opts, "Transfer", fromRule, toRule, tokenIdRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(NodeIDTransfer)
				if err := _NodeID.contract.UnpackLog(event, "Transfer", log); err != nil {
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
// Solidity: event Transfer(address indexed from, address indexed to, uint256 indexed tokenId)
func (_NodeID *NodeIDFilterer) ParseTransfer(log types.Log) (*NodeIDTransfer, error) {
	event := new(NodeIDTransfer)
	if err := _NodeID.contract.UnpackLog(event, "Transfer", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

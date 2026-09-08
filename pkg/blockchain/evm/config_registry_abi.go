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

// ConfigRegistryMetaData contains all meta data concerning the ConfigRegistry contract.
var ConfigRegistryMetaData = &bind.MetaData{
	ABI: "[{\"type\":\"constructor\",\"inputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"computeIssuerId\",\"inputs\":[{\"name\":\"issuerKeys_\",\"type\":\"address[]\",\"internalType\":\"address[]\"},{\"name\":\"threshold_\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"eip712Domain\",\"inputs\":[],\"outputs\":[{\"name\":\"fields\",\"type\":\"bytes1\",\"internalType\":\"bytes1\"},{\"name\":\"name\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"version\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"chainId\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"verifyingContract\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"salt\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"extensions\",\"type\":\"uint256[]\",\"internalType\":\"uint256[]\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"isRegistered\",\"inputs\":[{\"name\":\"issuerId\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"issuerKeys\",\"inputs\":[{\"name\":\"issuerId\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[{\"name\":\"\",\"type\":\"address[]\",\"internalType\":\"address[]\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"issuerSettings\",\"inputs\":[{\"name\":\"issuerId\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[{\"name\":\"issuerKeys_\",\"type\":\"address[]\",\"internalType\":\"address[]\"},{\"name\":\"threshold_\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"nonce_\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"nonce\",\"inputs\":[{\"name\":\"issuerId\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"registerIssuer\",\"inputs\":[{\"name\":\"issuerKeys_\",\"type\":\"address[]\",\"internalType\":\"address[]\"},{\"name\":\"threshold_\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"signatures\",\"type\":\"bytes[]\",\"internalType\":\"bytes[]\"}],\"outputs\":[{\"name\":\"issuerId\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"setConfig\",\"inputs\":[{\"name\":\"issuerId\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"key\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"checksum\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"expectedNonce\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"signatures\",\"type\":\"bytes[]\",\"internalType\":\"bytes[]\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"setConfigWithData\",\"inputs\":[{\"name\":\"issuerId\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"key\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"data\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"expectedNonce\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"signatures\",\"type\":\"bytes[]\",\"internalType\":\"bytes[]\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"threshold\",\"inputs\":[{\"name\":\"issuerId\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"updateIssuerSettings\",\"inputs\":[{\"name\":\"issuerId\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"newIssuerKeys\",\"type\":\"address[]\",\"internalType\":\"address[]\"},{\"name\":\"newThreshold\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"expectedNonce\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"signatures\",\"type\":\"bytes[]\",\"internalType\":\"bytes[]\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"event\",\"name\":\"ConfigCommitted\",\"inputs\":[{\"name\":\"issuerId\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"key\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"},{\"name\":\"checksum\",\"type\":\"bytes32\",\"indexed\":false,\"internalType\":\"bytes32\"},{\"name\":\"newNonce\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"ConfigWithDataCommitted\",\"inputs\":[{\"name\":\"issuerId\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"key\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"},{\"name\":\"checksum\",\"type\":\"bytes32\",\"indexed\":false,\"internalType\":\"bytes32\"},{\"name\":\"data\",\"type\":\"bytes\",\"indexed\":false,\"internalType\":\"bytes\"},{\"name\":\"newNonce\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"EIP712DomainChanged\",\"inputs\":[],\"anonymous\":false},{\"type\":\"event\",\"name\":\"IssuerRegistered\",\"inputs\":[{\"name\":\"issuerId\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"issuerKeys\",\"type\":\"address[]\",\"indexed\":false,\"internalType\":\"address[]\"},{\"name\":\"threshold\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"IssuerSettingsUpdated\",\"inputs\":[{\"name\":\"issuerId\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"newIssuerKeys\",\"type\":\"address[]\",\"indexed\":false,\"internalType\":\"address[]\"},{\"name\":\"newThreshold\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"},{\"name\":\"newNonce\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"error\",\"name\":\"BelowThreshold\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"ECDSAInvalidSignature\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"ECDSAInvalidSignatureLength\",\"inputs\":[{\"name\":\"length\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]},{\"type\":\"error\",\"name\":\"ECDSAInvalidSignatureS\",\"inputs\":[{\"name\":\"s\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}]},{\"type\":\"error\",\"name\":\"InvalidShortString\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"InvalidThreshold\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"IssuerAlreadyRegistered\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"IssuerKeysNotSorted\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"IssuerNotRegistered\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"NotAnIssuerKey\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"NotEnoughIssuerKeys\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"SignaturesNotOrdered\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"StringTooLong\",\"inputs\":[{\"name\":\"str\",\"type\":\"string\",\"internalType\":\"string\"}]},{\"type\":\"error\",\"name\":\"UnexpectedNonce\",\"inputs\":[{\"name\":\"current\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"supplied\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]},{\"type\":\"error\",\"name\":\"ZeroIssuerKey\",\"inputs\":[]}]",
	Bin: "0x6101606040523461013d57604051610018604082610141565b6014815260208101907f59656c6c6f77436f6e6669675265676973747279000000000000000000000000825260405191610053604084610141565b600183526020830191603160f81b835261006c81610178565b610120526100798461031a565b61014052519020918260e05251902080610100524660a0526040519060208201927f8b73c3c69bb8fe3d512ecc4cf759cc79239f7b179b0ffacaa9a75d522b39400f8452604083015260608201524660808201523060a082015260a081526100e260c082610141565b5190206080523060c052604051611be0908161045f823960805181611242015260a051816112ff015260c0518161120c015260e05181611291015261010051816112b7015261012051816104240152610140518161044e0152f35b5f80fd5b601f909101601f19168101906001600160401b0382119082101761016457604052565b634e487b7160e01b5f52604160045260245ffd5b908151602081105f146101f2575090601f8151116101b25760208151910151602082106101a3571790565b5f198260200360031b1b161790565b604460209160405192839163305a27a960e01b83528160048401528051918291826024860152018484015e5f828201840152601f01601f19168101030190fd5b6001600160401b038111610164575f54600181811c91168015610310575b60208210146102fc57601f81116102be575b50602092601f821160011461025f57928192935f92610254575b50508160011b915f199060031b1c1916175f5560ff90565b015190505f8061023c565b601f198216935f8052805f20915f5b8681106102a6575083600195961061028e575b505050811b015f5560ff90565b01515f1960f88460031b161c191690555f8080610281565b9192602060018192868501518155019401920161026e565b81811115610222575f805260205f20601f80840160051c809201920160051c03905f5b8281106102ef575050610222565b5f828201556001016102e1565b634e487b7160e01b5f52602260045260245ffd5b90607f1690610210565b908151602081105f14610345575090601f8151116101b25760208151910151602082106101a3571790565b6001600160401b03811161016457600154600181811c91168015610454575b60208210146102fc57601f8111610415575b50602092601f82116001146103b457928192935f926103a9575b50508160011b915f199060031b1c19161760015560ff90565b015190505f80610390565b601f1982169360015f52805f20915f5b8681106103fd57508360019596106103e5575b505050811b0160015560ff90565b01515f1960f88460031b161c191690555f80806103d7565b919260206001819286850151815501940192016103c4565b818111156103765760015f5260205f20601f80840160051c809201920160051c03905f5b828110610447575050610376565b5f82820155600101610439565b90607f169061036456fe60806040526004361015610011575f80fd5b5f5f3560e01c80631b7ae749146109d457806336ab7a61146107535780634a2c55211461059557806370ae92d214610559578063787779e81461050657806384b0196e1461040a578063c3c5a547146103c2578063c86ec2bf14610385578063cf92536114610334578063d9617437146100fe5763e6639d4714610093575f80fd5b346100fb5760203660031901126100fb576100ec906040906001600160a01b036100bb610c17565b16815260026020522060018101546100d7600283015492610dca565b91604051938493606085526060850190610c71565b91602084015260408301520390f35b80fd5b50346100fb5760a03660031901126100fb57610118610c17565b602435906044356001600160401b03811161033057366023820112156103305780600401356001600160401b03811161032c576024820191602482369201011161032c57606435926084356001600160401b0381116103285761017f903690600401610c41565b6001600160a01b0383165f9081526002602052604090206001015490959192901561031957600160a01b60019003811695868952600260205260408920600201938454938481818114916101d292610cd1565b6101dd36888a610ed4565b80519060200120906040519060208201927f0e0319a4710d6d4ab6a935e9c2d6c959f9efb30405ea281469d229f6311e90cd84528b60408401528c6060840152608083015260a082015260a0815261023660c082610cef565b51902061024290611064565b61024b9361108a565b6001018155833b1561030a5785604051634f86772560e11b81528660048201526040602482015281818061028360448201888a610f19565b0381838a5af1801561030e576102f5575b50507f4c57f3906e38359a4c973d7c01f642d34582e5d00263cb0452206692462b9903926102e96102c6368584610ed4565b602081519101209254916040519485948552606060208601526060850191610f19565b9060408301520390a380f35b816102ff91610cef565b61030a57855f610294565b8580fd5b6040513d84823e3d90fd5b63ec3b3a3760e01b8852600488fd5b8680fd5b8480fd5b8380fd5b50346100fb5760403660031901126100fb57600435906001600160401b0382116100fb57602061037361036a3660048601610c41565b60243591610e1d565b6040516001600160a01b039091168152f35b50346100fb5760203660031901126100fb576020906001906040906001600160a01b036103b0610c17565b16815260028452200154604051908152f35b50346100fb5760203660031901126100fb5760206104006103e1610c17565b6001600160a01b03165f90815260026020526040902060010154151590565b6040519015158152f35b50346100fb57806003193601126100fb576104aa906104487f0000000000000000000000000000000000000000000000000000000000000000611325565b906104727f000000000000000000000000000000000000000000000000000000000000000061144b565b9060206104b8604051936104868386610cef565b8385525f368137604051968796600f60f81b885260e08589015260e0880190610cad565b908682036040880152610cad565b904660608601523060808601528260a086015284820360c08601528080855193848152019401925b8281106104ef57505050500390f35b8351855286955093810193928101926001016104e0565b50346100fb5760203660031901126100fb5761055590610541906040906001600160a01b03610533610c17565b168152600260205220610dca565b604051918291602083526020830190610c71565b0390f35b50346100fb5760203660031901126100fb576020906002906040906001600160a01b03610584610c17565b168152828452200154604051908152f35b50346107405760a0366003190112610740576105af610c17565b60243590604435606435916084356001600160401b038111610740576105d9903690600401610c41565b6001600160a01b0383165f9081526002602052604090206001015491949091156107445761068e600192838060a01b03851696875f526002602052600260405f20019561068887549561062f8188808214610cd1565b60405160208101917fe0a34c2e8812f080d2ad116e9c1393a5cc631b7c23a4cdb49d693e34ba58d86083528c60408301528d60608301528b608083015260a082015260a0815261068060c082610cef565b519020611064565b9061108a565b018155823b156107405760405163d1fd27b360e01b81528460048201528260248201525f8160448183885af18015610735576106fa575b507fa727e52bcb62c238dd87fa146891db1d7e2ed6a4394fd61d5b299f63d633f61a916040915482519182526020820152a380f35b60409195509161072b5f7fa727e52bcb62c238dd87fa146891db1d7e2ed6a4394fd61d5b299f63d633f61a94610cef565b5f959150916106c5565b6040513d5f823e3d90fd5b5f80fd5b63ec3b3a3760e01b5f5260045ffd5b34610740576060366003190112610740576004356001600160401b03811161074057610783903690600401610c41565b90602435906044356001600160401b038111610740576107a7903690600401610c41565b906107bc846107b7368887610d10565b610f61565b6107c7848685610e1d565b6001600160a01b0381165f908152600260205260409020600101549092906109c55761087a9185916108686107fd368a89610d10565b6040516108208161081260208201809561102e565b03601f198101835282610cef565b51902060405160208101917f9b484a647fe5d836fce1a2659dfbb45e6f65b90f1347e06a90b49f302bb12b318352604082015285606082015260608152610680608082610cef565b91610874368a89610d10565b926110b1565b60405160208101906108928161081287898887610dad565b519020604051610594808201908282106001600160401b038311176109b157602091839161164c83393081520301905ff515610735576001600160a01b03165f8181526002602052604090209092906001600160401b0385116109b157600160401b85116109b1578054858255808610610987575b505f8181526020812090845b87821061096257602087807fd4a9f2e1f27ddd97144ea11c3582e0ab1fd269dedf40a26c12b5f8ff70c781218b8a6109578b8060018d015560405193849384610dad565b0390a2604051908152f35b8035916001600160a01b03831683036107405760206001920192818501550190610913565b815f52858060205f20019103905f5b8281106109a4575050610907565b5f82820155600101610996565b634e487b7160e01b5f52604160045260245ffd5b633c2dd12960e01b5f5260045ffd5b346107405760a0366003190112610740576109ed610c17565b6024356001600160401b03811161074057610a0c903690600401610c41565b6064359291906044356084356001600160401b03811161074057610a34903690600401610c41565b6001600160a01b0386165f908152600260205260409020600101549096901561074457600160a01b60019003861695865f52600260205260405f20976002890193845493848181811491610a8792610cd1565b86610a93368a8c610d10565b90610a9d91610f61565b610aa836898b610d10565b60405180602081019283610abb9161102e565b03601f1981018252610acd9082610cef565b519020906040519060208201927fca7e77e070f83544e4bcfcd460256c58af0b3448feb4216b11b6a37a866d918f84528c6040840152606083015288608083015260a082015260a08152610b2260c082610cef565b519020610b2e90611064565b610b379361108a565b60010181556001600160401b0383116109b157600160401b83116109b1578554838755808410610bed575b505f8681526020812090855b858210610bc8575050508160017ff3f30d031c773609e9b6609fb5782eaf3e1c63707cb41d667e45674eaf0819b79697015554610bb8604051948594606086526060860191610d6e565b91602084015260408301520390a2005b8035916001600160a01b03831683036107405760206001920192818501550190610b6e565b865f52838060205f20019103905f5b828110610c0a575050610b62565b5f82820155600101610bfc565b600435906001600160a01b038216820361074057565b35906001600160a01b038216820361074057565b9181601f84011215610740578235916001600160401b038311610740576020808501948460051b01011161074057565b90602080835192838152019201905f5b818110610c8e5750505090565b82516001600160a01b0316845260209384019390920191600101610c81565b805180835260209291819084018484015e5f828201840152601f01601f1916010190565b15610cda575050565b63018af85d60e51b5f5260045260245260445ffd5b90601f801991011681019081106001600160401b038211176109b157604052565b929190926001600160401b0384116109b1578360051b906020808301610d396040519182610cef565b809681520191810192831161074057905b828210610d5657505050565b60208091610d6384610c2d565b815201910190610d4a565b916020908281520191905f5b818110610d875750505090565b909192602080600192838060a01b03610d9f88610c2d565b168152019401929101610d7a565b939291602091610dc591604087526040870191610d6e565b930152565b90604051918281549182825260208201905f5260205f20925f5b818110610dfb575050610df992500383610cef565b565b84546001600160a01b0316835260019485019487945060209093019201610de4565b9091610e39600b93610812604051938492602084019687610dad565b519020604051610594610e4f6020820183610cef565b808252602082019061164c8239610eaa604051916020808401308152818552610e79604086610cef565b60405194859383850197518091895e840190838201905f8252519283915e01015f815203601f198101835282610cef565b5190209060405191604083015260208201523081520160ff8153605590206001600160a01b031690565b9291926001600160401b0382116109b15760405191610efd601f8201601f191660200184610cef565b829481845281830111610740578281602093845f960137010152565b908060209392818452848401375f828201840152601f01601f1916010190565b8051821015610f4d5760209160051b010190565b634e487b7160e01b5f52603260045260245ffd5b90801561101f57815110611010575f5b815181101561100c576001600160a01b03610f8c8284610f39565b511615610ffd5780610fa1575b600101610f71565b6001600160a01b03610fb38284610f39565b51165f198201828111610fe9576001600160a01b0390610fd39085610f39565b511610610f995763f4ce592960e01b5f5260045ffd5b634e487b7160e01b5f52601160045260245ffd5b634e46e4d360e01b5f5260045ffd5b5050565b63036004c160e61b5f5260045ffd5b63aabd5a0960e01b5f5260045ffd5b80516020909101905f5b8181106110455750505090565b82516001600160a01b0316845260209384019390920191600101611038565b60429061106f611209565b906040519161190160f01b8352600283015260228201522090565b92610df99360018060a01b03165f52600260205260405f2092610874600185015494610dca565b949390949291928184106111fb57909493925f955f925f965f965b848810156111ea578760051b820135601e19833603018112156107405782018035906001600160401b0382116107405760200181360381136107405761111a61112091611129933691610ed4565b8561151b565b90929192611555565b6001600160a01b038181169a168a11156111db5793955b87518110806111bf575b1561115757600101611140565b909294979998959193958751821090816111a2575b501561119357600101975f198114610fe957600180910199019693919794979290926110cc565b6362aeee5760e01b5f5260045ffd5b90506001600160a01b036111b6838a610f39565b5116145f61116c565b50896001600160a01b036111d3838b610f39565b51161061114a565b6303941dd360e21b5f5260045ffd5b98965050509450505050106111fb57565b625713a160e91b5f5260045ffd5b307f00000000000000000000000000000000000000000000000000000000000000006001600160a01b031614806112fc575b15611264577f000000000000000000000000000000000000000000000000000000000000000090565b60405160208101907f8b73c3c69bb8fe3d512ecc4cf759cc79239f7b179b0ffacaa9a75d522b39400f82527f000000000000000000000000000000000000000000000000000000000000000060408201527f000000000000000000000000000000000000000000000000000000000000000060608201524660808201523060a082015260a081526112f660c082610cef565b51902090565b507f0000000000000000000000000000000000000000000000000000000000000000461461123b565b60ff811461136b5760ff811690601f821161135c5760405191611349604084610cef565b6020808452838101919036833783525290565b632cd44ac360e21b5f5260045ffd5b506040515f5f548060011c9160018216918215611441575b60208410831461142d57838552849290811561140e57506001146113b1575b6113ae92500382610cef565b90565b505f80805290917f290decd9548b62a8d60345a988386fc84ba6bc95484008f6362f93160ef3e5635b8183106113f25750509060206113ae928201016113a2565b60209193508060019154838588010152019101909183926113da565b602092506113ae94915060ff191682840152151560051b8201016113a2565b634e487b7160e01b5f52602260045260245ffd5b92607f1692611383565b60ff811461146f5760ff811690601f821161135c5760405191611349604084610cef565b506040515f6001548060011c9160018216918215611511575b60208410831461142d57838552849290811561140e57506001146114b2576113ae92500382610cef565b5060015f90815290917fb10e2d527612073b26eecdfd717e6a320cf44b4afac2b0732d9fcbe2b7fa0cf65b8183106114f55750509060206113ae928201016113a2565b60209193508060019154838588010152019101909183926114dd565b92607f1692611488565b815191906041830361154b576115449250602082015190606060408401519301515f1a906115c9565b9192909190565b50505f9160029190565b60048110156115b55780611567575050565b6001810361157e5763f645eedf60e01b5f5260045ffd5b60028103611599575063fce698f760e01b5f5260045260245ffd5b6003146115a35750565b6335e2f38360e21b5f5260045260245ffd5b634e487b7160e01b5f52602160045260245ffd5b91907f7fffffffffffffffffffffffffffffff5d576e7357a4501ddfe92f46681b20a08411611640579160209360809260ff5f9560405194855216868401526040830152606082015282805260015afa15610735575f516001600160a01b0381161561163657905f905f90565b505f906001905f90565b5050505f916003919056fe60a03461008d57601f61059438819003918201601f19168301916001600160401b038311848410176100915780849260209460405283398101031261008d57516001600160a01b03811680820361008d571561007e576080526040516104ee90816100a682396080518181816101280152818161023801526103130152f35b6349e27cff60e01b5f5260045ffd5b5f80fd5b634e487b7160e01b5f52604160045260245ffdfe60806040526004361015610011575f80fd5b5f3560e01c80633cb37e57146103c557806346c736b5146103425780638da5cb5b146102fe5780639f0cee4a146101df578063af890358146101ac578063d1fd27b31461010f5763fec5bedb14610066575f80fd5b3461010b57602036600319011261010b576004355f525f60205260405f20604051806020835491828152019081935f5260205f20905f5b8181106100f557505050816100b3910382610452565b604051918291602083019060208452518091526040830191905f5b8181106100dc575050500390f35b82518452859450602093840193909201916001016100ce565b825484526020909301926001928301920161009d565b5f80fd5b3461010b57604036600319011261010b576024356004357f00000000000000000000000000000000000000000000000000000000000000006001600160a01b0316330361019d5767ffffffffffffffff6101698383610488565b6040519384521660208301527f952e9f054f8f14436d21495120b5658808398d89e8240fca0e0b5669e4dbb01360403393a3005b637138837360e11b5f5260045ffd5b3461010b57602036600319011261010b576004355f525f602052602067ffffffffffffffff60405f205416604051908152f35b3461010b57604036600319011261010b5760243560043567ffffffffffffffff821161010b573660238301121561010b5781600401359167ffffffffffffffff831161010b576024810190602484369201011161010b577f00000000000000000000000000000000000000000000000000000000000000006001600160a01b0316330361019d577f138bbba807de4352fcbdebae920532cde6774dba6fd049b88ea55bc3af412c00905f6080601f19601f87011695806040516102a560208a0182610452565b81815260208101908287833785602084830101525190209467ffffffffffffffff6102d0878a610488565b60405197885216602087015260606040870152816060870152838601378301015260808133958101030190a3005b3461010b575f36600319011261010b576040517f00000000000000000000000000000000000000000000000000000000000000006001600160a01b03168152602090f35b3461010b57604036600319011261010b576024356004355f525f60205260405f20811515806103ba575b156103ab575f1982019182116103975760209161038891610429565b90549060031b1c604051908152f35b634e487b7160e01b5f52601160045260245ffd5b6316f4c85360e01b5f5260045ffd5b50805482111561036c565b3461010b57602036600319011261010b57600435805f525f60205260405f20549081155f146103fc57505060205f5b604051908152f35b5f525f60205260405f205f1982019182116103975760209161041d91610429565b90549060031b1c6103f4565b805482101561043e575f5260205f2001905f90565b634e487b7160e01b5f52603260045260245ffd5b90601f8019910116810190811067ffffffffffffffff82111761047457604052565b634e487b7160e01b5f52604160045260245ffd5b80156104df575f525f60205260405f2080549168010000000000000000831015610474576104c583600167ffffffffffffffff9501845583610429565b819291549060031b91821b915f19901b1916179055541690565b6355a9397560e11b5f5260045ffd",
}

// ConfigRegistryABI is the input ABI used to generate the binding from.
// Deprecated: Use ConfigRegistryMetaData.ABI instead.
var ConfigRegistryABI = ConfigRegistryMetaData.ABI

// ConfigRegistryBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use ConfigRegistryMetaData.Bin instead.
var ConfigRegistryBin = ConfigRegistryMetaData.Bin

// DeployConfigRegistry deploys a new Ethereum contract, binding an instance of ConfigRegistry to it.
func DeployConfigRegistry(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *ConfigRegistry, error) {
	parsed, err := ConfigRegistryMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(ConfigRegistryBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &ConfigRegistry{ConfigRegistryCaller: ConfigRegistryCaller{contract: contract}, ConfigRegistryTransactor: ConfigRegistryTransactor{contract: contract}, ConfigRegistryFilterer: ConfigRegistryFilterer{contract: contract}}, nil
}

// ConfigRegistry is an auto generated Go binding around an Ethereum contract.
type ConfigRegistry struct {
	ConfigRegistryCaller     // Read-only binding to the contract
	ConfigRegistryTransactor // Write-only binding to the contract
	ConfigRegistryFilterer   // Log filterer for contract events
}

// ConfigRegistryCaller is an auto generated read-only Go binding around an Ethereum contract.
type ConfigRegistryCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ConfigRegistryTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ConfigRegistryTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ConfigRegistryFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type ConfigRegistryFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ConfigRegistrySession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ConfigRegistrySession struct {
	Contract     *ConfigRegistry   // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// ConfigRegistryCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ConfigRegistryCallerSession struct {
	Contract *ConfigRegistryCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts         // Call options to use throughout this session
}

// ConfigRegistryTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ConfigRegistryTransactorSession struct {
	Contract     *ConfigRegistryTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts         // Transaction auth options to use throughout this session
}

// ConfigRegistryRaw is an auto generated low-level Go binding around an Ethereum contract.
type ConfigRegistryRaw struct {
	Contract *ConfigRegistry // Generic contract binding to access the raw methods on
}

// ConfigRegistryCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ConfigRegistryCallerRaw struct {
	Contract *ConfigRegistryCaller // Generic read-only contract binding to access the raw methods on
}

// ConfigRegistryTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ConfigRegistryTransactorRaw struct {
	Contract *ConfigRegistryTransactor // Generic write-only contract binding to access the raw methods on
}

// NewConfigRegistry creates a new instance of ConfigRegistry, bound to a specific deployed contract.
func NewConfigRegistry(address common.Address, backend bind.ContractBackend) (*ConfigRegistry, error) {
	contract, err := bindConfigRegistry(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &ConfigRegistry{ConfigRegistryCaller: ConfigRegistryCaller{contract: contract}, ConfigRegistryTransactor: ConfigRegistryTransactor{contract: contract}, ConfigRegistryFilterer: ConfigRegistryFilterer{contract: contract}}, nil
}

// NewConfigRegistryCaller creates a new read-only instance of ConfigRegistry, bound to a specific deployed contract.
func NewConfigRegistryCaller(address common.Address, caller bind.ContractCaller) (*ConfigRegistryCaller, error) {
	contract, err := bindConfigRegistry(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ConfigRegistryCaller{contract: contract}, nil
}

// NewConfigRegistryTransactor creates a new write-only instance of ConfigRegistry, bound to a specific deployed contract.
func NewConfigRegistryTransactor(address common.Address, transactor bind.ContractTransactor) (*ConfigRegistryTransactor, error) {
	contract, err := bindConfigRegistry(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &ConfigRegistryTransactor{contract: contract}, nil
}

// NewConfigRegistryFilterer creates a new log filterer instance of ConfigRegistry, bound to a specific deployed contract.
func NewConfigRegistryFilterer(address common.Address, filterer bind.ContractFilterer) (*ConfigRegistryFilterer, error) {
	contract, err := bindConfigRegistry(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &ConfigRegistryFilterer{contract: contract}, nil
}

// bindConfigRegistry binds a generic wrapper to an already deployed contract.
func bindConfigRegistry(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := ConfigRegistryMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ConfigRegistry *ConfigRegistryRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ConfigRegistry.Contract.ConfigRegistryCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ConfigRegistry *ConfigRegistryRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ConfigRegistry.Contract.ConfigRegistryTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ConfigRegistry *ConfigRegistryRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ConfigRegistry.Contract.ConfigRegistryTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ConfigRegistry *ConfigRegistryCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ConfigRegistry.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ConfigRegistry *ConfigRegistryTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ConfigRegistry.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ConfigRegistry *ConfigRegistryTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ConfigRegistry.Contract.contract.Transact(opts, method, params...)
}

// ComputeIssuerId is a free data retrieval call binding the contract method 0xcf925361.
//
// Solidity: function computeIssuerId(address[] issuerKeys_, uint256 threshold_) view returns(address)
func (_ConfigRegistry *ConfigRegistryCaller) ComputeIssuerId(opts *bind.CallOpts, issuerKeys_ []common.Address, threshold_ *big.Int) (common.Address, error) {
	var out []interface{}
	err := _ConfigRegistry.contract.Call(opts, &out, "computeIssuerId", issuerKeys_, threshold_)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// ComputeIssuerId is a free data retrieval call binding the contract method 0xcf925361.
//
// Solidity: function computeIssuerId(address[] issuerKeys_, uint256 threshold_) view returns(address)
func (_ConfigRegistry *ConfigRegistrySession) ComputeIssuerId(issuerKeys_ []common.Address, threshold_ *big.Int) (common.Address, error) {
	return _ConfigRegistry.Contract.ComputeIssuerId(&_ConfigRegistry.CallOpts, issuerKeys_, threshold_)
}

// ComputeIssuerId is a free data retrieval call binding the contract method 0xcf925361.
//
// Solidity: function computeIssuerId(address[] issuerKeys_, uint256 threshold_) view returns(address)
func (_ConfigRegistry *ConfigRegistryCallerSession) ComputeIssuerId(issuerKeys_ []common.Address, threshold_ *big.Int) (common.Address, error) {
	return _ConfigRegistry.Contract.ComputeIssuerId(&_ConfigRegistry.CallOpts, issuerKeys_, threshold_)
}

// Eip712Domain is a free data retrieval call binding the contract method 0x84b0196e.
//
// Solidity: function eip712Domain() view returns(bytes1 fields, string name, string version, uint256 chainId, address verifyingContract, bytes32 salt, uint256[] extensions)
func (_ConfigRegistry *ConfigRegistryCaller) Eip712Domain(opts *bind.CallOpts) (struct {
	Fields            [1]byte
	Name              string
	Version           string
	ChainId           *big.Int
	VerifyingContract common.Address
	Salt              [32]byte
	Extensions        []*big.Int
}, error) {
	var out []interface{}
	err := _ConfigRegistry.contract.Call(opts, &out, "eip712Domain")

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
func (_ConfigRegistry *ConfigRegistrySession) Eip712Domain() (struct {
	Fields            [1]byte
	Name              string
	Version           string
	ChainId           *big.Int
	VerifyingContract common.Address
	Salt              [32]byte
	Extensions        []*big.Int
}, error) {
	return _ConfigRegistry.Contract.Eip712Domain(&_ConfigRegistry.CallOpts)
}

// Eip712Domain is a free data retrieval call binding the contract method 0x84b0196e.
//
// Solidity: function eip712Domain() view returns(bytes1 fields, string name, string version, uint256 chainId, address verifyingContract, bytes32 salt, uint256[] extensions)
func (_ConfigRegistry *ConfigRegistryCallerSession) Eip712Domain() (struct {
	Fields            [1]byte
	Name              string
	Version           string
	ChainId           *big.Int
	VerifyingContract common.Address
	Salt              [32]byte
	Extensions        []*big.Int
}, error) {
	return _ConfigRegistry.Contract.Eip712Domain(&_ConfigRegistry.CallOpts)
}

// IsRegistered is a free data retrieval call binding the contract method 0xc3c5a547.
//
// Solidity: function isRegistered(address issuerId) view returns(bool)
func (_ConfigRegistry *ConfigRegistryCaller) IsRegistered(opts *bind.CallOpts, issuerId common.Address) (bool, error) {
	var out []interface{}
	err := _ConfigRegistry.contract.Call(opts, &out, "isRegistered", issuerId)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsRegistered is a free data retrieval call binding the contract method 0xc3c5a547.
//
// Solidity: function isRegistered(address issuerId) view returns(bool)
func (_ConfigRegistry *ConfigRegistrySession) IsRegistered(issuerId common.Address) (bool, error) {
	return _ConfigRegistry.Contract.IsRegistered(&_ConfigRegistry.CallOpts, issuerId)
}

// IsRegistered is a free data retrieval call binding the contract method 0xc3c5a547.
//
// Solidity: function isRegistered(address issuerId) view returns(bool)
func (_ConfigRegistry *ConfigRegistryCallerSession) IsRegistered(issuerId common.Address) (bool, error) {
	return _ConfigRegistry.Contract.IsRegistered(&_ConfigRegistry.CallOpts, issuerId)
}

// IssuerKeys is a free data retrieval call binding the contract method 0x787779e8.
//
// Solidity: function issuerKeys(address issuerId) view returns(address[])
func (_ConfigRegistry *ConfigRegistryCaller) IssuerKeys(opts *bind.CallOpts, issuerId common.Address) ([]common.Address, error) {
	var out []interface{}
	err := _ConfigRegistry.contract.Call(opts, &out, "issuerKeys", issuerId)

	if err != nil {
		return *new([]common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new([]common.Address)).(*[]common.Address)

	return out0, err

}

// IssuerKeys is a free data retrieval call binding the contract method 0x787779e8.
//
// Solidity: function issuerKeys(address issuerId) view returns(address[])
func (_ConfigRegistry *ConfigRegistrySession) IssuerKeys(issuerId common.Address) ([]common.Address, error) {
	return _ConfigRegistry.Contract.IssuerKeys(&_ConfigRegistry.CallOpts, issuerId)
}

// IssuerKeys is a free data retrieval call binding the contract method 0x787779e8.
//
// Solidity: function issuerKeys(address issuerId) view returns(address[])
func (_ConfigRegistry *ConfigRegistryCallerSession) IssuerKeys(issuerId common.Address) ([]common.Address, error) {
	return _ConfigRegistry.Contract.IssuerKeys(&_ConfigRegistry.CallOpts, issuerId)
}

// IssuerSettings is a free data retrieval call binding the contract method 0xe6639d47.
//
// Solidity: function issuerSettings(address issuerId) view returns(address[] issuerKeys_, uint256 threshold_, uint256 nonce_)
func (_ConfigRegistry *ConfigRegistryCaller) IssuerSettings(opts *bind.CallOpts, issuerId common.Address) (struct {
	IssuerKeys []common.Address
	Threshold  *big.Int
	Nonce      *big.Int
}, error) {
	var out []interface{}
	err := _ConfigRegistry.contract.Call(opts, &out, "issuerSettings", issuerId)

	outstruct := new(struct {
		IssuerKeys []common.Address
		Threshold  *big.Int
		Nonce      *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.IssuerKeys = *abi.ConvertType(out[0], new([]common.Address)).(*[]common.Address)
	outstruct.Threshold = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	outstruct.Nonce = *abi.ConvertType(out[2], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// IssuerSettings is a free data retrieval call binding the contract method 0xe6639d47.
//
// Solidity: function issuerSettings(address issuerId) view returns(address[] issuerKeys_, uint256 threshold_, uint256 nonce_)
func (_ConfigRegistry *ConfigRegistrySession) IssuerSettings(issuerId common.Address) (struct {
	IssuerKeys []common.Address
	Threshold  *big.Int
	Nonce      *big.Int
}, error) {
	return _ConfigRegistry.Contract.IssuerSettings(&_ConfigRegistry.CallOpts, issuerId)
}

// IssuerSettings is a free data retrieval call binding the contract method 0xe6639d47.
//
// Solidity: function issuerSettings(address issuerId) view returns(address[] issuerKeys_, uint256 threshold_, uint256 nonce_)
func (_ConfigRegistry *ConfigRegistryCallerSession) IssuerSettings(issuerId common.Address) (struct {
	IssuerKeys []common.Address
	Threshold  *big.Int
	Nonce      *big.Int
}, error) {
	return _ConfigRegistry.Contract.IssuerSettings(&_ConfigRegistry.CallOpts, issuerId)
}

// Nonce is a free data retrieval call binding the contract method 0x70ae92d2.
//
// Solidity: function nonce(address issuerId) view returns(uint256)
func (_ConfigRegistry *ConfigRegistryCaller) Nonce(opts *bind.CallOpts, issuerId common.Address) (*big.Int, error) {
	var out []interface{}
	err := _ConfigRegistry.contract.Call(opts, &out, "nonce", issuerId)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Nonce is a free data retrieval call binding the contract method 0x70ae92d2.
//
// Solidity: function nonce(address issuerId) view returns(uint256)
func (_ConfigRegistry *ConfigRegistrySession) Nonce(issuerId common.Address) (*big.Int, error) {
	return _ConfigRegistry.Contract.Nonce(&_ConfigRegistry.CallOpts, issuerId)
}

// Nonce is a free data retrieval call binding the contract method 0x70ae92d2.
//
// Solidity: function nonce(address issuerId) view returns(uint256)
func (_ConfigRegistry *ConfigRegistryCallerSession) Nonce(issuerId common.Address) (*big.Int, error) {
	return _ConfigRegistry.Contract.Nonce(&_ConfigRegistry.CallOpts, issuerId)
}

// Threshold is a free data retrieval call binding the contract method 0xc86ec2bf.
//
// Solidity: function threshold(address issuerId) view returns(uint256)
func (_ConfigRegistry *ConfigRegistryCaller) Threshold(opts *bind.CallOpts, issuerId common.Address) (*big.Int, error) {
	var out []interface{}
	err := _ConfigRegistry.contract.Call(opts, &out, "threshold", issuerId)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Threshold is a free data retrieval call binding the contract method 0xc86ec2bf.
//
// Solidity: function threshold(address issuerId) view returns(uint256)
func (_ConfigRegistry *ConfigRegistrySession) Threshold(issuerId common.Address) (*big.Int, error) {
	return _ConfigRegistry.Contract.Threshold(&_ConfigRegistry.CallOpts, issuerId)
}

// Threshold is a free data retrieval call binding the contract method 0xc86ec2bf.
//
// Solidity: function threshold(address issuerId) view returns(uint256)
func (_ConfigRegistry *ConfigRegistryCallerSession) Threshold(issuerId common.Address) (*big.Int, error) {
	return _ConfigRegistry.Contract.Threshold(&_ConfigRegistry.CallOpts, issuerId)
}

// RegisterIssuer is a paid mutator transaction binding the contract method 0x36ab7a61.
//
// Solidity: function registerIssuer(address[] issuerKeys_, uint256 threshold_, bytes[] signatures) returns(address issuerId)
func (_ConfigRegistry *ConfigRegistryTransactor) RegisterIssuer(opts *bind.TransactOpts, issuerKeys_ []common.Address, threshold_ *big.Int, signatures [][]byte) (*types.Transaction, error) {
	return _ConfigRegistry.contract.Transact(opts, "registerIssuer", issuerKeys_, threshold_, signatures)
}

// RegisterIssuer is a paid mutator transaction binding the contract method 0x36ab7a61.
//
// Solidity: function registerIssuer(address[] issuerKeys_, uint256 threshold_, bytes[] signatures) returns(address issuerId)
func (_ConfigRegistry *ConfigRegistrySession) RegisterIssuer(issuerKeys_ []common.Address, threshold_ *big.Int, signatures [][]byte) (*types.Transaction, error) {
	return _ConfigRegistry.Contract.RegisterIssuer(&_ConfigRegistry.TransactOpts, issuerKeys_, threshold_, signatures)
}

// RegisterIssuer is a paid mutator transaction binding the contract method 0x36ab7a61.
//
// Solidity: function registerIssuer(address[] issuerKeys_, uint256 threshold_, bytes[] signatures) returns(address issuerId)
func (_ConfigRegistry *ConfigRegistryTransactorSession) RegisterIssuer(issuerKeys_ []common.Address, threshold_ *big.Int, signatures [][]byte) (*types.Transaction, error) {
	return _ConfigRegistry.Contract.RegisterIssuer(&_ConfigRegistry.TransactOpts, issuerKeys_, threshold_, signatures)
}

// SetConfig is a paid mutator transaction binding the contract method 0x4a2c5521.
//
// Solidity: function setConfig(address issuerId, bytes32 key, bytes32 checksum, uint256 expectedNonce, bytes[] signatures) returns()
func (_ConfigRegistry *ConfigRegistryTransactor) SetConfig(opts *bind.TransactOpts, issuerId common.Address, key [32]byte, checksum [32]byte, expectedNonce *big.Int, signatures [][]byte) (*types.Transaction, error) {
	return _ConfigRegistry.contract.Transact(opts, "setConfig", issuerId, key, checksum, expectedNonce, signatures)
}

// SetConfig is a paid mutator transaction binding the contract method 0x4a2c5521.
//
// Solidity: function setConfig(address issuerId, bytes32 key, bytes32 checksum, uint256 expectedNonce, bytes[] signatures) returns()
func (_ConfigRegistry *ConfigRegistrySession) SetConfig(issuerId common.Address, key [32]byte, checksum [32]byte, expectedNonce *big.Int, signatures [][]byte) (*types.Transaction, error) {
	return _ConfigRegistry.Contract.SetConfig(&_ConfigRegistry.TransactOpts, issuerId, key, checksum, expectedNonce, signatures)
}

// SetConfig is a paid mutator transaction binding the contract method 0x4a2c5521.
//
// Solidity: function setConfig(address issuerId, bytes32 key, bytes32 checksum, uint256 expectedNonce, bytes[] signatures) returns()
func (_ConfigRegistry *ConfigRegistryTransactorSession) SetConfig(issuerId common.Address, key [32]byte, checksum [32]byte, expectedNonce *big.Int, signatures [][]byte) (*types.Transaction, error) {
	return _ConfigRegistry.Contract.SetConfig(&_ConfigRegistry.TransactOpts, issuerId, key, checksum, expectedNonce, signatures)
}

// SetConfigWithData is a paid mutator transaction binding the contract method 0xd9617437.
//
// Solidity: function setConfigWithData(address issuerId, bytes32 key, bytes data, uint256 expectedNonce, bytes[] signatures) returns()
func (_ConfigRegistry *ConfigRegistryTransactor) SetConfigWithData(opts *bind.TransactOpts, issuerId common.Address, key [32]byte, data []byte, expectedNonce *big.Int, signatures [][]byte) (*types.Transaction, error) {
	return _ConfigRegistry.contract.Transact(opts, "setConfigWithData", issuerId, key, data, expectedNonce, signatures)
}

// SetConfigWithData is a paid mutator transaction binding the contract method 0xd9617437.
//
// Solidity: function setConfigWithData(address issuerId, bytes32 key, bytes data, uint256 expectedNonce, bytes[] signatures) returns()
func (_ConfigRegistry *ConfigRegistrySession) SetConfigWithData(issuerId common.Address, key [32]byte, data []byte, expectedNonce *big.Int, signatures [][]byte) (*types.Transaction, error) {
	return _ConfigRegistry.Contract.SetConfigWithData(&_ConfigRegistry.TransactOpts, issuerId, key, data, expectedNonce, signatures)
}

// SetConfigWithData is a paid mutator transaction binding the contract method 0xd9617437.
//
// Solidity: function setConfigWithData(address issuerId, bytes32 key, bytes data, uint256 expectedNonce, bytes[] signatures) returns()
func (_ConfigRegistry *ConfigRegistryTransactorSession) SetConfigWithData(issuerId common.Address, key [32]byte, data []byte, expectedNonce *big.Int, signatures [][]byte) (*types.Transaction, error) {
	return _ConfigRegistry.Contract.SetConfigWithData(&_ConfigRegistry.TransactOpts, issuerId, key, data, expectedNonce, signatures)
}

// UpdateIssuerSettings is a paid mutator transaction binding the contract method 0x1b7ae749.
//
// Solidity: function updateIssuerSettings(address issuerId, address[] newIssuerKeys, uint256 newThreshold, uint256 expectedNonce, bytes[] signatures) returns()
func (_ConfigRegistry *ConfigRegistryTransactor) UpdateIssuerSettings(opts *bind.TransactOpts, issuerId common.Address, newIssuerKeys []common.Address, newThreshold *big.Int, expectedNonce *big.Int, signatures [][]byte) (*types.Transaction, error) {
	return _ConfigRegistry.contract.Transact(opts, "updateIssuerSettings", issuerId, newIssuerKeys, newThreshold, expectedNonce, signatures)
}

// UpdateIssuerSettings is a paid mutator transaction binding the contract method 0x1b7ae749.
//
// Solidity: function updateIssuerSettings(address issuerId, address[] newIssuerKeys, uint256 newThreshold, uint256 expectedNonce, bytes[] signatures) returns()
func (_ConfigRegistry *ConfigRegistrySession) UpdateIssuerSettings(issuerId common.Address, newIssuerKeys []common.Address, newThreshold *big.Int, expectedNonce *big.Int, signatures [][]byte) (*types.Transaction, error) {
	return _ConfigRegistry.Contract.UpdateIssuerSettings(&_ConfigRegistry.TransactOpts, issuerId, newIssuerKeys, newThreshold, expectedNonce, signatures)
}

// UpdateIssuerSettings is a paid mutator transaction binding the contract method 0x1b7ae749.
//
// Solidity: function updateIssuerSettings(address issuerId, address[] newIssuerKeys, uint256 newThreshold, uint256 expectedNonce, bytes[] signatures) returns()
func (_ConfigRegistry *ConfigRegistryTransactorSession) UpdateIssuerSettings(issuerId common.Address, newIssuerKeys []common.Address, newThreshold *big.Int, expectedNonce *big.Int, signatures [][]byte) (*types.Transaction, error) {
	return _ConfigRegistry.Contract.UpdateIssuerSettings(&_ConfigRegistry.TransactOpts, issuerId, newIssuerKeys, newThreshold, expectedNonce, signatures)
}

// ConfigRegistryConfigCommittedIterator is returned from FilterConfigCommitted and is used to iterate over the raw logs and unpacked data for ConfigCommitted events raised by the ConfigRegistry contract.
type ConfigRegistryConfigCommittedIterator struct {
	Event *ConfigRegistryConfigCommitted // Event containing the contract specifics and raw log

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
func (it *ConfigRegistryConfigCommittedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ConfigRegistryConfigCommitted)
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
		it.Event = new(ConfigRegistryConfigCommitted)
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
func (it *ConfigRegistryConfigCommittedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ConfigRegistryConfigCommittedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ConfigRegistryConfigCommitted represents a ConfigCommitted event raised by the ConfigRegistry contract.
type ConfigRegistryConfigCommitted struct {
	IssuerId common.Address
	Key      [32]byte
	Checksum [32]byte
	NewNonce *big.Int
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterConfigCommitted is a free log retrieval operation binding the contract event 0xa727e52bcb62c238dd87fa146891db1d7e2ed6a4394fd61d5b299f63d633f61a.
//
// Solidity: event ConfigCommitted(address indexed issuerId, bytes32 indexed key, bytes32 checksum, uint256 newNonce)
func (_ConfigRegistry *ConfigRegistryFilterer) FilterConfigCommitted(opts *bind.FilterOpts, issuerId []common.Address, key [][32]byte) (*ConfigRegistryConfigCommittedIterator, error) {

	var issuerIdRule []interface{}
	for _, issuerIdItem := range issuerId {
		issuerIdRule = append(issuerIdRule, issuerIdItem)
	}
	var keyRule []interface{}
	for _, keyItem := range key {
		keyRule = append(keyRule, keyItem)
	}

	logs, sub, err := _ConfigRegistry.contract.FilterLogs(opts, "ConfigCommitted", issuerIdRule, keyRule)
	if err != nil {
		return nil, err
	}
	return &ConfigRegistryConfigCommittedIterator{contract: _ConfigRegistry.contract, event: "ConfigCommitted", logs: logs, sub: sub}, nil
}

// WatchConfigCommitted is a free log subscription operation binding the contract event 0xa727e52bcb62c238dd87fa146891db1d7e2ed6a4394fd61d5b299f63d633f61a.
//
// Solidity: event ConfigCommitted(address indexed issuerId, bytes32 indexed key, bytes32 checksum, uint256 newNonce)
func (_ConfigRegistry *ConfigRegistryFilterer) WatchConfigCommitted(opts *bind.WatchOpts, sink chan<- *ConfigRegistryConfigCommitted, issuerId []common.Address, key [][32]byte) (event.Subscription, error) {

	var issuerIdRule []interface{}
	for _, issuerIdItem := range issuerId {
		issuerIdRule = append(issuerIdRule, issuerIdItem)
	}
	var keyRule []interface{}
	for _, keyItem := range key {
		keyRule = append(keyRule, keyItem)
	}

	logs, sub, err := _ConfigRegistry.contract.WatchLogs(opts, "ConfigCommitted", issuerIdRule, keyRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ConfigRegistryConfigCommitted)
				if err := _ConfigRegistry.contract.UnpackLog(event, "ConfigCommitted", log); err != nil {
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

// ParseConfigCommitted is a log parse operation binding the contract event 0xa727e52bcb62c238dd87fa146891db1d7e2ed6a4394fd61d5b299f63d633f61a.
//
// Solidity: event ConfigCommitted(address indexed issuerId, bytes32 indexed key, bytes32 checksum, uint256 newNonce)
func (_ConfigRegistry *ConfigRegistryFilterer) ParseConfigCommitted(log types.Log) (*ConfigRegistryConfigCommitted, error) {
	event := new(ConfigRegistryConfigCommitted)
	if err := _ConfigRegistry.contract.UnpackLog(event, "ConfigCommitted", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ConfigRegistryConfigWithDataCommittedIterator is returned from FilterConfigWithDataCommitted and is used to iterate over the raw logs and unpacked data for ConfigWithDataCommitted events raised by the ConfigRegistry contract.
type ConfigRegistryConfigWithDataCommittedIterator struct {
	Event *ConfigRegistryConfigWithDataCommitted // Event containing the contract specifics and raw log

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
func (it *ConfigRegistryConfigWithDataCommittedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ConfigRegistryConfigWithDataCommitted)
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
		it.Event = new(ConfigRegistryConfigWithDataCommitted)
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
func (it *ConfigRegistryConfigWithDataCommittedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ConfigRegistryConfigWithDataCommittedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ConfigRegistryConfigWithDataCommitted represents a ConfigWithDataCommitted event raised by the ConfigRegistry contract.
type ConfigRegistryConfigWithDataCommitted struct {
	IssuerId common.Address
	Key      [32]byte
	Checksum [32]byte
	Data     []byte
	NewNonce *big.Int
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterConfigWithDataCommitted is a free log retrieval operation binding the contract event 0x4c57f3906e38359a4c973d7c01f642d34582e5d00263cb0452206692462b9903.
//
// Solidity: event ConfigWithDataCommitted(address indexed issuerId, bytes32 indexed key, bytes32 checksum, bytes data, uint256 newNonce)
func (_ConfigRegistry *ConfigRegistryFilterer) FilterConfigWithDataCommitted(opts *bind.FilterOpts, issuerId []common.Address, key [][32]byte) (*ConfigRegistryConfigWithDataCommittedIterator, error) {

	var issuerIdRule []interface{}
	for _, issuerIdItem := range issuerId {
		issuerIdRule = append(issuerIdRule, issuerIdItem)
	}
	var keyRule []interface{}
	for _, keyItem := range key {
		keyRule = append(keyRule, keyItem)
	}

	logs, sub, err := _ConfigRegistry.contract.FilterLogs(opts, "ConfigWithDataCommitted", issuerIdRule, keyRule)
	if err != nil {
		return nil, err
	}
	return &ConfigRegistryConfigWithDataCommittedIterator{contract: _ConfigRegistry.contract, event: "ConfigWithDataCommitted", logs: logs, sub: sub}, nil
}

// WatchConfigWithDataCommitted is a free log subscription operation binding the contract event 0x4c57f3906e38359a4c973d7c01f642d34582e5d00263cb0452206692462b9903.
//
// Solidity: event ConfigWithDataCommitted(address indexed issuerId, bytes32 indexed key, bytes32 checksum, bytes data, uint256 newNonce)
func (_ConfigRegistry *ConfigRegistryFilterer) WatchConfigWithDataCommitted(opts *bind.WatchOpts, sink chan<- *ConfigRegistryConfigWithDataCommitted, issuerId []common.Address, key [][32]byte) (event.Subscription, error) {

	var issuerIdRule []interface{}
	for _, issuerIdItem := range issuerId {
		issuerIdRule = append(issuerIdRule, issuerIdItem)
	}
	var keyRule []interface{}
	for _, keyItem := range key {
		keyRule = append(keyRule, keyItem)
	}

	logs, sub, err := _ConfigRegistry.contract.WatchLogs(opts, "ConfigWithDataCommitted", issuerIdRule, keyRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ConfigRegistryConfigWithDataCommitted)
				if err := _ConfigRegistry.contract.UnpackLog(event, "ConfigWithDataCommitted", log); err != nil {
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

// ParseConfigWithDataCommitted is a log parse operation binding the contract event 0x4c57f3906e38359a4c973d7c01f642d34582e5d00263cb0452206692462b9903.
//
// Solidity: event ConfigWithDataCommitted(address indexed issuerId, bytes32 indexed key, bytes32 checksum, bytes data, uint256 newNonce)
func (_ConfigRegistry *ConfigRegistryFilterer) ParseConfigWithDataCommitted(log types.Log) (*ConfigRegistryConfigWithDataCommitted, error) {
	event := new(ConfigRegistryConfigWithDataCommitted)
	if err := _ConfigRegistry.contract.UnpackLog(event, "ConfigWithDataCommitted", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ConfigRegistryEIP712DomainChangedIterator is returned from FilterEIP712DomainChanged and is used to iterate over the raw logs and unpacked data for EIP712DomainChanged events raised by the ConfigRegistry contract.
type ConfigRegistryEIP712DomainChangedIterator struct {
	Event *ConfigRegistryEIP712DomainChanged // Event containing the contract specifics and raw log

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
func (it *ConfigRegistryEIP712DomainChangedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ConfigRegistryEIP712DomainChanged)
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
		it.Event = new(ConfigRegistryEIP712DomainChanged)
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
func (it *ConfigRegistryEIP712DomainChangedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ConfigRegistryEIP712DomainChangedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ConfigRegistryEIP712DomainChanged represents a EIP712DomainChanged event raised by the ConfigRegistry contract.
type ConfigRegistryEIP712DomainChanged struct {
	Raw types.Log // Blockchain specific contextual infos
}

// FilterEIP712DomainChanged is a free log retrieval operation binding the contract event 0x0a6387c9ea3628b88a633bb4f3b151770f70085117a15f9bf3787cda53f13d31.
//
// Solidity: event EIP712DomainChanged()
func (_ConfigRegistry *ConfigRegistryFilterer) FilterEIP712DomainChanged(opts *bind.FilterOpts) (*ConfigRegistryEIP712DomainChangedIterator, error) {

	logs, sub, err := _ConfigRegistry.contract.FilterLogs(opts, "EIP712DomainChanged")
	if err != nil {
		return nil, err
	}
	return &ConfigRegistryEIP712DomainChangedIterator{contract: _ConfigRegistry.contract, event: "EIP712DomainChanged", logs: logs, sub: sub}, nil
}

// WatchEIP712DomainChanged is a free log subscription operation binding the contract event 0x0a6387c9ea3628b88a633bb4f3b151770f70085117a15f9bf3787cda53f13d31.
//
// Solidity: event EIP712DomainChanged()
func (_ConfigRegistry *ConfigRegistryFilterer) WatchEIP712DomainChanged(opts *bind.WatchOpts, sink chan<- *ConfigRegistryEIP712DomainChanged) (event.Subscription, error) {

	logs, sub, err := _ConfigRegistry.contract.WatchLogs(opts, "EIP712DomainChanged")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ConfigRegistryEIP712DomainChanged)
				if err := _ConfigRegistry.contract.UnpackLog(event, "EIP712DomainChanged", log); err != nil {
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
func (_ConfigRegistry *ConfigRegistryFilterer) ParseEIP712DomainChanged(log types.Log) (*ConfigRegistryEIP712DomainChanged, error) {
	event := new(ConfigRegistryEIP712DomainChanged)
	if err := _ConfigRegistry.contract.UnpackLog(event, "EIP712DomainChanged", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ConfigRegistryIssuerRegisteredIterator is returned from FilterIssuerRegistered and is used to iterate over the raw logs and unpacked data for IssuerRegistered events raised by the ConfigRegistry contract.
type ConfigRegistryIssuerRegisteredIterator struct {
	Event *ConfigRegistryIssuerRegistered // Event containing the contract specifics and raw log

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
func (it *ConfigRegistryIssuerRegisteredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ConfigRegistryIssuerRegistered)
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
		it.Event = new(ConfigRegistryIssuerRegistered)
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
func (it *ConfigRegistryIssuerRegisteredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ConfigRegistryIssuerRegisteredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ConfigRegistryIssuerRegistered represents a IssuerRegistered event raised by the ConfigRegistry contract.
type ConfigRegistryIssuerRegistered struct {
	IssuerId   common.Address
	IssuerKeys []common.Address
	Threshold  *big.Int
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterIssuerRegistered is a free log retrieval operation binding the contract event 0xd4a9f2e1f27ddd97144ea11c3582e0ab1fd269dedf40a26c12b5f8ff70c78121.
//
// Solidity: event IssuerRegistered(address indexed issuerId, address[] issuerKeys, uint256 threshold)
func (_ConfigRegistry *ConfigRegistryFilterer) FilterIssuerRegistered(opts *bind.FilterOpts, issuerId []common.Address) (*ConfigRegistryIssuerRegisteredIterator, error) {

	var issuerIdRule []interface{}
	for _, issuerIdItem := range issuerId {
		issuerIdRule = append(issuerIdRule, issuerIdItem)
	}

	logs, sub, err := _ConfigRegistry.contract.FilterLogs(opts, "IssuerRegistered", issuerIdRule)
	if err != nil {
		return nil, err
	}
	return &ConfigRegistryIssuerRegisteredIterator{contract: _ConfigRegistry.contract, event: "IssuerRegistered", logs: logs, sub: sub}, nil
}

// WatchIssuerRegistered is a free log subscription operation binding the contract event 0xd4a9f2e1f27ddd97144ea11c3582e0ab1fd269dedf40a26c12b5f8ff70c78121.
//
// Solidity: event IssuerRegistered(address indexed issuerId, address[] issuerKeys, uint256 threshold)
func (_ConfigRegistry *ConfigRegistryFilterer) WatchIssuerRegistered(opts *bind.WatchOpts, sink chan<- *ConfigRegistryIssuerRegistered, issuerId []common.Address) (event.Subscription, error) {

	var issuerIdRule []interface{}
	for _, issuerIdItem := range issuerId {
		issuerIdRule = append(issuerIdRule, issuerIdItem)
	}

	logs, sub, err := _ConfigRegistry.contract.WatchLogs(opts, "IssuerRegistered", issuerIdRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ConfigRegistryIssuerRegistered)
				if err := _ConfigRegistry.contract.UnpackLog(event, "IssuerRegistered", log); err != nil {
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

// ParseIssuerRegistered is a log parse operation binding the contract event 0xd4a9f2e1f27ddd97144ea11c3582e0ab1fd269dedf40a26c12b5f8ff70c78121.
//
// Solidity: event IssuerRegistered(address indexed issuerId, address[] issuerKeys, uint256 threshold)
func (_ConfigRegistry *ConfigRegistryFilterer) ParseIssuerRegistered(log types.Log) (*ConfigRegistryIssuerRegistered, error) {
	event := new(ConfigRegistryIssuerRegistered)
	if err := _ConfigRegistry.contract.UnpackLog(event, "IssuerRegistered", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ConfigRegistryIssuerSettingsUpdatedIterator is returned from FilterIssuerSettingsUpdated and is used to iterate over the raw logs and unpacked data for IssuerSettingsUpdated events raised by the ConfigRegistry contract.
type ConfigRegistryIssuerSettingsUpdatedIterator struct {
	Event *ConfigRegistryIssuerSettingsUpdated // Event containing the contract specifics and raw log

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
func (it *ConfigRegistryIssuerSettingsUpdatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ConfigRegistryIssuerSettingsUpdated)
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
		it.Event = new(ConfigRegistryIssuerSettingsUpdated)
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
func (it *ConfigRegistryIssuerSettingsUpdatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ConfigRegistryIssuerSettingsUpdatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ConfigRegistryIssuerSettingsUpdated represents a IssuerSettingsUpdated event raised by the ConfigRegistry contract.
type ConfigRegistryIssuerSettingsUpdated struct {
	IssuerId      common.Address
	NewIssuerKeys []common.Address
	NewThreshold  *big.Int
	NewNonce      *big.Int
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterIssuerSettingsUpdated is a free log retrieval operation binding the contract event 0xf3f30d031c773609e9b6609fb5782eaf3e1c63707cb41d667e45674eaf0819b7.
//
// Solidity: event IssuerSettingsUpdated(address indexed issuerId, address[] newIssuerKeys, uint256 newThreshold, uint256 newNonce)
func (_ConfigRegistry *ConfigRegistryFilterer) FilterIssuerSettingsUpdated(opts *bind.FilterOpts, issuerId []common.Address) (*ConfigRegistryIssuerSettingsUpdatedIterator, error) {

	var issuerIdRule []interface{}
	for _, issuerIdItem := range issuerId {
		issuerIdRule = append(issuerIdRule, issuerIdItem)
	}

	logs, sub, err := _ConfigRegistry.contract.FilterLogs(opts, "IssuerSettingsUpdated", issuerIdRule)
	if err != nil {
		return nil, err
	}
	return &ConfigRegistryIssuerSettingsUpdatedIterator{contract: _ConfigRegistry.contract, event: "IssuerSettingsUpdated", logs: logs, sub: sub}, nil
}

// WatchIssuerSettingsUpdated is a free log subscription operation binding the contract event 0xf3f30d031c773609e9b6609fb5782eaf3e1c63707cb41d667e45674eaf0819b7.
//
// Solidity: event IssuerSettingsUpdated(address indexed issuerId, address[] newIssuerKeys, uint256 newThreshold, uint256 newNonce)
func (_ConfigRegistry *ConfigRegistryFilterer) WatchIssuerSettingsUpdated(opts *bind.WatchOpts, sink chan<- *ConfigRegistryIssuerSettingsUpdated, issuerId []common.Address) (event.Subscription, error) {

	var issuerIdRule []interface{}
	for _, issuerIdItem := range issuerId {
		issuerIdRule = append(issuerIdRule, issuerIdItem)
	}

	logs, sub, err := _ConfigRegistry.contract.WatchLogs(opts, "IssuerSettingsUpdated", issuerIdRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ConfigRegistryIssuerSettingsUpdated)
				if err := _ConfigRegistry.contract.UnpackLog(event, "IssuerSettingsUpdated", log); err != nil {
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

// ParseIssuerSettingsUpdated is a log parse operation binding the contract event 0xf3f30d031c773609e9b6609fb5782eaf3e1c63707cb41d667e45674eaf0819b7.
//
// Solidity: event IssuerSettingsUpdated(address indexed issuerId, address[] newIssuerKeys, uint256 newThreshold, uint256 newNonce)
func (_ConfigRegistry *ConfigRegistryFilterer) ParseIssuerSettingsUpdated(log types.Log) (*ConfigRegistryIssuerSettingsUpdated, error) {
	event := new(ConfigRegistryIssuerSettingsUpdated)
	if err := _ConfigRegistry.contract.UnpackLog(event, "IssuerSettingsUpdated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// SPDX-License-Identifier: MIT
pragma solidity 0.8.34;
// Canonical vectors from clearnet-sdk/pkg/blockchain/evm/testdata/eip712.json.
// The Go test independently verifies every vector using go-ethereum apitypes.
import {Test} from "forge-std/Test.sol";
import {ConfigRegistryDigests} from "../src/libraries/ConfigRegistryDigests.sol";

contract EIP712VectorsTest is Test {
    function test_vectorRegisterIssuer() public pure {
        address[] memory keys = new address[](3);
        keys[0] = address(1);
        keys[1] = address(2);
        keys[2] = address(3);
        bytes32 structure = ConfigRegistryDigests.registrationStructHash(keys, 2);
        assertEq(structure, 0x4b7116e47b75b5b3a65fc68648a6bb46c3585e52b516832d064ca3e0344596bd);
        bytes32 domain = keccak256(
            abi.encode(
                keccak256("EIP712Domain(string name,string version,uint256 chainId,address verifyingContract)"),
                keccak256("YellowConfigRegistry"),
                keccak256("1"),
                uint256(31337),
                address(100)
            )
        );
        assertEq(domain, 0xe3494c96e75b3c9e7479a92492d1125d76ff805948f28da8580b6f4b70eb521c);
        bytes32 digest = keccak256(abi.encodePacked(hex"1901", domain, structure));
        assertEq(digest, 0xc42c1c69c248917dc3ef86ccfd4c592409fad394a2377208ad353c558ad78daa);
        assertEq(
            ecrecover(
                digest,
                27,
                0x25b612be48a624b8eb5ca792200e749558134b73f60cf5ca2b9f6d4e8eb1356f,
                0x581fd48f03b5995186fe48b02bee4bd32f32fb909c1adf963abcbf766361750c
            ),
            0x7E5F4552091A69125d5DfCb7b8C2659029395Bdf
        );
    }

    function test_vectorSetConfig() public pure {
        bytes32 structure = ConfigRegistryDigests.setConfigStructHash(
            address(12),
            bytes32(0x000000000000000000000000000000000000000000000000000000000000000d),
            bytes32(0x000000000000000000000000000000000000000000000000000000000000000e),
            7
        );
        assertEq(structure, 0xfcf9ebd7a31d483e1972eb26d07a0aa554f889a7ffc04f40da90b6af4a22a65e);
        bytes32 domain = keccak256(
            abi.encode(
                keccak256("EIP712Domain(string name,string version,uint256 chainId,address verifyingContract)"),
                keccak256("YellowConfigRegistry"),
                keccak256("1"),
                uint256(31337),
                address(100)
            )
        );
        assertEq(domain, 0xe3494c96e75b3c9e7479a92492d1125d76ff805948f28da8580b6f4b70eb521c);
        bytes32 digest = keccak256(abi.encodePacked(hex"1901", domain, structure));
        assertEq(digest, 0x69bf7057d0c7984e601f35e56a6704de3016c76d755db25ddbf2cad6beae4e31);
        assertEq(
            ecrecover(
                digest,
                27,
                0x0d7003083ef16a70335f9e316fe3c274b5d006e31ac1aeb81acf816a9dd73d62,
                0x2f8e4684cd19d7109741a5266740460e49f0dab3dc962db240dc0884eb8f5961
            ),
            0x7E5F4552091A69125d5DfCb7b8C2659029395Bdf
        );
    }

    function test_vectorSetConfigWithData() public pure {
        bytes32 structure = ConfigRegistryDigests.setConfigWithDataStructHash(
            address(12), bytes32(0x000000000000000000000000000000000000000000000000000000000000000d), hex"010203", 7
        );
        assertEq(structure, 0xe07124522c2b8290855b64dd6565de24b918e1b9ab76cf161b350715b5844fbb);
        bytes32 domain = keccak256(
            abi.encode(
                keccak256("EIP712Domain(string name,string version,uint256 chainId,address verifyingContract)"),
                keccak256("YellowConfigRegistry"),
                keccak256("1"),
                uint256(31337),
                address(100)
            )
        );
        assertEq(domain, 0xe3494c96e75b3c9e7479a92492d1125d76ff805948f28da8580b6f4b70eb521c);
        bytes32 digest = keccak256(abi.encodePacked(hex"1901", domain, structure));
        assertEq(digest, 0x109299c76de3c094c6e885472a39a185fefa205d233da886ba0d6f481265592c);
        assertEq(
            ecrecover(
                digest,
                27,
                0xe50853fd64a186735dfb530ccf29209835976ce3b4f12bcfe68afb8d49ca1bb6,
                0x7c75fe77fa24b0e934717f0a8a44e513f1750fd561c39a001d94f15666056a1e
            ),
            0x7E5F4552091A69125d5DfCb7b8C2659029395Bdf
        );
    }

    function test_vectorUpdateIssuerSettings() public pure {
        address[] memory keys = new address[](3);
        keys[0] = address(1);
        keys[1] = address(2);
        keys[2] = address(3);
        bytes32 structure = ConfigRegistryDigests.updateIssuerSettingsStructHash(address(12), keys, 2, 7);
        assertEq(structure, 0x20274439143199ef58fc96dd1550a53eaa7c6848413699cad5df4e07c57e04f1);
        bytes32 domain = keccak256(
            abi.encode(
                keccak256("EIP712Domain(string name,string version,uint256 chainId,address verifyingContract)"),
                keccak256("YellowConfigRegistry"),
                keccak256("1"),
                uint256(31337),
                address(100)
            )
        );
        assertEq(domain, 0xe3494c96e75b3c9e7479a92492d1125d76ff805948f28da8580b6f4b70eb521c);
        bytes32 digest = keccak256(abi.encodePacked(hex"1901", domain, structure));
        assertEq(digest, 0x2df862b0649cbe3d5b0e3b0fc4a429e3168d93f7ce621eec5e9d0e4db2e7fa29);
        assertEq(
            ecrecover(
                digest,
                27,
                0xb55c90cc1a33a9e283ae20018e3d6b3495f4e884a02cbaaf620da6d0852f5ee1,
                0x54a9e3fd668b09c5f0687dd4f113df533a43b627e2c0db8e0f8f757eb084a2e6
            ),
            0x7E5F4552091A69125d5DfCb7b8C2659029395Bdf
        );
    }
}

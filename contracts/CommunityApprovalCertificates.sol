// SPDX-License-Identifier: MIT
pragma solidity 0.8.24;

import "@openzeppelin/contracts-upgradeable/token/ERC721/ERC721Upgradeable.sol";
import "@openzeppelin/contracts-upgradeable/access/Ownable2StepUpgradeable.sol";
import "@openzeppelin/contracts-upgradeable/proxy/utils/Initializable.sol";
import "@openzeppelin/contracts-upgradeable/proxy/utils/UUPSUpgradeable.sol";

    error NonTransferable();
    error ZeroAddress();
    error OnlyModuleCanMint();
    error OnlyModuleOrOwnerOrShariahBoard();

contract CommunityApprovalCertificates is
Initializable,
ERC721Upgradeable,
Ownable2StepUpgradeable,
UUPSUpgradeable
{
    uint256 public version;
    string public baseURI;
    uint256 private _nextTokenId; // next token id to mint
    address public moduleAddress; // address of mudule contrat that will be able to mint tokens
    address public ShariahAdvisoryBoard; // address of Shariah Advisory Board
    mapping(address => uint256) private _ownedTokensId;

    modifier onlyModule() {
        if (moduleAddress != _msgSender()) {
            revert OnlyModuleCanMint();
        }
        _;
    }

    modifier onlyOwnerOrModuleOrShariahBoard() {
        if (
            moduleAddress != _msgSender() &&
            owner() != _msgSender() &&
            ShariahAdvisoryBoard != _msgSender()
        ) {
            revert OnlyModuleOrOwnerOrShariahBoard();
        }
        _;
    }

    /// @custom:oz-upgrades-unsafe-allow constructor
    constructor() {
        _disableInitializers();
    }

    function initialize(
        address initialOwner,
        address module,
        address shariahOracle,
        string calldata _baseURI
    ) public initializer {
        __ERC721_init("Community Approval Certificates", "CAC");
        __Ownable_init(initialOwner);
        __UUPSUpgradeable_init();
        version = 1;
        moduleAddress = module;
        ShariahAdvisoryBoard = shariahOracle;
        baseURI = _baseURI;
        _nextTokenId = 1;
    }

    function safeMint(address to) public onlyModule {
        if (to == address(0)) {
            revert ZeroAddress();
        }
        require(balanceOf(to) == 0, "Only one certificate per address");
        uint256 tokenId = _nextTokenId++;
        _safeMint(to, tokenId);
        _ownedTokensId[to] = tokenId;
    }

    function batchSafeMint(address[] calldata to) public onlyModule {
        for (uint256 i = 0; i < to.length; i++) {
            if (to[i] == address(0)) {
                revert ZeroAddress();
            }
            uint256 tokenId = _nextTokenId++;
            require(balanceOf(to[i]) == 0, "Only one certificate per address");
            _safeMint(to[i], tokenId);
            _ownedTokensId[to[i]] = tokenId;
        }
    }

    function burn(
        address from
    ) public virtual onlyOwnerOrModuleOrShariahBoard {
        if (from == address(0)) {
            revert ZeroAddress();
        }
        uint256 tokenId = _ownedTokensId[from];
        _update(address(0), tokenId, from);
        _ownedTokensId[from] = 0;
    }

    function batchBurn(
        address[] calldata from
    ) public virtual onlyOwnerOrModuleOrShariahBoard {
        for (uint256 i = 0; i < from.length; i++) {
            if (from[i] == address(0)) {
                revert ZeroAddress();
            }
            uint256 tokenId = _ownedTokensId[from[i]];
            _update(address(0), tokenId, from[i]);
            _ownedTokensId[from[i]] = 0;
        }
    }

    function _authorizeUpgrade(
        address newImplementation
    ) internal override onlyModule {
        version++;
    }

    function _baseURI() internal view override returns (string memory) {
        return baseURI;
    }

    function setBaseURI(string calldata _uri) public onlyOwner {
        baseURI = _uri;
    }

    function tokenURI(
        uint256 tokenId
    ) public view virtual override returns (string memory) {
        _requireOwned(tokenId);

        string memory uri = _baseURI();
        return bytes(uri).length > 0 ? uri : "";
    }

    // // disable any transfers
    function transferFrom(
        address from,
        address to,
        uint256 tokenId
    ) public virtual override {
        revert NonTransferable();
    }

    /**
     * @dev See {IERC721-safeTransferFrom}.
     */
    function safeTransferFrom(
        address from,
        address to,
        uint256 tokenId,
        bytes memory data
    ) public virtual override {
        revert NonTransferable();
    }
}

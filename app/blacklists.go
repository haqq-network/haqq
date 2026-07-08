package app

// blockedAccounts is the single source of truth for accounts frozen after the
// precompile abuse exploit. Keys are bech32 (haqq1...) addresses. The same set
// is enforced across every ante pipeline (Cosmos, EIP-712 and EVM): the EVM
// decorator derives the bech32 form from the recovered sender's address bytes,
// since a haqq1... account and its 0x... EVM address are the same 20-byte
// account.
var blockedAccounts = map[string]bool{
	"haqq19pxv2r4key79twjfv0gdc5yhc4xmw9vqxkj2nl": true,
	"haqq15gl76py2lqqrlawzs0afkmh9k7kxc6wmvcqqlm": true,
}

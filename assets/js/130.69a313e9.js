(window.webpackJsonp=window.webpackJsonp||[]).push([[130],{685:function(e,t,o){"use strict";o.r(t);var s=o(1),n=Object(s.a)({},(function(){var e=this,t=e.$createElement,o=e._self._c||t;return o("ContentSlotsDistributor",{attrs:{"slot-key":e.$parent.slotKey}},[o("h1",{attrs:{id:"x-bank"}},[o("a",{staticClass:"header-anchor",attrs:{href:"#x-bank"}},[e._v("#")]),e._v(" "),o("code",[e._v("x/bank")])]),e._v(" "),o("h2",{attrs:{id:"abstract"}},[o("a",{staticClass:"header-anchor",attrs:{href:"#abstract"}},[e._v("#")]),e._v(" Abstract")]),e._v(" "),o("p",[e._v("This document specifies the bank module of the Cosmos SDK.")]),e._v(" "),o("p",[e._v("The bank module is responsible for handling multi-asset coin transfers between\naccounts and tracking special-case pseudo-transfers which must work differently\nwith particular kinds of accounts (notably delegating/undelegating for vesting\naccounts). It exposes several interfaces with varying capabilities for secure\ninteraction with other modules which must alter user balances.")]),e._v(" "),o("p",[e._v("In addition, the bank module tracks and provides query support for the total\nsupply of all assets used in the application.")]),e._v(" "),o("p",[e._v("This module will be used in the Cosmos Hub.")]),e._v(" "),o("h2",{attrs:{id:"supply"}},[o("a",{staticClass:"header-anchor",attrs:{href:"#supply"}},[e._v("#")]),e._v(" Supply")]),e._v(" "),o("p",[e._v("The "),o("code",[e._v("supply")]),e._v(" functionality:")]),e._v(" "),o("ul",[o("li",[e._v("passively tracks the total supply of coins within a chain,")]),e._v(" "),o("li",[e._v("provides a pattern for modules to hold/interact with "),o("code",[e._v("Coins")]),e._v(", and")]),e._v(" "),o("li",[e._v("introduces the invariant check to verify a chain's total supply.")])]),e._v(" "),o("h3",{attrs:{id:"total-supply"}},[o("a",{staticClass:"header-anchor",attrs:{href:"#total-supply"}},[e._v("#")]),e._v(" Total Supply")]),e._v(" "),o("p",[e._v("The total "),o("code",[e._v("Supply")]),e._v(" of the network is equal to the sum of all coins from the\naccount. The total supply is updated every time a "),o("code",[e._v("Coin")]),e._v(" is minted (eg: as part\nof the inflation mechanism) or burned (eg: due to slashing or if a governance\nproposal is vetoed).")]),e._v(" "),o("h2",{attrs:{id:"module-accounts"}},[o("a",{staticClass:"header-anchor",attrs:{href:"#module-accounts"}},[e._v("#")]),e._v(" Module Accounts")]),e._v(" "),o("p",[e._v("The supply functionality introduces a new type of "),o("code",[e._v("auth.Account")]),e._v(" which can be used by\nmodules to allocate tokens and in special cases mint or burn tokens. At a base\nlevel these module accounts are capable of sending/receiving tokens to and from\n"),o("code",[e._v("auth.Account")]),e._v("s and other module accounts. This design replaces previous\nalternative designs where, to hold tokens, modules would burn the incoming\ntokens from the sender account, and then track those tokens internally. Later,\nin order to send tokens, the module would need to effectively mint tokens\nwithin a destination account. The new design removes duplicate logic between\nmodules to perform this accounting.")]),e._v(" "),o("p",[e._v("The "),o("code",[e._v("ModuleAccount")]),e._v(" interface is defined as follows:")]),e._v(" "),o("tm-code-block",{staticClass:"codeblock",attrs:{language:"go",base64:"dHlwZSBNb2R1bGVBY2NvdW50IGludGVyZmFjZSB7CiAgYXV0aC5BY2NvdW50ICAgICAgICAgICAgICAgLy8gc2FtZSBtZXRob2RzIGFzIHRoZSBBY2NvdW50IGludGVyZmFjZQoKICBHZXROYW1lKCkgc3RyaW5nICAgICAgICAgICAvLyBuYW1lIG9mIHRoZSBtb2R1bGU7IHVzZWQgdG8gb2J0YWluIHRoZSBhZGRyZXNzCiAgR2V0UGVybWlzc2lvbnMoKSBbXXN0cmluZyAgLy8gcGVybWlzc2lvbnMgb2YgbW9kdWxlIGFjY291bnQKICBIYXNQZXJtaXNzaW9uKHN0cmluZykgYm9vbAp9Cg=="}}),e._v(" "),o("blockquote",[o("p",[o("strong",[e._v("WARNING!")]),e._v("\nAny module or message handler that allows either direct or indirect sending of funds must explicitly guarantee those funds cannot be sent to module accounts (unless allowed).")])]),e._v(" "),o("p",[e._v("The supply "),o("code",[e._v("Keeper")]),e._v(" also introduces new wrapper functions for the auth "),o("code",[e._v("Keeper")]),e._v("\nand the bank "),o("code",[e._v("Keeper")]),e._v(" that are related to "),o("code",[e._v("ModuleAccount")]),e._v("s in order to be able\nto:")]),e._v(" "),o("ul",[o("li",[e._v("Get and set "),o("code",[e._v("ModuleAccount")]),e._v("s by providing the "),o("code",[e._v("Name")]),e._v(".")]),e._v(" "),o("li",[e._v("Send coins from and to other "),o("code",[e._v("ModuleAccount")]),e._v("s or standard "),o("code",[e._v("Account")]),e._v("s\n("),o("code",[e._v("BaseAccount")]),e._v(" or "),o("code",[e._v("VestingAccount")]),e._v(") by passing only the "),o("code",[e._v("Name")]),e._v(".")]),e._v(" "),o("li",[o("code",[e._v("Mint")]),e._v(" or "),o("code",[e._v("Burn")]),e._v(" coins for a "),o("code",[e._v("ModuleAccount")]),e._v(" (restricted to its permissions).")])]),e._v(" "),o("h3",{attrs:{id:"permissions"}},[o("a",{staticClass:"header-anchor",attrs:{href:"#permissions"}},[e._v("#")]),e._v(" Permissions")]),e._v(" "),o("p",[e._v("Each "),o("code",[e._v("ModuleAccount")]),e._v(" has a different set of permissions that provide different\nobject capabilities to perform certain actions. Permissions need to be\nregistered upon the creation of the supply "),o("code",[e._v("Keeper")]),e._v(" so that every time a\n"),o("code",[e._v("ModuleAccount")]),e._v(" calls the allowed functions, the "),o("code",[e._v("Keeper")]),e._v(" can lookup the\npermissions to that specific account and perform or not the action.")]),e._v(" "),o("p",[e._v("The available permissions are:")]),e._v(" "),o("ul",[o("li",[o("code",[e._v("Minter")]),e._v(": allows for a module to mint a specific amount of coins.")]),e._v(" "),o("li",[o("code",[e._v("Burner")]),e._v(": allows for a module to burn a specific amount of coins.")]),e._v(" "),o("li",[o("code",[e._v("Staking")]),e._v(": allows for a module to delegate and undelegate a specific amount of coins.")])]),e._v(" "),o("h2",{attrs:{id:"contents"}},[o("a",{staticClass:"header-anchor",attrs:{href:"#contents"}},[e._v("#")]),e._v(" Contents")]),e._v(" "),o("ol",[o("li",[o("strong",[o("RouterLink",{attrs:{to:"/modules/bank/01_state.html"}},[e._v("State")])],1)]),e._v(" "),o("li",[o("strong",[o("RouterLink",{attrs:{to:"/modules/bank/02_keepers.html"}},[e._v("Keepers")])],1),e._v(" "),o("ul",[o("li",[o("RouterLink",{attrs:{to:"/modules/bank/02_keepers.html#common-types"}},[e._v("Common Types")])],1),e._v(" "),o("li",[o("RouterLink",{attrs:{to:"/modules/bank/02_keepers.html#basekeeper"}},[e._v("BaseKeeper")])],1),e._v(" "),o("li",[o("RouterLink",{attrs:{to:"/modules/bank/02_keepers.html#sendkeeper"}},[e._v("SendKeeper")])],1),e._v(" "),o("li",[o("RouterLink",{attrs:{to:"/modules/bank/02_keepers.html#viewkeeper"}},[e._v("ViewKeeper")])],1)])]),e._v(" "),o("li",[o("strong",[o("RouterLink",{attrs:{to:"/modules/bank/03_messages.html"}},[e._v("Messages")])],1),e._v(" "),o("ul",[o("li",[o("RouterLink",{attrs:{to:"/modules/bank/03_messages.html#msgsend"}},[e._v("MsgSend")])],1)])]),e._v(" "),o("li",[o("strong",[o("RouterLink",{attrs:{to:"/modules/bank/04_events.html"}},[e._v("Events")])],1),e._v(" "),o("ul",[o("li",[o("RouterLink",{attrs:{to:"/modules/bank/04_events.html#handlers"}},[e._v("Handlers")])],1)])]),e._v(" "),o("li",[o("strong",[o("RouterLink",{attrs:{to:"/modules/bank/05_params.html"}},[e._v("Parameters")])],1)])])],1)}),[],!1,null,null,null);t.default=n.exports}}]);
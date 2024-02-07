(window.webpackJsonp=window.webpackJsonp||[]).push([[73],{623:function(e,t,n){"use strict";n.r(t);var s=n(1),o=Object(s.a)({},(function(){var e=this,t=e.$createElement,n=e._self._c||t;return n("ContentSlotsDistributor",{attrs:{"slot-key":e.$parent.slotKey}},[n("h1",{attrs:{id:"events"}},[n("a",{staticClass:"header-anchor",attrs:{href:"#events"}},[e._v("#")]),e._v(" Events")]),e._v(" "),n("p",{attrs:{synopsis:""}},[n("code",[e._v("Event")]),e._v("s are objects that contain information about the execution of the application. They are\nmainly used by service providers like block explorers and wallet to track the execution of various\nmessages and index transactions.")]),e._v(" "),n("h2",{attrs:{id:"pre-requisite-readings"}},[n("a",{staticClass:"header-anchor",attrs:{href:"#pre-requisite-readings"}},[e._v("#")]),e._v(" Pre-requisite Readings")]),e._v(" "),n("ul",[n("li",{attrs:{prereq:""}},[n("a",{attrs:{href:"https://docs.cosmos.network/master/core/events.html",target:"_blank",rel:"noopener noreferrer"}},[e._v("Cosmos SDK Events"),n("OutboundLink")],1)]),e._v(" "),n("li",{attrs:{prereq:""}},[n("a",{attrs:{href:"https://geth.ethereum.org/docs/rpc/pubsub",target:"_blank",rel:"noopener noreferrer"}},[e._v("Ethereum's PubSub JSON-RPC API"),n("OutboundLink")],1)])]),e._v(" "),n("h2",{attrs:{id:"subscribing-to-events"}},[n("a",{staticClass:"header-anchor",attrs:{href:"#subscribing-to-events"}},[e._v("#")]),e._v(" Subscribing to Events")]),e._v(" "),n("h3",{attrs:{id:"sdk-and-tendermint-events"}},[n("a",{staticClass:"header-anchor",attrs:{href:"#sdk-and-tendermint-events"}},[e._v("#")]),e._v(" SDK and Tendermint Events")]),e._v(" "),n("p",[e._v("It is possible to subscribe to "),n("code",[e._v("Events")]),e._v(" via Tendermint's "),n("a",{attrs:{href:"https://docs.tendermint.com/v0.33/app-dev/subscribing-to-events-via-websocket.html",target:"_blank",rel:"noopener noreferrer"}},[e._v("Websocket"),n("OutboundLink")],1),e._v(".\nThis is done by calling the "),n("code",[e._v("subscribe")]),e._v(" RPC method via Websocket:")]),e._v(" "),n("tm-code-block",{staticClass:"codeblock",attrs:{language:"json",base64:"ewogICAgJnF1b3Q7anNvbnJwYyZxdW90OzogJnF1b3Q7Mi4wJnF1b3Q7LAogICAgJnF1b3Q7bWV0aG9kJnF1b3Q7OiAmcXVvdDtzdWJzY3JpYmUmcXVvdDssCiAgICAmcXVvdDtpZCZxdW90OzogJnF1b3Q7MCZxdW90OywKICAgICZxdW90O3BhcmFtcyZxdW90OzogewogICAgICAgICZxdW90O3F1ZXJ5JnF1b3Q7OiAmcXVvdDt0bS5ldmVudD0nZXZlbnRDYXRlZ29yeScgQU5EIGV2ZW50VHlwZS5ldmVudEF0dHJpYnV0ZT0nYXR0cmlidXRlVmFsdWUnJnF1b3Q7CiAgICB9Cn0K"}}),e._v(" "),n("p",[e._v("The main "),n("code",[e._v("eventCategory")]),e._v(" you can subscribe to are:")]),e._v(" "),n("ul",[n("li",[n("code",[e._v("NewBlock")]),e._v(": Contains "),n("code",[e._v("events")]),e._v(" triggered during "),n("code",[e._v("BeginBlock")]),e._v(" and "),n("code",[e._v("EndBlock")]),e._v(".")]),e._v(" "),n("li",[n("code",[e._v("Tx")]),e._v(": Contains "),n("code",[e._v("events")]),e._v(" triggered during "),n("code",[e._v("DeliverTx")]),e._v(" (i.e. transaction processing).")]),e._v(" "),n("li",[n("code",[e._v("ValidatorSetUpdates")]),e._v(": Contains validator set updates for the block.")])]),e._v(" "),n("p",[e._v("These events are triggered from the "),n("code",[e._v("state")]),e._v(" package after a block is committed. You can get the full\nlist of "),n("code",[e._v("event")]),e._v(" categories\n"),n("a",{attrs:{href:"https://godoc.org/github.com/tendermint/tendermint/types#pkg-constants",target:"_blank",rel:"noopener noreferrer"}},[e._v("here"),n("OutboundLink")],1),e._v(".")]),e._v(" "),n("p",[e._v("The "),n("code",[e._v("type")]),e._v(" and "),n("code",[e._v("attribute")]),e._v(" value of the "),n("code",[e._v("query")]),e._v(" allow you to filter the specific "),n("code",[e._v("event")]),e._v(" you are\nlooking for. For example, a "),n("code",[e._v("MsgEthereumTx")]),e._v(" transaction triggers an "),n("code",[e._v("event")]),e._v(" of type "),n("code",[e._v("ethermint")]),e._v(" and\nhas "),n("code",[e._v("sender")]),e._v(" and "),n("code",[e._v("recipient")]),e._v(" as "),n("code",[e._v("attributes")]),e._v(". Subscribing to this "),n("code",[e._v("event")]),e._v(" would be done like so:")]),e._v(" "),n("tm-code-block",{staticClass:"codeblock",attrs:{language:"json",base64:"ewogICAgJnF1b3Q7anNvbnJwYyZxdW90OzogJnF1b3Q7Mi4wJnF1b3Q7LAogICAgJnF1b3Q7bWV0aG9kJnF1b3Q7OiAmcXVvdDtzdWJzY3JpYmUmcXVvdDssCiAgICAmcXVvdDtpZCZxdW90OzogJnF1b3Q7MCZxdW90OywKICAgICZxdW90O3BhcmFtcyZxdW90OzogewogICAgICAgICZxdW90O3F1ZXJ5JnF1b3Q7OiAmcXVvdDt0bS5ldmVudD0nVHgnIEFORCBldGhlcmV1bS5yZWNpcGllbnQ9J2hleEFkZHJlc3MnJnF1b3Q7CiAgICB9Cn0K"}}),e._v(" "),n("p",[e._v("where "),n("code",[e._v("hexAddress")]),e._v(" is an Ethereum hex address (eg: "),n("code",[e._v("0x1122334455667788990011223344556677889900")]),e._v(").")]),e._v(" "),n("h3",{attrs:{id:"ethereum-json-rpc-events"}},[n("a",{staticClass:"header-anchor",attrs:{href:"#ethereum-json-rpc-events"}},[e._v("#")]),e._v(" Ethereum JSON-RPC Events")]),e._v(" "),n("p",[e._v("Haqq also supports the Ethereum "),n("a",{attrs:{href:"https://eth.wiki/json-rpc/API",target:"_blank",rel:"noopener noreferrer"}},[e._v("JSON-RPC"),n("OutboundLink")],1),e._v(" filters calls to\nsubscribe to "),n("a",{attrs:{href:"https://eth.wiki/json-rpc/API#eth_newfilter",target:"_blank",rel:"noopener noreferrer"}},[e._v("state logs"),n("OutboundLink")],1),e._v(",\n"),n("a",{attrs:{href:"https://eth.wiki/json-rpc/API#eth_newblockfilter",target:"_blank",rel:"noopener noreferrer"}},[e._v("blocks"),n("OutboundLink")],1),e._v(" or "),n("a",{attrs:{href:"https://eth.wiki/json-rpc/API#eth_newpendingtransactionfilter",target:"_blank",rel:"noopener noreferrer"}},[e._v("pending\ntransactions"),n("OutboundLink")],1),e._v(" changes.")]),e._v(" "),n("p",[e._v("Under the hood, it uses the Tendermint RPC client's event system to process subscriptions that are\nthen formatted to Ethereum-compatible events.")]),e._v(" "),n("tm-code-block",{staticClass:"codeblock",attrs:{language:"bash",base64:"Y3VybCAtWCBQT1NUIC0tZGF0YSAneyZxdW90O2pzb25ycGMmcXVvdDs6JnF1b3Q7Mi4wJnF1b3Q7LCZxdW90O21ldGhvZCZxdW90OzomcXVvdDtldGhfbmV3QmxvY2tGaWx0ZXImcXVvdDssJnF1b3Q7cGFyYW1zJnF1b3Q7OltdLCZxdW90O2lkJnF1b3Q7OjF9JyAtSCAmcXVvdDtDb250ZW50LVR5cGU6IGFwcGxpY2F0aW9uL2pzb24mcXVvdDsgaHR0cDovL2xvY2FsaG9zdDo4NTQ1Cgp7JnF1b3Q7anNvbnJwYyZxdW90OzomcXVvdDsyLjAmcXVvdDssJnF1b3Q7aWQmcXVvdDs6MSwmcXVvdDtyZXN1bHQmcXVvdDs6JnF1b3Q7MHgzNTAzZGU1ZjBjNzY2YzY4Zjc4YTAzYTNiMDUwMzZhNSZxdW90O30K"}}),e._v(" "),n("p",[e._v("Then you can check if the state changes with the "),n("a",{attrs:{href:"https://eth.wiki/json-rpc/API#eth_getfilterchanges",target:"_blank",rel:"noopener noreferrer"}},[n("code",[e._v("eth_getFilterChanges")]),n("OutboundLink")],1),e._v(" call:")]),e._v(" "),n("tm-code-block",{staticClass:"codeblock",attrs:{language:"bash",base64:"Y3VybCAtWCBQT1NUIC0tZGF0YSAneyZxdW90O2pzb25ycGMmcXVvdDs6JnF1b3Q7Mi4wJnF1b3Q7LCZxdW90O21ldGhvZCZxdW90OzomcXVvdDtldGhfZ2V0RmlsdGVyQ2hhbmdlcyZxdW90OywmcXVvdDtwYXJhbXMmcXVvdDs6WyZxdW90OzB4MzUwM2RlNWYwYzc2NmM2OGY3OGEwM2EzYjA1MDM2YTUmcXVvdDtdLCZxdW90O2lkJnF1b3Q7OjF9JyAtSCAmcXVvdDtDb250ZW50LVR5cGU6IGFwcGxpY2F0aW9uL2pzb24mcXVvdDsgaHR0cDovL2xvY2FsaG9zdDo4NTQ1Cgp7JnF1b3Q7anNvbnJwYyZxdW90OzomcXVvdDsyLjAmcXVvdDssJnF1b3Q7aWQmcXVvdDs6MSwmcXVvdDtyZXN1bHQmcXVvdDs6WyZxdW90OzB4N2Q0NGRjZWZmMDVkNTk2M2I1YmM4MWRmN2U5Zjc5YjI3ZTc3N2IwYTAzYTZmZWNhMDlmMzQ0N2I5OWM2ZmE3MSZxdW90OywmcXVvdDsweDM5NjFlNDA1MGMyN2NlMDE0NWQzNzUyNTViM2NiODI5YTViNGU3OTVhYzQ3NWMwNWEyMTliMzczMzcyM2QzNzYmcXVvdDssJnF1b3Q7MHhkN2E0OTdmOTUxNjdkNjNlNmZlY2E3MGYzNDRkOWY2ZTg0M2QwOTdiNjI3MjliOGY0M2JkY2Q1ZmViZjE0MmFiJnF1b3Q7LCZxdW90OzB4NTVkODBhNGJhNmVmNTRmMmE4YzBiOTk1ODlkMDE3YjgxMGVkMTNhMWZkYTZhMTExZTFiODc3MjViYzhjZWIwZSZxdW90OywmcXVvdDsweDllOGI5MmMxNzI4MGRkMDVmMjU2MmFmNmVlYTMyODUxODFjNTYyZWJmNDFmYzc1ODUyN2Q0YzMwMzY0YmNiYzQmcXVvdDssJnF1b3Q7MHg3MzUzYTRiOWQ2YjM1YzllYWZlY2NhZjk3MjJkZDI5M2M0NmFlMmZmZDQwOTNiMjM2NzE2NWMzNjIwYTBjN2M5JnF1b3Q7LCZxdW90OzB4MDI2ZDkxYmRhNjFjODc4OWM1OTYzMmMzNDliMzhmZDdlNzU1N2U2YjU5OGI5NDg3OTY1NGE2NDRjZmE3NWYzMCZxdW90OywmcXVvdDsweDczZTMyNDVkNGRkYzNiYmE0OGZhNjc2MzNmOTk5M2M2ZTExNzI4YTM2NDAxZmExMjA2NDM3ZjhiZTk0ZWYxZDMmcXVvdDtdfQo="}}),e._v(" "),n("h2",{attrs:{id:"websocket-connection"}},[n("a",{staticClass:"header-anchor",attrs:{href:"#websocket-connection"}},[e._v("#")]),e._v(" Websocket Connection")]),e._v(" "),n("h3",{attrs:{id:"tendermint-websocket"}},[n("a",{staticClass:"header-anchor",attrs:{href:"#tendermint-websocket"}},[e._v("#")]),e._v(" Tendermint Websocket")]),e._v(" "),n("p",[e._v("To start a connection with the Tendermint websocket you need to define the address with the "),n("code",[e._v("--rpc.laddr")]),e._v("\nflag when starting the node (default "),n("code",[e._v("tcp://127.0.0.1:26657")]),e._v("):")]),e._v(" "),n("tm-code-block",{staticClass:"codeblock",attrs:{language:"bash",base64:"aGFxcWQgc3RhcnQgLS1ycGMubGFkZHI9JnF1b3Q7dGNwOi8vMTI3LjAuMC4xOjI2NjU3JnF1b3Q7Cg=="}}),e._v(" "),n("p",[e._v("Then, start a websocket subscription with "),n("a",{attrs:{href:"https://github.com/hashrocket/ws",target:"_blank",rel:"noopener noreferrer"}},[e._v("ws"),n("OutboundLink")],1)]),e._v(" "),n("tm-code-block",{staticClass:"codeblock",attrs:{language:"bash",base64:"IyBjb25uZWN0IHRvIHRlbmRlcm1pbnQgd2Vic29ja2V0IGF0IHBvcnQgODA4MCBhcyBkZWZpbmVkIGFib3ZlCndzIHdzOi8vbG9jYWxob3N0OjgwODAvd2Vic29ja2V0CgojIHN1YnNjcmliZSB0byBuZXcgVGVuZGVybWludCBibG9jayBoZWFkZXJzCiZndDsgeyAmcXVvdDtqc29ucnBjJnF1b3Q7OiAmcXVvdDsyLjAmcXVvdDssICZxdW90O21ldGhvZCZxdW90OzogJnF1b3Q7c3Vic2NyaWJlJnF1b3Q7LCAmcXVvdDtwYXJhbXMmcXVvdDs6IFsmcXVvdDt0bS5ldmVudD0nTmV3QmxvY2tIZWFkZXInJnF1b3Q7XSwgJnF1b3Q7aWQmcXVvdDs6IDEgfQo="}}),e._v(" "),n("h3",{attrs:{id:"ethereum-websocket"}},[n("a",{staticClass:"header-anchor",attrs:{href:"#ethereum-websocket"}},[e._v("#")]),e._v(" Ethereum Websocket")]),e._v(" "),n("p",[e._v("Since Haqq runs uses Tendermint Core as it's consensus Engine and it's built with the Cosmos\nSDK framework, it inherits the event format from them. However, in order to support the native Web3\ncompatibility for websockets of the "),n("a",{attrs:{href:"https://geth.ethereum.org/docs/rpc/pubsub",target:"_blank",rel:"noopener noreferrer"}},[e._v("Ethereum's\nPubSubAPI"),n("OutboundLink")],1),e._v(", Haqq needs to cast the Tendermint\nresponses retrieved into the Ethereum types.")]),e._v(" "),n("p",[e._v("You can start a connection with the Ethereum websocket using the "),n("code",[e._v("--json-rpc.ws-address")]),e._v(" flag when starting\nthe node (default "),n("code",[e._v('"0.0.0.0:8546"')]),e._v("):")]),e._v(" "),n("tm-code-block",{staticClass:"codeblock",attrs:{language:"bash",base64:"aGFxcWQgc3RhcnQgIC0tanNvbi1ycGMuYWRkcmVzcyZxdW90OzAuMC4wLjA6ODU0NSZxdW90OyAtLWpzb24tcnBjLndzLWFkZHJlc3M9JnF1b3Q7MC4wLjAuMDo4NTQ2JnF1b3Q7IC0tZXZtLnJwYy5hcGk9JnF1b3Q7ZXRoLHdlYjMsbmV0LHR4cG9vbCxkZWJ1ZyZxdW90OyAtLWpzb24tcnBjLmVuYWJsZQo="}}),e._v(" "),n("p",[e._v("Then, start a websocket subscription with "),n("a",{attrs:{href:"https://github.com/hashrocket/ws",target:"_blank",rel:"noopener noreferrer"}},[n("code",[e._v("ws")]),n("OutboundLink")],1)]),e._v(" "),n("tm-code-block",{staticClass:"codeblock",attrs:{language:"bash",base64:"IyBjb25uZWN0IHRvIHRlbmRlcm1pbnQgd2Vic29jZXQgYXQgcG9ydCA4NTQ2IGFzIGRlZmluZWQgYWJvdmUKd3Mgd3M6Ly9sb2NhbGhvc3Q6ODU0Ni8KCiMgc3Vic2NyaWJlIHRvIG5ldyBFdGhlcmV1bS1mb3JtYXR0ZWQgYmxvY2sgSGVhZGVycwomZ3Q7IHsmcXVvdDtpZCZxdW90OzogMSwgJnF1b3Q7bWV0aG9kJnF1b3Q7OiAmcXVvdDtldGhfc3Vic2NyaWJlJnF1b3Q7LCAmcXVvdDtwYXJhbXMmcXVvdDs6IFsmcXVvdDtuZXdIZWFkcyZxdW90Oywge31dfQombHQ7IHsmcXVvdDtqc29ucnBjJnF1b3Q7OiZxdW90OzIuMCZxdW90OywmcXVvdDtyZXN1bHQmcXVvdDs6JnF1b3Q7MHg0NGUwMTBjYjJjMzE2MWU5YzAyMjA3ZmYxNzIxNjZlZiZxdW90OywmcXVvdDtpZCZxdW90OzoxfQo="}})],1)}),[],!1,null,null,null);t.default=o.exports}}]);
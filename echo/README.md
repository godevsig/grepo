Echo example demonstrates that:

- server can handle requests from many clients simultaneously in auto scale way
- client can use multiplexed connection that provides independent streams
- subscrip/publish patten can be used in client and server

To start an echo server:
`go run server/echoserver.go`

To start an echo client that sends echo request and gets echo response in multiplexed streams:
`go run client/echoclient.go`

To start an echo client that subscrips "SubWhoElseEvent" event and continuously query the server by
"WhoElse" message:
`go run client/echoclient.go -cmd whoelse`

# Tokyo C Messanger Server

## Establish a Server

Execute `sudo go run *.go -port 9000` and you have an endpoint at `http://localhost:9000/`.

## Make a Listen
Just execute
`curl http://localhost/messages/groupname`.

## Publish a Message

JSON messages will be accepted:
`curl -X POST -H'Content-Type: application/json' -d '{"Text":"Hello, world!"}' http://localhost/messages/groupname`.
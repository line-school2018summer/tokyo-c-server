package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
)

type MessageServer struct {
	listeners *Hub
}

func NewMessageServer(stamper func(Message) error) *MessageServer {
	return &MessageServer{NewHub(stamper)}
}

func (server *MessageServer) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	last_slash_index := strings.LastIndex(request.URL.Path, "/")

	if last_slash_index == -1 {
		writer.WriteHeader(404)
		fmt.Fprintln(writer, "not found")
		return
	}

	channel_string := request.URL.Path[last_slash_index+1:]

	var (
		channel int
		err     error
	)

	if channel, err = strconv.Atoi(channel_string); err != nil {
		writer.WriteHeader(401)
		fmt.Fprintln(writer, "invalid channel name")
		return
	}

	switch request.Method {
	case "GET":
		var (
			flushable bool
			flusher   http.Flusher
		)

		writer.Header().Set("Transfer-Encoding", "chunked")

		if flusher, flushable = writer.(http.Flusher); !flushable {
			return
		}

		listener := make(chan Message)

		server.listeners.Subscribe(channel, listener)

		defer server.listeners.Unsubscribe(listener)

		for {
			if payload, err := json.Marshal(<-listener); err == nil {
				fmt.Fprintf(writer, "%s\n", payload)
			}

			flusher.Flush()
		}
	case "POST":
		if request.Header.Get("Content-Type") != "application/json" {
			writer.WriteHeader(400)
			fmt.Fprintln(writer, "bad content type")
			return
		}

		var (
			err     error
			buffer  []byte
			message Message
		)

		if buffer, err = ioutil.ReadAll(request.Body); err != nil {
			writer.WriteHeader(400)
			fmt.Fprintln(writer, "invalid content stream")
			return
		}

		request.Body.Close()

		if json.Unmarshal(buffer, &message) != nil {
			writer.WriteHeader(400)
			fmt.Fprintln(writer, "corrupt content format")
			return
		}

		err = server.listeners.Publish(channel, message)

		if err != nil {
			writer.WriteHeader(201)
			fmt.Fprintln(writer, err.Error())
			return
		}

		writer.WriteHeader(201)
		fmt.Fprintln(writer, "Posted.")
	}
}

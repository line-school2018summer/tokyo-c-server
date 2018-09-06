package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
)

func endpoint_friendships(writer http.ResponseWriter, request *http.Request) *http_status {
	subject, err := authenticate(request)

	if err != nil {
		return &http_status{401, err.Error()}
	}

	person_id := strings.TrimPrefix(request.URL.Path, "/friendships/")

	if person_id == "" {
		return endpoint_friendships_without_person_id(subject, writer, request)
	}

	switch request.Method {
	case "PUT":
		pin_table.mutex.Lock()
		defer pin_table.mutex.Unlock()

		ticket, exists := pin_table.by_owner[subject]

		if !exists {
			return &http_status{400, "your PIN not issued"}
		}

		ticket.mutex.Lock()
		defer ticket.mutex.Unlock()

		if _, exists := ticket.pendings[person_id]; !exists {
			return &http_status{403, "unable to approve unsent request"}
		}

		delete(ticket.pendings, person_id)

		if _, err := db.Exec("INSERT INTO friendships (person_0, person_1) VALUES (?, ?)", subject, person_id); err != nil {
			return &http_status{500, err.Error()}
		}
	case "DELETE":
		if _, err := db.Exec("DELETE FROM friendships WHERE (person_0 = ? AND person_1 = ?) OR (person_0 = ? AND person_1 = ?)", subject, person_id, person_id, subject); err != nil {
			return &http_status{500, err.Error()}
		}
	default:
		return &http_status{405, "method not allowed"}
	}

	return write_friendships(subject, writer)
}

func endpoint_friendships_without_person_id(subject string, writer http.ResponseWriter, request *http.Request) *http_status {
	switch request.Method {
	case "GET":
		return write_friendships(subject, writer)
	case "POST":
		if request.Header.Get("Content-Type") != "text/plain" {
			return &http_status{415, "bad content type"}
		}

		buffer, err := ioutil.ReadAll(request.Body)

		if err != nil {
			return &http_status{400, "invalid content stream"}
		}

		request.Body.Close()

		pin, err := strconv.Atoi(string(buffer))

		if err != nil {
			return &http_status{400, "corrupt content format"}
		}

		pin_table.mutex.Lock()
		defer pin_table.mutex.Unlock()

		ticket, pin_exists := pin_table.by_pin[pin]

		if !pin_exists {
			return &http_status{400, "nonexistent PIN"}
		}

		ticket.mutex.Lock()
		defer ticket.mutex.Unlock()

		if subject == ticket.owner {
			return &http_status{400, "oneself cannot be his friend"}
		}

		if _, exists := ticket.pendings[subject]; exists {
			return &http_status{400, "you already sent request"}
		}

		ticket.channel <- subject
		ticket.pendings[subject] = true

		return &http_status{201, "wait for your request approved"}
	default:
		return &http_status{405, "method not allowed"}
	}
}

func write_friendships(subject string, writer http.ResponseWriter) *http_status {
	rows, err := db.Query("SELECT person_0, person_1 FROM friendships WHERE person_0 = ? OR person_1 = ?", subject, subject)

	if err != nil {
		return &http_status{500, err.Error()}
	}

	friends := make([]string, 0, 16)

	var person_0, person_1 string

	for rows.Next() {
		if err := rows.Scan(&person_0, &person_1); err != nil {
			return &http_status{500, err.Error()}
		}

		if person_0 == subject {
			friends = append(friends, person_1)
		} else {
			friends = append(friends, person_0)
		}
	}

	buffer, err := json.Marshal(friends)

	if err != nil {
		return &http_status{500, err.Error()}
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.Write(buffer)

	return nil
}
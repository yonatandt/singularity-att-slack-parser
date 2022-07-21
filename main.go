package main

import (
	"crypto/sha1"
	"fmt"
	"log"
	"net/http"

	"singularity-slack-reader/message"
)

const (
	LOG_FILE_PATH = "../logs/message.log"
	SlackHash     = "f9e720f6ca356a15d61a5895566c2560ca110e9a"
)

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello World!")
	// handle post request:
	if r.Method == "POST" {
		fmt.Println("POST request")
		// print the post data:
		r.ParseForm()
		fmt.Println("Form data: ", r.Form)

		token := r.FormValue("token")
		is_authenticated := checkPassword(token)
		if !is_authenticated {
			fmt.Println("Authentication failed!")
			// return 401 unauthorized
			// w.WriteHeader(http.StatusUnauthorized)
			// return
		}

		// get "text" param from post data:
		text := r.PostFormValue("text")
		fmt.Println("Message (before marshalling): ", text)

		// parse the post data to a Message struct:
		incomingMessage := message.ParseJSONToMessage(text)
		fmt.Println("Message (after marshalling): ", incomingMessage)

		var structuredMessage message.StructuredMessage

		switch incomingMessage.Type {
		case "code-change":
			structuredMessage = incomingMessage.ToDeploy()
		case "ff-change":
			structuredMessage = incomingMessage.ToFeatureFlagChange()
		case "postback-update":
			structuredMessage = incomingMessage.ToPostbackUpdate()
		default:
			fmt.Println("Unknown message type: ", incomingMessage.Type)
			return
		}

		fmt.Printf("FinalStructured: %+v\n", structuredMessage)
		// write the DeployStructured struct to a file:
		structuredMessage.Write_structured_json_to_file(LOG_FILE_PATH)
	}
}

func main() {
	http.HandleFunc("/att-ops", handler)
	log.Println("Server starting on port 8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}

func checkPassword(password string) bool {
	hsha1 := sha1.Sum([]byte(password))
	return fmt.Sprintf("%x", hsha1) == SlackHash
}

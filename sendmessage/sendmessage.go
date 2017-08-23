package main

import (
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"log"
	"net/http"
	"os"
)

var (
	sendGridKey string = os.Getenv("SENDGRID_KEY")
)

func WriteOutcome(w http.ResponseWriter, outType string, outMsg string) {
	w.Header().Set("Content-Type", "application/json")
	outJson := "{\"" + outType + "\": \"" + outMsg + "\"}"
	w.Write([]byte(outJson))
}

func SendEmail(w http.ResponseWriter, messageTxt string, fromAddr string, toAddr string) {

	from := mail.NewEmail("Helicon Message Gateway", fromAddr)
	subject := "Helicon Test message"
	to := mail.NewEmail("GP test user", toAddr)
	plainTextContent := messageTxt
	htmlContent := "<p>" + messageTxt + "</p>"
	message := mail.NewSingleEmail(from, subject, to, plainTextContent, htmlContent)
	client := sendgrid.NewSendClient(sendGridKey)
	// response, err := client.Send(message)
	_, err := client.Send(message)
	if err != nil {
		WriteOutcome(w, "error", "Problem sending email.")

	} else {
		WriteOutcome(w, "success", "Email successfully sent.")
	}

}

func serve(w http.ResponseWriter, r *http.Request) {

	switch r.Method {
	case "POST":
		msgtype := r.FormValue("msgtype")
		message := r.FormValue("message")
		address := r.FormValue("address")
		userid := r.FormValue("userid")

		if msgtype == "" || message == "" || address == "" || userid == "" {
			WriteOutcome(w, "error", "One or more of the required parameters (msgtype, message, address, userid) are missing.")
			return
		}
		if msgtype != "email" {
			WriteOutcome(w, "error", "Invalid msgtype parameter, currently needs to be 'email'.")
			return
		} else {

			SendEmail(w, message, userid+"@sendgrid.mariomenti.com", address)
		}
	default:
		WriteOutcome(w, "error", "Parameter sendmessage requires a POST request")
	}
}

func main() {
	http.HandleFunc("/", serve)
	err := http.ListenAndServe(":80", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

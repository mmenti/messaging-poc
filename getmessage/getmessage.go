package main

import (
	"bufio"
	"github.com/recapco/emailreplyparser"
	redis "gopkg.in/redis.v5"
	"io"
	"io/ioutil"
	"log"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
)

var (
	sendGridKey string = os.Getenv("SENDGRID_KEY")
	redisClient *redis.Client
)

func getBoundary(value string, contentType string) (string, *strings.Reader) {
	body := strings.NewReader(value)
	bodySplit := strings.Split(string(value), contentType)
	scanner := bufio.NewScanner(strings.NewReader(bodySplit[1]))
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
		break
	}
	boundary := lines[0][9:]
	return boundary, body
}

func init() {

	redisClient = redis.NewClient(&redis.Options{
		Addr:     "redis-master:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})

}

func serve(w http.ResponseWriter, r *http.Request) {

	mediaType, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil {
		log.Fatal(err)
	}
	if strings.HasPrefix(mediaType, "multipart/") {
		mr := multipart.NewReader(r.Body, params["boundary"])
		parsedEmail := make(map[string]string)
		emailHeader := make(map[string]string)
		binaryFiles := make(map[string][]byte)
		parsedRawEmail := make(map[string]string)
		rawFiles := make(map[string]string)
		for {
			p, err := mr.NextPart()
			// We have found an attachment with binary data
			if err == nil && p.FileName() != "" {
				contents, err := ioutil.ReadAll(p)
				if err != nil {
					log.Fatal(err)
				}
				binaryFiles[p.FileName()] = contents
			}
			if err == io.EOF {
				// We have finished parsing, do something with the parsed data
				msgText := ""
				msgSender := ""
				for key, value := range parsedEmail {
					if key == "text" {
						// try and remove the quoted reply
						reply, err := emailreplyparser.ParseReply(value)
						if err != nil {
							log.Fatal(err)
						}
						msgText = reply
					}
					if key == "to" {
						// extract email address between < and >
						tmp1 := strings.Split(value, "<")
						tmp2 := strings.Split(tmp1[1], ">")
						msgSender = tmp2[0]
					}
				}
				// write to redis
				err := redisClient.Set(msgSender, msgText, 0).Err()
				if err != nil {
					panic(err)
				}
				// fmt.Println(msgSender, msgText)

				// SendGrid needs a 200 OK response to stop POSTing
				w.WriteHeader(http.StatusOK)
				return
			}
			if err != nil {
				log.Fatal(err)
			}
			value, err := ioutil.ReadAll(p)
			if err != nil {
				log.Fatal(err)
			}
			header := p.Header.Get("Content-Disposition")
			if strings.Contains(header, "filename") != true {
				header = header[17 : len(header)-1]
				parsedEmail[header] = string(value)
			} else {
				header = header[11:]
				f := strings.Split(header, "=")
				parsedEmail[f[1][1:len(f[1])-11]] = f[2][1 : len(f[2])-1]
			}
			if header == "headers" {
				s := strings.Split(string(value), "\n")
				var a []string
				for _, v := range s {
					t := strings.Split(string(v), ": ")
					a = append(a, t...)
				}
				for i := 0; i < len(a)-1; i += 2 {
					emailHeader[a[i]] = a[i+1]
				}
			}
			// Since we have parsed the headers, we can delete the original
			delete(parsedEmail, "headers")

			// We have a raw message
			if header == "email" {
				boundary, body := getBoundary(string(value), "Content-Type: multipart/mixed; ")
				raw := multipart.NewReader(body, boundary)
				for {
					next, err := raw.NextPart()
					if err == io.EOF {
						// We have finished parsing
						break
					}
					value, err := ioutil.ReadAll(next)
					if err != nil {
						log.Fatal(err)
					}
					header := next.Header.Get("Content-Type")

					// Parse the headers
					if strings.Contains(header, "multipart/alternative") {
						boundary, body := getBoundary(string(value), "Content-Type: multipart/alternative; ")
						raw := multipart.NewReader(body, boundary)
						for {
							next, err := raw.NextPart()
							if err == io.EOF {
								// We have finished parsing
								break
							}
							value, err := ioutil.ReadAll(next)
							if err != nil {
								log.Fatal(err)
							}
							header := next.Header.Get("Content-Type")
							parsedRawEmail[header] = string(value)
						}
					} else {
						// It's a base64 encoded attachment
						rawFiles[header] = string(value)
					}
				}
			}
			// Since we've parsed this header, we can delete the original
			delete(parsedEmail, "email")
		}
	}

}

func main() {
	http.HandleFunc("/", serve)
	err := http.ListenAndServe(":80", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

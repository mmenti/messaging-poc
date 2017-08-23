package main

import (
	redis "gopkg.in/redis.v5"
	"log"
	"net/http"
)

var (
	redisClient *redis.Client
)

func init() {

	redisClient = redis.NewClient(&redis.Options{
		Addr:     "redis-master:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})

}

func serve(w http.ResponseWriter, r *http.Request) {

	userId := r.FormValue("userid")
	if userId == "" {
		w.Write([]byte("Missing userid parameter"))
		return
	}
	userId = userId + "@sendgrid.mariomenti.com"
	lastMsg := ""

	val, err := redisClient.Get(userId).Result()
	if err != nil {
		w.Write([]byte("Couldn't find userid = " + r.FormValue("userid")))
		return
	} else {
		lastMsg = val
	}
	w.Write([]byte(lastMsg))
}

func main() {
	http.HandleFunc("/", serve)
	err := http.ListenAndServe(":80", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

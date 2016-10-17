package main

import (
	"net/http"
	"math/rand"
	"time"
	"github.com/gorilla/mux"
	"github.com/garyburd/redigo/redis"
	"fmt"
	"encoding/json"
	"io/ioutil"
	"bytes"
)
const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
//var client redis.Client
var router mux.Router

var c redis.Conn
var err error

func main() {
	initRand()
	initRedis()
	router := mux.NewRouter()
	initRouting(router)
	http.ListenAndServe(":8000", router)
}

func initRand(){
	rand.Seed(time.Now().UnixNano())
}

func initRouting(router *mux.Router){
	router.HandleFunc("/", monikerHandler).
		Methods("GET")
	router.HandleFunc("/generateURL", urlSuffixGeneratorHandler).
		Methods("GET")
	router.HandleFunc("/validateURL/{url}", urlSuffixValidatorHandler).
		Methods("GET")
	router.HandleFunc("/getMoniker/{url}", getMonikerHandler).
		Methods("GET")
	router.HandleFunc("/createMoniker", createMonikerHandler).
		Methods("POST")

	router.HandleFunc("/{anypath}", monikerHandler).
		Methods("GET")
}

type redisConfig struct {
	Host string
	Password string
}

type urlValid struct {
	IsValid bool
	ValidUrl string
}

type socialData struct {
	SocialMediaSite string
	SocialHandle string
}

type moniker struct {
	Url string
	SocialData []socialData
}

func loadRedisConfig() (redisConfig, error){
	b, _ := ioutil.ReadFile("config.json")
	var config redisConfig

	if err != nil {
		config := redisConfig{}
		config.Host = "127.0.0.1";
		fmt.Println("Cannot find config file, defaulting to localHost.")
		return config, err
	}


	e := json.Unmarshal(b, &config)

	if e != nil {
		fmt.Println("Error reading redis config file.")
		fmt.Println("error:", e)
		return redisConfig{}, e;
	}

	return config, nil;
}

func createMonikerHandler(w http.ResponseWriter, r *http.Request){
	body, err := ioutil.ReadAll(r.Body)
	var m moniker
	if err != nil {
		panic(err)
	}

	err = json.Unmarshal(body, &m)
	if err != nil {
		panic(err)
	}

	isUrlValid := validURL(m.Url);

	fmt.Println(isUrlValid)

	if  isUrlValid {

		s, err := compressJsonToString(body)

		if err != nil {
			panic(err)
		}

		fmt.Println(s)
		//save data
		reserveURL(m.Url);
		c.Do("SET", "moniker:" + m.Url, s)
		w.Write([]byte("ok"))
	}else{
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Invalid or taken url"))
	}
}

func getMonikerHandler(w http.ResponseWriter, r *http.Request){
	vars := mux.Vars(r)
	url := vars["url"]
	var d = moniker{}

	  if( len(url) != 0){
		//get data
		g, err := redis.Bytes(c.Do("GET", "moniker:" + url));

		if err != nil && err != redis.ErrNil{
			panic(err)
		}else if(err == redis.ErrNil){
			w.WriteHeader(http.StatusNotFound)
			j, _ := json.MarshalIndent(d, "", "\t")
			w.Write([]byte(j))
		}else{
			err = json.Unmarshal(g, &d)
			if err != nil {
				panic(err)
			}

			j, _ := json.MarshalIndent(d, "", "\t")
			w.Write([]byte(j))
		}

	}else{
		  j, _ := json.MarshalIndent(d, "", "\t")
		  w.Write([]byte(j))
	}

}

func initRedis() {
	config, err := loadRedisConfig()
	if err != nil || config.Host == ""{
		c, err = redis.Dial("tcp", "127.0.0.1:6379")
		fmt.Println("Connected to localhost on port 6379.")
		if err != nil {
			panic(err)
		}
	}else if config.Host != ""{
		c, err = redis.Dial("tcp", config.Host + ":" + "6379")
		fmt.Printf("Connected to %s on port 6379.\n", config.Host)

		if err != nil {
			panic(err)
		}
		if config.Password != ""{
			c.Do("AUTH", config.Password)
		}

	}else{
		panic("Config file missing data.")
	}
}

func monikerHandler(w http.ResponseWriter, r *http.Request) {
	b, _ := ioutil.ReadFile("index.html")
	w.Write(b)
}

func urlSuffixGeneratorHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(GetFreeURL(2)))
}

func urlSuffixValidatorHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	var v urlValid
	s := vars["url"]
	if err != nil {
		panic(err)
	}
	fmt.Println(s)
	if(validURL(s)){
		v.ValidUrl = s
		v.IsValid = true
	}else {
		v.ValidUrl = GetFreeURL(2)
		v.IsValid = false
	}

	j, _ := json.MarshalIndent(v, "", "\t")
	w.Write([]byte(j))
}

func GetFreeURL(length int) string {
	urlLength := length
	var urlValid = false
	var result string
	var attempts int
	for !urlValid {
		result = GenerateURL(urlLength)
		urlValid = validURL(result)

		if attempts % 5 == 0 {
			urlLength++;
		}

		attempts++;
	}

	return result
}

func reserveURL(URL string){
	c.Do("SET", "resURL:" + URL, URL)
}

func GenerateURL(length int) string {
	b := make([]byte, length)

	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

func validURL(URL string) bool{
	if URL == "validateURL" || URL == "generateURL" || URL == "createMoniker" || URL == "" {return false}
	URLValidResult, err := redis.String(c.Do("GET", "resURL:" +URL))
	if err != nil && err != redis.ErrNil{
		panic(err)
	}
	return URLValidResult == ""
}

func compressJsonToString(d []byte) (string, error) {
	buffer :=  new(bytes.Buffer)
	err := json.Compact(buffer, d)
	return string(buffer.Bytes()), err
}

package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"log/syslog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/golang-jwt/jwt/v5"
)

func handler(w http.ResponseWriter, r *http.Request) {
	//write_n_log(w, fmt.Sprintf("%v\n\n", r)) // DEBUG
	//write_n_log(w, Sprintf("%v\n", r.Header))
	// NOTE: Expect: 100-continue does not work

	write_n_log(w, fmt.Sprintf("%s %s %s\r\n", r.Method, r.URL.Path, r.Proto))

	// We need to add the following headers because they are left out: http://golang.org/src/net/http/request.go
	write_n_log(w, fmt.Sprintf("Host: %s\r\n", r.Host))
	if r.TransferEncoding != nil {
		write_n_log(w, fmt.Sprintf("Transfer-Encoding: %s\r\n", strings.Join(r.TransferEncoding, ",")))
	}
	for key, value := range r.Header {
		write_n_log(w, fmt.Sprintf("%s: %s\r\n", key, strings.Join(value, ", ")))
	}

	// Read the body
	// defer r.Body.Close()
	// body, err := ioutil.ReadAll(r.Body)
	// if err != nil {
	// 	log.Printf("Error reading body: %s", r.URL.Path)
	// }
	// write_n_log(w, fmt.Sprintf("\r\n%s", body))

	// Now get the form data
	err := r.ParseForm()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	idtoken := r.PostFormValue("idtoken")
	write_n_log(w, fmt.Sprintf("idtoken: %s\r\n", idtoken))
	claims, _ := ValidateGoogleJWT(idtoken)
	write_n_log(w, fmt.Sprintf("claims \r\n%+v\r\n", claims))

}

var port string

func init() {
	// https://gobyexample.com/command-line-flags
	// http://golang.org/pkg/flag/
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "A simple HTTP server that will echo the incoming request as the body of the response\n\n")
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(2)
	}
	const (
		defaultPort = "9999"
		usage       = "port to listen on"
	)
	flag.StringVar(&port, "port", defaultPort, usage)
	flag.StringVar(&port, "p", defaultPort, usage+" (shorthand)")
}

func main() {
	flag.Parse()

	// Configure logger to write to the syslog. You could do this in init(), too.
	syslog_id := "http-echo"
	logwriter, e := syslog.New(syslog.LOG_NOTICE, syslog_id)
	if e == nil {
		log.SetOutput(logwriter)
	}

	fmt.Printf("Starting and listening on port: %s\n", port)
	fmt.Printf("Logging to syslog as: %s\n", syslog_id)
	log.Printf("Starting and listening on port: %s", port)

	// Wait for a SIGINT (perhaps triggered by user with CTRL-C)
	// Run cleanup when signal is received
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan,
		os.Interrupt,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	go func() {
		sig := <-signalChan
		fmt.Printf("\nStopping service, received: %s\n", sig)
		log.Printf("Stopping service, received: %s", sig)
		os.Exit(0)
	}()

	// Configure the handler and listen and serve  :)
	http.HandleFunc("/", handler)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func write_n_log(w http.ResponseWriter, s string) {
	fmt.Fprintf(w, "%s", s)
	fmt.Printf("%s", s) // log to the console
}

// GoogleClaims -
type GoogleClaims struct {
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	FirstName     string `json:"given_name"`
	LastName      string `json:"family_name"`
	Audience      string `json:"client_id"`
	ExpiresAt     string `json:"exp"`
	jwt.RegisteredClaims
}

func getGooglePublicKey(keyID string) (string, error) {
	resp, err := http.Get("https://www.googleapis.com/oauth2/v1/certs")
	if err != nil {
		return "", err
	}
	dat, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	myResp := map[string]string{}
	err = json.Unmarshal(dat, &myResp)
	if err != nil {
		return "", err
	}
	key, ok := myResp[keyID]
	if !ok {
		return "", errors.New("key not found")
	}
	return key, nil
}

func ValidateGoogleJWT(tokenString string) (GoogleClaims, error) {
	claimsStruct := GoogleClaims{}

	token, err := jwt.ParseWithClaims(
		tokenString,
		&claimsStruct,
		func(token *jwt.Token) (interface{}, error) {
			pem, err := getGooglePublicKey(fmt.Sprintf("%s", token.Header["kid"]))
			if err != nil {
				return nil, err
			}
			key, err := jwt.ParseRSAPublicKeyFromPEM([]byte(pem))
			if err != nil {
				return nil, err
			}
			return key, nil
		},
	)
	if err != nil {
		return GoogleClaims{}, err
	}

	claims, ok := token.Claims.(*GoogleClaims)
	if !ok {
		return GoogleClaims{}, errors.New("Invalid Google JWT")
	}

	if claims.Issuer != "accounts.google.com" && claims.Issuer != "https://accounts.google.com" {
		return GoogleClaims{}, errors.New("iss is invalid")
	}

	if claims.Audience != "237091407305-s09hultg1pldqfv1nu1de783i60g15s7.apps.googleusercontent.com" {
		return GoogleClaims{}, errors.New("aud is invalid")
	}

	//if claims.ExpiresAt < time.Now().UTC().Unix() {
	//		return GoogleClaims{}, errors.New("JWT is expired")
	//	}

	return *claims, nil
}

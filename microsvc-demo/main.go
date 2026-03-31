package microsvcdemo

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"
)

func main() {

	http.Handle("/", myHandler("Customer service"))

	var handlerFunc http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, r.URL.String())
	}

	http.HandleFunc("/url/", handlerFunc)

	// http.HandleFunc(
	// 	"/", func(w http.ResponseWriter, r *http.Request) {
	// 		fmt.Fprintln(w, "customer service")
	// 	})

	s := http.Server{
		Addr: ":3000",
	}

	go func() {
		log.Fatal(s.ListenAndServe())
	}()

	fmt.Println("Server started, press <Enter> to shutdown")
	fmt.Scanln()
	s.Shutdown(context.Background())
	fmt.Println("Server stopped")
}

type myHandler string

func (mh myHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("X-Powered-By", "energetic gophers")
	http.SetCookie(w, &http.Cookie{
		Name:    "session-id",
		Value:   "12345",
		Expires: time.Now().Add(24 * time.Hour * 365),
	})
	// r.Cookies()
	fmt.Fprintln(w, string(mh))
	fmt.Fprintln(w, r.Header)
}

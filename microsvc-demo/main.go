package microsvcdemo

import (
	"context"
	"fmt"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc(
		"/", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintln(w, "customer service")
		})

	s := http.Server{
		Addr: ":3000",
	}

	go func() {
		log.Fatal(http.ListenAndServeTLS("./cert.pem", "./key.pem"))
	}()

	fmt.Println("Server started, press <Enter> to shutdown")
	fmt.Scanln()
	s.Shutdown(context.Background())
	fmt.Println("Server stopped")
}

package microsvcdemo

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc(
		"/", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintln(w, "customer service")
		})

	log.Fatal(http.ListenAndServeTLS(":3000", "./cert.pem", "./key.pem", nil))

}

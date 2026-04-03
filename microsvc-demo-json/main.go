package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
)

func main() {

	http.HandleFunc("/customers", func(w http.ResponseWriter, r *http.Request) {
		customers, err := readCustomers()
		if err != nil {
			log.Print(err)
			w.WriteHeader((http.StatusInternalServerError))
			return
		}

		data, err := json.Marshal(customers)
		if err != nil {
			log.Print(err)
			w.WriteHeader((http.StatusInternalServerError))
			return
		}

		w.Header().Add("content-type", "application/json")
		w.Write(data)
	})

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

type Customer struct {
	ID        int
	FirstName string
	LastName  string
	Address   string
}

func readCustomers() ([]Customer, error) {
	f, err := os.Open("customers.csv")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	customers := make([]Customer, 0)
	csvReader := csv.NewReader(f)
	csvReader.Read() // throw away header
	for {
		fields, err := csvReader.Read()
		if err == io.EOF {
			return customers, nil
		}
		if err != nil {
			return nil, err
		}
		var c Customer
		id, err := strconv.Atoi(fields[0])
		if err != nil {
			continue
		}
		c.ID = id
		c.FirstName = fields[1]
		c.LastName = fields[2]
		c.Address = fields[3]
		customers = append(customers, c)
	}
}

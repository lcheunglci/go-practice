package main

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

func main() {

	http.HandleFunc("/customers/add", func(w http.ResponseWriter, r *http.Request) {
		var c Customer
		dec := json.NewDecoder(r.Body)
		err := dec.Decode((&c))
		if err != nil {
			w.WriteHeader((http.StatusBadRequest))
			return
		}
		log.Print(c)

	})

	go func() {
		time.Sleep(2 * time.Second)
		_, err := http.Post("http://localhost:3000/customers/add",
			"application/json", bytes.NewBuffer([]byte(`{
					"id": 999,
					"firstName": "Bob",
					"lastName":  "Doe",
					"address":   "123 Test St., Test, Test"
				}
			`)))
		if err != nil {
			log.Fatal(err)
		}
	}()

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
	ID        int    `json:"id"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Address   string `json:"address"`
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

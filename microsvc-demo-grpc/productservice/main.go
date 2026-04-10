package main

import (
	"fmt"
	"log"

	"google.golang.org/protobuf/proto"
)

type Product struct {
	ID         int
	Name       string
	USDPerUnit float64
	Unit       string
}

func main() {

	p := productpb.Product{
		Id:         int32(products[0].ID),
		Name:       products[0].Name,
		UsdPerUnit: products[0].USDPerUnit,
		Unit:       products[0].Unit,
	}

	data, err := proto.Marshal(&p)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(data))

	var p2 productpb.Product

	err = proto.Unmarshal(data, &p)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%+v", p2)
}

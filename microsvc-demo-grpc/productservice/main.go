package main

import (
	"context"
	"fmt"
	"log"
	"microsvc-demo-microsvc-grpc/productpb"
	"net"
	"time"

	"google.golang.org/grpc"
)

type Product struct {
	ID         int
	Name       string
	USDPerUnit float64
	Unit       string
}

func main() {

	go startGRPCServer()

	time.Sleep(1 * time.Second)

	callGRPCService()

	// p := productpb.Product{
	// 	Id:         int32(products[0].ID),
	// 	Name:       products[0].Name,
	// 	UsdPerUnit: products[0].USDPerUnit,
	// 	Unit:       products[0].Unit,
	// }

	// data, err := proto.Marshal(&p)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// fmt.Println(string(data))

	// var p2 productpb.Product

	// err = proto.Unmarshal(data, &p)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// fmt.Printf("%+v", p2)
}

type ProductService struct {
	productpb.UnimplementedProductServer
}

func (ps ProductService) GetProduct(ctx context.Context,
	req *productpb.GetProductRequest) (*productpb.GetProductReply, error) {

	for _, p := range products {
		if p.ID == int(req.ProductId) {
			return &productpb.GetProductReply{
					Product: &productpb.Product{
						Id:         int32(p.ID),
						Name:       p.Name,
						UsdPerUnit: p.USDPerUnit,
						Unit:       p.Unit,
					},
				},
				nil
		}
	}

	return nil, fmt.Errorf("product not found with ID: %v", req.ProductId)
}

func startGRPCServer() {
	lis, err := net.Listen("tcp", "localhost:4001")
	if err != nil {
		log.Fatal(err)
	}
	grpcServer := grpc.NewServer()
	productpb.RegisterProductServer(grpcServer, &ProductService{})
	log.Fatal(grpcServer.Serve(lis))
}

func callGRPCService() {
	opts := []grpc.DialOption{grpc.WithInsecure()}
	conn, err := grpc.Dial("localhost:4001", opts...)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	client := productpb.NewProductClient(conn)
	res, err := client.GetProduct(context.TODO(), &productdb.GetProductRequest{ProductId: 3})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%+v", res.Product)
}

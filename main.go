package main

import (
	"context"
	"fmt"
	"log"
	"net"

	config "github.com/kerosiinikone/go-docker-grpc/config"
	gen "github.com/kerosiinikone/go-docker-grpc/grpc"
	"google.golang.org/grpc"
)

// GetUser(context.Context, *Identifier) (*User, error)
// GetUserList(*EmptyParams, UserService_GetUserListServer) error

type UserSvcServer struct{
	gen.UnimplementedUserServiceServer
}

func (u *UserSvcServer) GetUser(context.Context, *gen.Identifier) (*gen.User, error) {
	// Context...
	return &gen.User{
		Id: 1,
		Name: "Foo",
	}, nil
}


func (u *UserSvcServer) GetUserList(p *gen.EmptyParams, svc gen.UserService_GetUserListServer) error {
	// Context...
	for i := 0; i < 10; i++ {
		if err := svc.Send(&gen.UserList{
			Users: []*gen.User{
				{
					Id: int32(i),
					Name: fmt.Sprintf("User %d", i),
				},
			},
		}); err != nil {
			return err
		}
	}
	return nil
}

func main() {
	cfg := config.New()
	usr := UserSvcServer{}

	// Server

	l, err := net.Listen("tcp", fmt.Sprintf("%s:%d", cfg.Addr, cfg.Port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	var opts []grpc.ServerOption

	grpcServer := grpc.NewServer(opts...)
	gen.RegisterUserServiceServer(grpcServer, &usr)
	grpcServer.Serve(l)

}
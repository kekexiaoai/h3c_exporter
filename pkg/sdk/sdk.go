package sdk

import (
	"context"
	"fmt"
	"log"
	"time"

	"google.golang.org/grpc"
	md "google.golang.org/grpc/metadata"

	h3c "github.com/kekexiaoai/h3c_exporter/pkg/grpc_service"
)

type GrpcSession struct {
	Client h3c.GrpcServiceClient
	Conn   *grpc.ClientConn
	Token  string
}

func NewClient(addr string, port uint, username string, password string) (*GrpcSession, error) {
	address := fmt.Sprintf("%s:%d", addr, port)

	//log.Printf("Server address: %v, UserName: %v, Password: %v\n", address, username, password)

	// Set up a connection to the server.
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		log.Printf("did not connect: %v", err)
		return nil, err
	}

	//create grpc_service client
	c := h3c.NewGrpcServiceClient(conn)

	//prepare context
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()

	var token string

	loginReply, err := c.Login(ctx, &h3c.LoginRequest{UserName: &username, Password: &password})
	if err != nil {
		log.Printf("could not login: %v", err)
		conn.Close()
		return nil, err
	}

	token = loginReply.GetTokenId()
	log.Printf("Token: %s", token)

	s := &GrpcSession{Client: c,
		Conn:  conn,
		Token: token}

	return s, nil
}

func (s *GrpcSession) Close() {
	var logoutReq = h3c.LogoutRequest{TokenId: &s.Token}
	ctx, cancel := CtxWithToken(s.Token, time.Second)
	defer cancel()
	s.Client.Logout(ctx, &logoutReq)
	s.Conn.Close()
	return
}

func CtxWithToken(tk string, timeout time.Duration) (context.Context, context.CancelFunc) {
	//Add token to meta data
	var mdata = md.Pairs("token_id", tk)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	var ctx_with_token = md.NewOutgoingContext(ctx, mdata)
	return ctx_with_token, cancel
}

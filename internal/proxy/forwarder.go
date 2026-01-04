package proxy

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func Forward(ctx context.Context, backendAddr, method string, req []byte) ([]byte, error) {
	conn, err := grpc.NewClient(
		backendAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()), // TODO: Use secure credentials in production
	)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	var resp []byte
	err = conn.Invoke(ctx, method, req, &resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

package proxy

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func Forward(ctx context.Context, addr, method string, data []byte) ([]byte, error) {
	conn, err := grpc.DialContext(ctx, addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	var out RawMessage
	err = conn.Invoke(ctx, method, &RawMessage{Data: data}, &out)
	if err != nil {
		return nil, err
	}

	return out.Data, nil
}

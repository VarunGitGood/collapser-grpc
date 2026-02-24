package proxy

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net"

	"github.com/VarunGitGood/collapser-grpc/internal/collapser"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Handler struct {
	backendAddr string
	collapser   *collapser.Collapser
}

func NewHandler(c *collapser.Collapser, backendAddr string) *Handler {
	return &Handler{
		backendAddr: backendAddr,
		collapser:   c,
	}
}

func (h *Handler) Serve(lis net.Listener) error {
	s := grpc.NewServer(grpc.UnknownServiceHandler(h.Handle))
	return s.Serve(lis)
}

func (h *Handler) Handle(srv interface{}, stream grpc.ServerStream) error {
	method, ok := grpc.MethodFromServerStream(stream)
	if !ok {
		return status.Errorf(codes.Internal, "cannot extract method")
	}

	in := &RawMessage{}
	if err := stream.RecvMsg(in); err != nil {
		if err == io.EOF {
			return status.Errorf(codes.InvalidArgument, "empty request")
		}
		return err
	}

	key := h.generateKey(method, in.Data)
	resp, err := h.collapser.Execute(stream.Context(), key, func(ctx context.Context) ([]byte, error) {
		return Forward(ctx, h.backendAddr, method, in.Data)
	})

	if err != nil {
		return err
	}

	return stream.SendMsg(&RawMessage{Data: resp})
}

func (h *Handler) generateKey(method string, data []byte) string {
	hash := sha256.Sum256(data)
	return method + ":" + hex.EncodeToString(hash[:])
}

type RawMessage struct {
	Data []byte
}

func (m *RawMessage) Reset()         { m.Data = nil }
func (m *RawMessage) String() string { return string(m.Data) }
func (m *RawMessage) ProtoMessage()  {}

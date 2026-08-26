package grpcx

import (
	"context"
	"fmt"
	app "github.com/acme/media-watermark-fingerprinting/internal/service/application"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/structpb"
	"time"
)

type TaskServiceServer interface {
	Health(context.Context, *emptypb.Empty) (*structpb.Struct, error)
	Submit(context.Context, *structpb.Struct) (*structpb.Struct, error)
	Get(context.Context, *structpb.Struct) (*structpb.Struct, error)
	Watch(*structpb.Struct, grpc.ServerStream) error
}
type Server struct{ app *app.App }

func New(a *app.App) *Server { return &Server{app: a} }
func (s *Server) Health(ctx context.Context, _ *emptypb.Empty) (*structpb.Struct, error) {
	return structpb.NewStruct(map[string]any{"status": "ok", "ready": s.app.Ready()})
}
func (s *Server) Submit(ctx context.Context, in *structpb.Struct) (*structpb.Struct, error) {
	fields := in.AsMap()
	kind, _ := fields["kind"].(string)
	assetID, _ := fields["asset_id"].(string)
	priority := number(fields["priority"])
	j, err := s.app.CreateJob(ctx, app.JobRequest{Kind: kind, AssetID: assetID, Priority: priority})
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	return jobStruct(j.ID, string(j.Status), j.Attempt, j.Error)
}
func (s *Server) Get(ctx context.Context, in *structpb.Struct) (*structpb.Struct, error) {
	id, _ := in.AsMap()["id"].(string)
	j, err := s.app.GetJob(ctx, id)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}
	return jobStruct(j.ID, string(j.Status), j.Attempt, j.Error)
}
func (s *Server) Watch(in *structpb.Struct, stream grpc.ServerStream) error {
	id, _ := in.AsMap()["id"].(string)
	last := ""
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		j, err := s.app.GetJob(stream.Context(), id)
		if err != nil {
			return status.Error(codes.NotFound, err.Error())
		}
		state := string(j.Status) + fmt.Sprint(j.Attempt)
		if state != last {
			out, _ := jobStruct(j.ID, string(j.Status), j.Attempt, j.Error)
			if err := stream.SendMsg(out); err != nil {
				return err
			}
			last = state
		}
		if j.Status == "succeeded" || j.Status == "failed" || j.Status == "dead" || j.Status == "canceled" {
			return nil
		}
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		case <-ticker.C:
		}
	}
}
func number(v any) int {
	if n, ok := v.(float64); ok {
		return int(n)
	}
	return 0
}
func jobStruct(id, state string, attempt int, message string) (*structpb.Struct, error) {
	return structpb.NewStruct(map[string]any{"id": id, "status": state, "attempt": attempt, "error": message})
}
func Auth(apiKey string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if info.FullMethod == "/media.v1.TaskService/Health" {
			return handler(ctx, req)
		}
		md, _ := metadata.FromIncomingContext(ctx)
		values := md.Get("x-api-key")
		if apiKey != "" && (len(values) == 0 || values[0] != apiKey) {
			return nil, status.Error(codes.Unauthenticated, "missing or invalid API key")
		}
		return handler(ctx, req)
	}
}
func Register(server *grpc.Server, service TaskServiceServer) {
	server.RegisterService(&grpc.ServiceDesc{ServiceName: "media.v1.TaskService", HandlerType: (*TaskServiceServer)(nil), Methods: []grpc.MethodDesc{{MethodName: "Health", Handler: healthHandler}, {MethodName: "Submit", Handler: submitHandler}, {MethodName: "Get", Handler: getHandler}}, Streams: []grpc.StreamDesc{{StreamName: "Watch", Handler: watchHandler, ServerStreams: true}}, Metadata: "api/media.proto"}, service)
}
func healthHandler(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	in := new(emptypb.Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(TaskServiceServer).Health(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/media.v1.TaskService/Health"}
	return interceptor(ctx, in, info, func(ctx context.Context, req any) (any, error) {
		return srv.(TaskServiceServer).Health(ctx, req.(*emptypb.Empty))
	})
}
func submitHandler(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	in := new(structpb.Struct)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(TaskServiceServer).Submit(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/media.v1.TaskService/Submit"}
	return interceptor(ctx, in, info, func(ctx context.Context, req any) (any, error) {
		return srv.(TaskServiceServer).Submit(ctx, req.(*structpb.Struct))
	})
}
func getHandler(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	in := new(structpb.Struct)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(TaskServiceServer).Get(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/media.v1.TaskService/Get"}
	return interceptor(ctx, in, info, func(ctx context.Context, req any) (any, error) {
		return srv.(TaskServiceServer).Get(ctx, req.(*structpb.Struct))
	})
}
func watchHandler(srv any, stream grpc.ServerStream) error {
	in := new(structpb.Struct)
	if err := stream.RecvMsg(in); err != nil {
		return err
	}
	return srv.(TaskServiceServer).Watch(in, stream)
}

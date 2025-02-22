package main

import (
	"context"
	"crypto/tls"
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"io/ioutil"
	"mime"
	"net"
	"net/http"
	"os"

	"github.com/gogo/gateway"
	grpc_validator "github.com/grpc-ecosystem/go-grpc-middleware/validator"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/grpclog"

	"github.com/gogo/grpc-example/insecure"
	pbExample "github.com/gogo/grpc-example/proto"
	"github.com/gogo/grpc-example/server"
)

var (
	gRPCPort    = flag.Int("grpc-port", 10000, "The gRPC server port")
	gatewayPort = flag.Int("gateway-port", 11000, "The gRPC-Gateway server port")
	nocheck     = flag.Bool("insecure", false, "don't complain about self-signed certs")
)

var log grpclog.LoggerV2

func init() {
	log = grpclog.NewLoggerV2(os.Stdout, ioutil.Discard, ioutil.Discard)
	grpclog.SetLoggerV2(log)
}

//go:embed third_party/OpenAPI/*
var content embed.FS

var public embed.FS

func fsHandler() http.Handler {
	sub, err := fs.Sub(public, "frontend/public")
	if err != nil {
		panic(err)
	}

	return http.FileServer(http.FS(sub))
}

// serveOpenAPI serves an OpenAPI UI on /openapi-ui/
// Adapted from https://github.com/philips/grpc-gateway-example/blob/a269bcb5931ca92be0ceae6130ac27ae89582ecc/cmd/serve.go#L63
func serveOpenAPI(mux *http.ServeMux) error {
	mime.AddExtensionType(".svg", "image/svg+xml")

	prefix := "/openapi-ui/"
	dirname := "third_party/OpenAPI"
	sub, err := fs.Sub(content, dirname)
	if err != nil {
		return fmt.Errorf("sub dir fail: %w", err)
	}
	dir := http.FS(sub)

	mux.Handle(prefix, http.StripPrefix(prefix, http.FileServer(dir)))
	return nil
}

func main() {
	flag.Parse()
	addr := fmt.Sprintf("localhost:%d", *gRPCPort)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalln("Failed to listen:", err)
	}
	s := grpc.NewServer(
		grpc.Creds(credentials.NewServerTLSFromCert(&insecure.Cert)),
		grpc.UnaryInterceptor(grpc_validator.UnaryServerInterceptor()),
		grpc.StreamInterceptor(grpc_validator.StreamServerInterceptor()),
	)
	pbExample.RegisterUserServiceServer(s, server.New())

	// Serve gRPC Server
	log.Info("Serving gRPC on https://", addr)
	go func() {
		log.Fatal(s.Serve(lis))
	}()

	// See https://github.com/grpc/grpc/blob/master/doc/naming.md
	// for gRPC naming standard information.
	dialAddr := fmt.Sprintf("passthrough://localhost/%s", addr)
	conn, err := grpc.DialContext(
		context.Background(),
		dialAddr,
		grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(insecure.CertPool, "")),
		grpc.WithBlock(),
	)
	if err != nil {
		log.Fatalln("Failed to dial server:", err)
	}

	mux := http.NewServeMux()

	jsonpb := &gateway.JSONPb{
		EmitDefaults: true,
		Indent:       "  ",
		OrigName:     true,
	}
	gwmux := runtime.NewServeMux(
		runtime.WithMarshalerOption(runtime.MIMEWildcard, jsonpb),
		// This is necessary to get error details properly
		// marshalled in unary requests.
		runtime.WithProtoErrorHandler(runtime.DefaultHTTPProtoErrorHandler),
	)
	err = pbExample.RegisterUserServiceHandler(context.Background(), gwmux, conn)
	if err != nil {
		log.Fatalln("Failed to register gateway:", err)
	}

	mux.Handle("/", gwmux)
	err = serveOpenAPI(mux)
	if err != nil {
		log.Fatalln("Failed to serve OpenAPI UI")
	}

	gatewayAddr := fmt.Sprintf("localhost:%d", *gatewayPort)
	log.Info("Serving gRPC-Gateway on https://", gatewayAddr)
	log.Info("Serving OpenAPI Documentation on https://", gatewayAddr, "/openapi-ui/")
	gwServer := http.Server{
		Addr: gatewayAddr,
		TLSConfig: &tls.Config{
			Certificates:       []tls.Certificate{insecure.Cert},
			InsecureSkipVerify: *nocheck,
		},
		Handler: mux,
	}
	log.Fatalln(gwServer.ListenAndServeTLS("", ""))
}

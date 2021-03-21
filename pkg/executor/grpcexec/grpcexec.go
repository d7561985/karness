package grpcexec

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/jhump/protoreflect/grpcreflect"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/fullstorydev/grpcurl"

	reflectpb "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
)

// To avoid confusion between program error codes and the gRPC resonse
// status codes 'Cancelled' and 'Unknown', 1 and 2 respectively,
// the response status codes emitted use an offest of 64
const statusCodeOffset = 64

const noVersion = "dev build <no version set>"

// Option dynamically change any internal requirements
type Option func(*Config)

// Config extend grpc client for enhance opt use Option which you should write inside this package ;)
type Config struct {
	connectTimeout float64
	keepaliveTime  float64
	maxMsgSz       int
	serverName     string
	authority      string
	cacert         string
	cert           string
	key            string
	userAgent      string
	plaintext      bool
	insecure       bool
	version        string

	verbosityLevel     int
	emitDefaults       bool
	allowUnknownFields bool
	format             grpcurl.Format

	formatError bool
}

type service struct {
	Config
}

func New(opt ...Option) *service {
	g := &Config{
		plaintext: true, insecure: true, version: "v1",
		format:             grpcurl.FormatJSON,
		formatError:        true,
		allowUnknownFields: true,
	}

	for _, opt := range opt {
		opt(g)
	}

	return &service{Config: *g}
}

// @symbol: {package}.{service}/{rpc}
// @request - json with request
func (g *service) Call(ctx context.Context, addr string, symbol Path, request string) (codes.Code, []byte, error) {
	cc, err := g.dial(ctx, addr)
	if err != nil {
		return 0, nil, fmt.Errorf("call error: %w", err)
	}

	md := grpcurl.MetadataFromHeaders(nil)
	refCtx := metadata.NewOutgoingContext(ctx, md)

	refClient := grpcreflect.NewClient(refCtx, reflectpb.NewServerReflectionClient(cc))
	reflSource := grpcurl.DescriptorSourceFromServer(ctx, refClient)

	//if fileSource != nil {
	//	descSource = compositeSource{reflSource, fileSource}
	//} else {
	//	descSource = reflSource
	//}
	descSource := reflSource

	// if not verbose output, then also include record delimiters
	// between each message, so output could potentially be piped
	// to another grpcurl process
	includeSeparators := g.verbosityLevel == 0
	options := grpcurl.FormatOptions{
		EmitJSONDefaultFields: g.emitDefaults,
		IncludeTextSeparator:  includeSeparators,
		AllowUnknownFields:    g.allowUnknownFields,
	}

	in := bytes.NewBufferString(request)
	rf, formatter, err := grpcurl.RequestParserAndFormatter(g.format, descSource, in, options)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to construct request parser and formatter for %q: %w", g.format, err)
	}

	buf := bytes.NewBuffer(nil)

	h := &grpcurl.DefaultEventHandler{
		Out:            buf,
		Formatter:      formatter,
		VerbosityLevel: g.verbosityLevel,
	}

	err = grpcurl.InvokeRPC(ctx, descSource, cc, symbol.String(), nil, h, rf.Next)

	if err != nil {
		if errStatus, ok := status.FromError(err); ok && g.formatError {
			h.Status = errStatus
		} else {
			return 0, nil, fmt.Errorf("error invoking method %q: %w", symbol, err)
		}
	}

	reqSuffix := ""
	respSuffix := ""
	reqCount := rf.NumRequests()

	if reqCount != 1 {
		reqSuffix = "s"
	}

	if h.NumResponses != 1 {
		respSuffix = "s"
	}

	if g.verbosityLevel > 0 {
		fmt.Printf("Sent %d request%s and received %d response%s\n", reqCount, reqSuffix, h.NumResponses, respSuffix)
	}

	if h.Status.Code() != codes.OK {
		if g.formatError {
			printFormattedStatus(buf, h.Status, formatter)
		} else {
			grpcurl.PrintStatus(buf, h.Status, formatter)
		}
	}

	return h.Status.Code(), buf.Bytes(), nil
}

func (g *Config) dial(ctx context.Context, addr string) (*grpc.ClientConn, error) {
	dialTime := 10 * time.Second
	if g.connectTimeout > 0 {
		dialTime = time.Duration(g.connectTimeout * float64(time.Second))
	}

	ctx, cancel := context.WithTimeout(ctx, dialTime)
	defer cancel()

	var opts []grpc.DialOption
	if g.keepaliveTime > 0 {
		timeout := time.Duration(g.keepaliveTime * float64(time.Second))
		opts = append(opts, grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:    timeout,
			Timeout: timeout,
		}))
	}

	if g.maxMsgSz > 0 {
		opts = append(opts, grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(g.maxMsgSz)))
	}

	var creds credentials.TransportCredentials
	if !g.plaintext {
		var err error

		creds, err = grpcurl.ClientTransportCredentials(g.insecure, g.cacert, g.cert, g.key)
		if err != nil {
			return nil, fmt.Errorf("failed to configure transport credentials: %w", err)
		}

		// can use either -servername or -authority; but not both
		if g.serverName != "" && g.authority != "" {
			if g.serverName == g.authority {
				log.Println("Both -servername and -authority are present; prefer only -authority")
			} else {
				return nil, fmt.Errorf("cannot specify different values for -servername and -authority")
			}
		}

		overrideName := g.serverName
		if overrideName == "" {
			overrideName = g.authority
		}

		if overrideName != "" {
			if err := creds.OverrideServerName(overrideName); err != nil {
				return nil, fmt.Errorf("failed to override MockServer name as %q: %w", overrideName, err)
			}
		}
	} else if g.authority != "" {
		opts = append(opts, grpc.WithAuthority(g.authority))
	}

	grpcurlUA := "grpcurl/" + g.version
	if g.version == noVersion {
		grpcurlUA = "grpcurl/dev-build (no version set)"
	}

	if g.userAgent != "" {
		grpcurlUA = g.userAgent + " " + grpcurlUA
	}

	opts = append(opts, grpc.WithUserAgent(grpcurlUA))

	network := "tcp"
	//if isUnixSocket != nil && isUnixSocket() {
	//	network = "unix"
	//}

	cc, err := grpcurl.BlockingDial(ctx, network, addr, creds, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to dial addr host %q: %w", addr, err)
	}

	return cc, nil
}

func printFormattedStatus(w io.Writer, stat *status.Status, formatter grpcurl.Formatter) {
	formattedStatus, err := formatter(stat.Proto())
	if err != nil {
		_, _ = fmt.Fprintf(w, "ERROR: %v", err.Error())
	}

	_, _ = fmt.Fprint(w, formattedStatus)
}

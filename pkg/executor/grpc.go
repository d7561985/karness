package executor

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
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

const no_version = "dev build <no version set>"

type GRPC struct {
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

func NewGRPC() *GRPC {
	return &GRPC{
		plaintext: true, insecure: true, version: "v1",
		format:      grpcurl.FormatJSON,
		formatError: true,
	}
}

// addr: sportsbook-settings-sv.odds-compiler.svc.cluster.local:9000
// symbol: egt.oddscompiler.sportsbooksettings.v3.public.InfoService/GetEventInfo
func (g *GRPC) Go(addr, symbol string, body string) error {
	ctx := context.Background()

	cc, err := g.dial(ctx, addr)
	if err != nil {
		return fmt.Errorf("call error: %w", err)
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

	in := bytes.NewBufferString(body)
	rf, formatter, err := grpcurl.RequestParserAndFormatter(g.format, descSource, in, options)
	if err != nil {
		return fmt.Errorf("failed to construct request parser and formatter for %q: %w", g.format, err)
	}

	h := &grpcurl.DefaultEventHandler{
		Out:            os.Stdout,
		Formatter:      formatter,
		VerbosityLevel: g.verbosityLevel,
	}

	err = grpcurl.InvokeRPC(ctx, descSource, cc, symbol, nil, h, rf.Next)
	if err != nil {
		if errStatus, ok := status.FromError(err); ok && g.formatError {
			h.Status = errStatus
		} else {
			return fmt.Errorf("error invoking method %q: %w", symbol, err)
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
			printFormattedStatus(os.Stderr, h.Status, formatter)
		} else {
			grpcurl.PrintStatus(os.Stderr, h.Status, formatter)
		}

		os.Exit(statusCodeOffset + int(h.Status.Code()))
	}

	return nil
}

func (g *GRPC) dial(ctx context.Context, addr string) (*grpc.ClientConn, error) {
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
				return nil, fmt.Errorf("failed to override server name as %q: %w", overrideName, err)
			}
		}
	} else if g.authority != "" {
		opts = append(opts, grpc.WithAuthority(g.authority))
	}

	grpcurlUA := "grpcurl/" + g.version
	if g.version == no_version {
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
		fmt.Fprintf(w, "ERROR: %v", err.Error())
	}

	fmt.Fprint(w, formattedStatus)
}

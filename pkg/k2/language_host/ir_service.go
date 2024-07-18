package language_host

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/klothoplatform/klotho/pkg/k2/language_host/go"
	"github.com/klothoplatform/klotho/pkg/k2/model"
	"github.com/klothoplatform/klotho/pkg/logging"
)

type LanguageHost struct {
	debugCfg DebugConfig
	langHost *exec.Cmd
	conn     *grpc.ClientConn
}

func (irs *LanguageHost) Start(ctx context.Context, debug DebugConfig) (err error) {
	log := logging.GetLogger(ctx).Sugar()

	irs.debugCfg = debug

	var addr *ServerAddress
	irs.langHost, addr, err = StartPythonClient(ctx, debug)
	if err != nil {
		return
	}
	log.Debug("Waiting for Python server to start")
	if debug.Enabled() {
		// Don't add a timeout in case there are breakpoints in the language host before an address is printed
		<-addr.HasAddr
	} else {
		select {
		case <-addr.HasAddr:
		case <-time.After(30 * time.Second):
			return errors.New("timeout waiting for Python server to start")
		}
	}

	irs.conn, err = grpc.NewClient(addr.Address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("failed to connect to Python server: %w", err)
	}

	return nil
}

func (irs *LanguageHost) NewClient() pb.KlothoServiceClient {
	return pb.NewKlothoServiceClient(irs.conn)
}

func (irs *LanguageHost) GetIR(ctx context.Context, req *pb.IRRequest) (*model.ApplicationEnvironment, error) {
	// Don't set the timeout if debugging, otherwise it may timeout while at a breakpoint or waiting to connect
	// to the debug server
	if !irs.debugCfg.Enabled() {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Second*10)
		defer cancel()
	}

	client := irs.NewClient()
	res, err := client.SendIR(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("error sending IR request: %w", err)
	}

	ir, err := model.ParseIRFile([]byte(res.GetYamlPayload()))
	if err != nil {
		return nil, fmt.Errorf("error parsing IR file: %w", err)
	}
	return ir, nil
}

func (irs *LanguageHost) Close() error {
	var errs []error
	if conn := irs.conn; conn != nil {
		errs = append(errs, conn.Close())
	}
	if p := irs.langHost.Process; p != nil {
		errs = append(errs, p.Kill())
	}
	return errors.Join(errs...)
}

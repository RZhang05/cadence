package server

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/dapperlabs/flow-go/sdk/templates"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/dapperlabs/flow-go/model/flow"
	"github.com/dapperlabs/flow-go/sdk/keys"

	"github.com/dapperlabs/flow-go/language/tools/language-server/protocol"
)

const (
	CommandSubmitTransaction = "cadence.server.submitTransaction"
	CommandExecuteScript     = "cadence.server.executeScript"
	CommandUpdateAccountCode = "cadence.server.updateAccountCode"
)

// CommandHandler represents the form of functions that handle commands
// submitted from the client using workspace/executeCommand.
type CommandHandler func(conn protocol.Conn, args ...interface{}) (interface{}, error)

// Registers the commands that the server is able to handle.
//
// The best reference I've found for how this works is:
// https://stackoverflow.com/questions/43328582/how-to-implement-quickfix-via-a-language-server
func (s Server) registerCommands(conn protocol.Conn) {
	// Send a message to the client indicating which commands we support
	registration := protocol.RegistrationParams{
		Registrations: []protocol.Registration{
			{
				ID:     "registerCommand",
				Method: "workspace/executeCommand",
				RegisterOptions: protocol.ExecuteCommandRegistrationOptions{
					ExecuteCommandOptions: protocol.ExecuteCommandOptions{
						Commands: []string{
							CommandSubmitTransaction,
							CommandExecuteScript,
							CommandUpdateAccountCode,
						},
					},
				},
			},
		},
	}

	// We have occasionally observed the client failing to recognize this
	// method if the request is sent too soon after the extension loads.
	// Retrying with a backoff avoids this problem.
	retryAfter := time.Millisecond * 100
	nRetries := 10
	for i := 0; i < nRetries; i++ {
		err := conn.RegisterCapability(&registration)
		if err == nil {
			break
		}
		conn.LogMessage(&protocol.LogMessageParams{
			Type: protocol.Warning,
			Message: fmt.Sprintf(
				"Failed to register command. Will retry %d more times... err: %s",
				nRetries-1-i, err.Error()),
		})
		time.Sleep(retryAfter)
		retryAfter *= 2
	}

	// Register each command handler function in the server
	s.commands[CommandSubmitTransaction] = s.submitTransaction
	s.commands[CommandExecuteScript] = s.executeScript
	s.commands[CommandUpdateAccountCode] = s.updateAccountCode
}

// submitTransaction handles submitting a transaction defined in the
// source document in VS Code.
//
// There should be exactly 2 arguments:
//   * the DocumentURI of the file to submit
//   * the address of the account to sign with
func (s *Server) submitTransaction(conn protocol.Conn, args ...interface{}) (interface{}, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("must have 2 args, got: %d", len(args))
	}
	uri, ok := args[0].(string)
	if !ok {
		return nil, errors.New("invalid uri argument")
	}
	doc, ok := s.documents[protocol.DocumentUri(uri)]
	if !ok {
		return nil, fmt.Errorf("could not find document for URI %s", uri)
	}
	addrHex, ok := args[1].(string)
	if !ok {
		return nil, errors.New("invalid uri argument")
	}
	addr := flow.HexToAddress(addrHex)

	return nil, s.sendTransaction(conn, []byte(doc.text), addr)
}

// executeScript handles executing a script defined in the source document in
// VS Code.
//
// There should be exactly 1 argument:
//   * the DocumentURI of the file to submit
func (s *Server) executeScript(conn protocol.Conn, args ...interface{}) (interface{}, error) {
	if len(args) != 1 {
		return nil, errors.New("missing argument")
	}
	uri, ok := args[0].(string)
	if !ok {
		return nil, errors.New("invalid uri argument")
	}
	doc, ok := s.documents[protocol.DocumentUri(uri)]
	if !ok {
		return nil, fmt.Errorf("could not find document for URI %s", uri)
	}

	script := []byte(doc.text)
	res, err := s.flowClient.ExecuteScript(context.Background(), script)
	if err == nil {
		conn.LogMessage(&protocol.LogMessageParams{
			Type:    protocol.Info,
			Message: fmt.Sprintf("Executed script with result: %v", res),
		})
		return res, nil
	}

	grpcErr, ok := status.FromError(err)
	if ok {
		if grpcErr.Code() == codes.Unavailable {
			// The emulator server isn't running
			conn.ShowMessage(&protocol.ShowMessageParams{
				Type:    protocol.Warning,
				Message: "The emulator server is unavailable. Please start the emulator (`cadence.runEmulator`) first.",
			})
			return nil, nil
		} else if grpcErr.Code() == codes.InvalidArgument {
			// The request was invalid
			conn.ShowMessage(&protocol.ShowMessageParams{
				Type:    protocol.Warning,
				Message: "The script could not be executed.",
			})
			conn.LogMessage(&protocol.LogMessageParams{
				Type:    protocol.Warning,
				Message: fmt.Sprintf("Failed to execute script: %s", grpcErr.Message()),
			})
			return nil, nil
		}
	}

	return nil, err
}

// createAccount creates a new account with the given
//func (s *Server) createAccount(conn protocol.Conn, args ...interface{}) (interface{}, error) {
//	conn.LogMessage(&protocol.LogMessageParams{
//		Type:    protocol.Log,
//		Message: fmt.Sprintf("create acct args: %v", args),
//	})
//
//	if len(args) != 0 {
//		return nil, errors.New("missing argument")
//	}
//
//	tx := flow.Transaction{
//		Script:             templates.CreateAccount(),
//		ReferenceBlockHash: nil,
//		Nonce:              0,
//		ComputeLimit:       0,
//		PayerAccount:       flow.Address{},
//		ScriptAccounts:     nil,
//		Signatures:         nil,
//		Status:             0,
//		Events:             nil,
//	}
//}

// updateAccountCode updates the configured account with the code of the given
// file.
//
// There should be exactly 2 arguments:
//   * the DocumentURI of the file to submit
//   * the address of the account to sign with
func (s *Server) updateAccountCode(conn protocol.Conn, args ...interface{}) (interface{}, error) {
	conn.LogMessage(&protocol.LogMessageParams{
		Type:    protocol.Log,
		Message: fmt.Sprintf("update acct code args: %v", args),
	})
	if len(args) != 2 {
		return nil, fmt.Errorf("must have 2 args, got: %d", len(args))
	}
	uri, ok := args[0].(string)
	if !ok {
		return nil, errors.New("invalid uri argument")
	}
	doc, ok := s.documents[protocol.DocumentUri(uri)]
	if !ok {
		return nil, fmt.Errorf("could not find document for URI %s", uri)
	}
	addrHex, ok := args[1].(string)
	if !ok {
		return nil, errors.New("invalid uri argument")
	}
	addr := flow.HexToAddress(addrHex)

	accountCode := []byte(doc.text)
	script := templates.UpdateAccountCode(accountCode)

	return nil, s.sendTransaction(conn, script, addr)
}

// sendTransaction sends a transaction with the given script, from the given
// address. Returns an error if we do not have the private key for the address.
//
// If an error occurs, attempts to show an appropriate message (either via logs
// or UI popups in the client).
func (s *Server) sendTransaction(conn protocol.Conn, script []byte, addr flow.Address) error {
	key, ok := s.accounts[addr]
	if !ok {
		return fmt.Errorf("cannot sign transaction for account with unknown address %s", addr)
	}

	tx := flow.Transaction{
		Script:         script,
		Nonce:          s.getNextNonce(),
		ComputeLimit:   10,
		PayerAccount:   addr,
		ScriptAccounts: []flow.Address{addr},
	}

	conn.LogMessage(&protocol.LogMessageParams{
		Type:    protocol.Info,
		Message: fmt.Sprintf("submitting transaction %d", tx.Nonce),
	})

	sig, err := keys.SignTransaction(tx, key)
	if err != nil {
		return err
	}
	tx.AddSignature(addr, sig)

	err = s.flowClient.SendTransaction(context.Background(), tx)
	if err == nil {
		conn.LogMessage(&protocol.LogMessageParams{
			Type:    protocol.Info,
			Message: fmt.Sprintf("Submitted transaction nonce=%d\thash=%s", tx.Nonce, tx.Hash().Hex()),
		})
		return nil
	}

	grpcErr, ok := status.FromError(err)
	if ok {
		if grpcErr.Code() == codes.Unavailable {
			// The emulator server isn't running
			conn.ShowMessage(&protocol.ShowMessageParams{
				Type:    protocol.Warning,
				Message: "The emulator server is unavailable. Please start the emulator (`cadence.runEmulator`) first.",
			})
			return nil
		} else if grpcErr.Code() == codes.InvalidArgument {
			// The request was invalid
			conn.ShowMessage(&protocol.ShowMessageParams{
				Type:    protocol.Warning,
				Message: "The transaction could not be submitted.",
			})
			conn.LogMessage(&protocol.LogMessageParams{
				Type:    protocol.Warning,
				Message: fmt.Sprintf("Failed to submit transaction: %s", grpcErr.Message()),
			})
			return nil
		}
	}

	return err
}

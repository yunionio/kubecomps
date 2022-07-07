package execute

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"syscall"
	"time"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/kubecomps/pkg/utils/ansibler/stdoutcallback"
	"yunion.io/x/kubecomps/pkg/utils/ansibler/stdoutcallback/results"
)

// DefaultExecute is a simple definition of an executor
type DefaultExecute struct {
	Write        io.Writer
	ResultsFunc  stdoutcallback.StdoutCallbackResultsFunc
	ShowDuration bool
}

const (
	// AnsiblePlaybookErrorCodeGeneralError
	AnsiblePlaybookErrorCodeGeneralError = 1
	// AnsiblePlaybookErrorCodeOneOrMoreHostFailed
	AnsiblePlaybookErrorCodeOneOrMoreHostFailed = 2
	// AnsiblePlaybookErrorCodeOneOrMoreHostUnreachable
	AnsiblePlaybookErrorCodeOneOrMoreHostUnreachable = 3
	// AnsiblePlaybookErrorCodeParserError
	AnsiblePlaybookErrorCodeParserError = 4
	// AnsiblePlaybookErrorCodeBadOrIncompleteOptions
	AnsiblePlaybookErrorCodeBadOrIncompleteOptions = 5
	// AnsiblePlaybookErrorCodeUserInterruptedExecution
	AnsiblePlaybookErrorCodeUserInterruptedExecution = 99
	// AnsiblePlaybookErrorCodeUnexpectedError
	AnsiblePlaybookErrorCodeUnexpectedError = 250

	// AnsiblePlaybookErrorMessageGeneralError
	AnsiblePlaybookErrorMessageGeneralError = "ansible-playbook error: general error"
	// AnsiblePlaybookErrorMessageOneOrMoreHostFailed
	AnsiblePlaybookErrorMessageOneOrMoreHostFailed = "ansible-playbook error: one or more host failed"
	// AnsiblePlaybookErrorMessageOneOrMoreHostUnreachable
	AnsiblePlaybookErrorMessageOneOrMoreHostUnreachable = "ansible-playbook error: one or more host unreachable"
	// AnsiblePlaybookErrorMessageParserError
	AnsiblePlaybookErrorMessageParserError = "ansible-playbook error: parser error"
	// AnsiblePlaybookErrorMessageBadOrIncompleteOptions
	AnsiblePlaybookErrorMessageBadOrIncompleteOptions = "ansible-playbook error: bad or incomplete options"
	// AnsiblePlaybookErrorMessageUserInterruptedExecution
	AnsiblePlaybookErrorMessageUserInterruptedExecution = "ansible-playbook error: user interrupted execution"
	// AnsiblePlaybookErrorMessageUnexpectedError
	AnsiblePlaybookErrorMessageUnexpectedError = "ansible-playbook error: unexpected error"
)

// Execute takes a command and args and runs it, streaming output to stdout
func (e *DefaultExecute) Execute(command string, args []string, prefix string) error {

	execDoneChan := make(chan int8)
	defer close(execDoneChan)
	execErrChan := make(chan error)
	defer close(execErrChan)

	if e.Write == nil {
		e.Write = os.Stdout
	}

	cmd := exec.Command(command, args...)

	cmdReader, err := cmd.StdoutPipe()
	defer cmdReader.Close()
	if err != nil {
		return errors.Wrap(err, "(DefaultExecute::Execute) Error creating stdout pipe")
	}

	timeInit := time.Now()
	err = cmd.Start()
	if err != nil {
		return errors.Wrap(err, "(DefaultExecute::Execute) Error starting command")
	}

	go func() {

		if e.ResultsFunc == nil {
			e.ResultsFunc = results.DefaultStdoutCallbackResults
		}
		err := e.ResultsFunc(prefix, cmdReader, e.Write)
		if err != nil {
			execErrChan <- err
			return
		}

		execDoneChan <- int8(0)
	}()

	select {
	case <-execDoneChan:
	case err := <-execErrChan:
		return errors.Wrap(err, "(DefaultExecute::Execute) Error managing results output")
	}

	err = cmd.Wait()
	if err != nil {
		errorMessage := string(err.(*exec.ExitError).Stderr)
		errorMessage = fmt.Sprintf("%s\n%s", errorMessage, err.Error())

		exitError, exists := err.(*exec.ExitError)
		if exists {

			ws := exitError.Sys().(syscall.WaitStatus)
			switch ws.ExitStatus() {
			case AnsiblePlaybookErrorCodeGeneralError:
				errorMessage = fmt.Sprintf("%s\n\n%s", AnsiblePlaybookErrorMessageGeneralError, errorMessage)
			case AnsiblePlaybookErrorCodeOneOrMoreHostFailed:
				errorMessage = fmt.Sprintf("%s\n\n%s", AnsiblePlaybookErrorMessageOneOrMoreHostFailed, errorMessage)
			case AnsiblePlaybookErrorCodeOneOrMoreHostUnreachable:
				errorMessage = fmt.Sprintf("%s\n\n%s", AnsiblePlaybookErrorMessageOneOrMoreHostUnreachable, errorMessage)
			case AnsiblePlaybookErrorCodeParserError:
				errorMessage = fmt.Sprintf("%s\n\n%s", AnsiblePlaybookErrorMessageParserError, errorMessage)
			case AnsiblePlaybookErrorCodeBadOrIncompleteOptions:
				errorMessage = fmt.Sprintf("%s\n\n%s", AnsiblePlaybookErrorMessageBadOrIncompleteOptions, errorMessage)
			case AnsiblePlaybookErrorCodeUserInterruptedExecution:
				errorMessage = fmt.Sprintf("%s\n\n%s", AnsiblePlaybookErrorMessageUserInterruptedExecution, errorMessage)
			case AnsiblePlaybookErrorCodeUnexpectedError:
				errorMessage = fmt.Sprintf("%s\n\n%s", AnsiblePlaybookErrorMessageUnexpectedError, errorMessage)
			}
		}
		return errors.Errorf("(DefaultExecute::Execute) Error during command execution: %s", errorMessage)
	}

	elapsedTime := time.Since(timeInit)

	if e.ShowDuration {
		fmt.Fprintf(e.Write, "Duration: %s\n", elapsedTime.String())
	}

	return nil
}

package skopeo

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
)

type CopyParams struct {
	SrcTLSVerify bool
	SrcUsername  string
	SrcPassword  string
	SrcPath      string
	TargetPath   string
}

type ISkopeo interface {
	Copy(params *CopyParams) error
}

type skopeo struct{}

func NewSkopeo() ISkopeo {
	return &skopeo{}
}

func (s *skopeo) GetCommand(args ...string) string {
	cmd := []string{"/usr/bin/skopeo"}
	cmd = append(cmd, args...)
	return strings.Join(cmd, " ")
}

func (s *skopeo) getCopyCommand(params *CopyParams) string {
	args := []string{"copy"}
	if params.SrcTLSVerify {
		args = append(args, "--src-tls-verify=true")
	} else {
		args = append(args, "--src-tls-verify=false")
	}
	if params.SrcUsername != "" {
		args = append(args, fmt.Sprintf("--src-username %s", params.SrcUsername))
	}
	if params.SrcPassword != "" {
		args = append(args, fmt.Sprintf("--src-password %s", params.SrcPassword))
	}
	args = append(args, fmt.Sprintf("docker://%s", params.SrcPath))
	args = append(args, fmt.Sprintf("docker-archive:%s", params.TargetPath))
	return s.GetCommand(args...)
}

func (s *skopeo) Copy(params *CopyParams) error {
	if params.SrcPath == "" {
		return errors.Error("src path is empty")
	}
	if params.TargetPath == "" {
		return errors.Error("target path is empty")
	}

	cmd := s.getCopyCommand(params)
	log.Debugf("execute cmd: %s", cmd)

	return s.executeCmd(cmd)
}

func (s *skopeo) executeCmd(cmd string) error {
	parts := strings.Split(cmd, " ")
	if len(parts) <= 2 {
		return errors.Errorf("invalid command: %q", cmd)
	}
	return s.execute(parts[0], parts[1:]...)
}

func (s *skopeo) execute(command string, args ...string) error {
	execDoneChan := make(chan int8)
	defer close(execDoneChan)

	log.Debugf("execute command: %s %v", command, args)

	cmd := exec.Command(command, args...)
	cmdReader, err := cmd.StdoutPipe()
	if err != nil {
		return errors.Wrapf(err, "create %s stdout pipe", command)
	}
	defer func() {
		err := cmdReader.Close()
		if err != nil {
			log.Errorf("%s %v command reader close: %v", command, err)
		}
	}()

	timeInit := time.Now()
	if err := cmd.Start(); err != nil {
		return errors.Wrapf(err, "start command %s", command)
	}

	go func() {
		w := os.Stdout
		scanner := bufio.NewScanner(cmdReader)
		for scanner.Scan() {
			fmt.Fprintf(w, scanner.Text())
		}
		execDoneChan <- int8(0)
	}()

	select {
	case <-execDoneChan:
	}

	if err := cmd.Wait(); err != nil {
		return errors.Wrapf(err, "wait %s command", command)
	}

	elapsedTime := time.Since(timeInit)
	log.Infof("%s command duration: %s", command, elapsedTime.String())
	return nil
}

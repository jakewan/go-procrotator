package childproc

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/jakewan/go-procrotator/logger"
	"github.com/jakewan/go-procrotator/runtimeconfig"
	"github.com/jakewan/go-procrotator/watchdirs"
)

type Dependencies interface {
	Logger() logger.Logger
}

type procState int

const (
	procStateNotStarted procState = iota
	procStateStarting
	procStateStarted
	procStateStopping
)

func (r procState) String() string {
	return [...]string{"NotStarted", "Starting", "Started", "Stopping"}[r]
}

func (r procState) EnumIndex() int {
	return int(r)
}

type state struct {
	locker             sync.Locker
	currentProcState   procState
	cmd                *exec.Cmd
	lastRestartAt      time.Time
	minRestartInterval time.Duration
}

func StartChildProcess(
	deps Dependencies,
	cfg runtimeconfig.Config,
	fileChangedChan <-chan watchdirs.FileChangedEvent,
	done chan<- bool,
) {
	defer func() {
		done <- true
	}()
	l := deps.Logger()
	st := state{
		locker:             &sync.Mutex{},
		minRestartInterval: 5 * time.Second,
	}

	func() {
		st.locker.Lock()
		defer st.locker.Unlock()
		if err := startChildProcess(l, cfg, &st); err != nil {
			l.Errorf(logger.ERROR, "Error starting server process: %s", err)
		}
	}()

	for range fileChangedChan {
		handleEvent(l, cfg, &st)
	}

	func() {
		st.locker.Lock()
		defer st.locker.Unlock()
		if err := stopChildProcess(l, cfg, &st); err != nil {
			l.Errorf(logger.ERROR, "Error stopping child process: %s", err)
		}
	}()
}

func handleEvent(
	l logger.Logger,
	cfg runtimeconfig.Config,
	st *state,
) {
	st.locker.Lock()
	defer st.locker.Unlock()
	elapsed := time.Since(st.lastRestartAt)
	if elapsed > st.minRestartInterval {
		if err := stopChildProcess(l, cfg, st); err != nil {
			l.Errorf(logger.DEBUG, "Error stopping current child process: %s", err)
		} else if err := startChildProcess(l, cfg, st); err != nil {
			l.Errorf(logger.DEBUG, "Error starting new child process: %s", err)
		}
		st.lastRestartAt = time.Now()
	} else {
		l.Errorf(
			logger.DEBUG,
			"Skipping restart. The minimum restart interval is %s and the last restart was %s ago.",
			st.minRestartInterval,
			elapsed,
		)
	}
}

// stopChildProcess stops the child process.
//
// The caller should manage locking and unlocking the mutex carried by
// the state object st.
func stopChildProcess(l logger.Logger, cfg runtimeconfig.Config, st *state) error {
	if st.currentProcState == procStateStarted {
		l.Errorf(logger.INFO, "Stopping child process")
		st.currentProcState = procStateStopping
		shutdownStaredAt := time.Now()
		if err := st.cmd.Process.Signal(cfg.QuitSignal()); err != nil {
			return fmt.Errorf("sending signal to child process: %w", err)
		} else if st.cmd == nil {
			return errors.New("child process should not be nil")
		} else if err := st.cmd.Wait(); err != nil {
			return fmt.Errorf("waiting for child process to finish: %w", err)
		} else {
			st.currentProcState = procStateNotStarted
			l.Errorf(logger.DEBUG, "Child process quit in %s", time.Since(shutdownStaredAt))
			return nil
		}
	} else {
		return fmt.Errorf("unexpected process state: %s", st.currentProcState.String())
	}
}

// startChildProcess starts the child process.
//
// The caller should manage locking and unlocking the mutex carried by
// the state object st.
func startChildProcess(
	l logger.Logger,
	cfg runtimeconfig.Config,
	st *state,
) error {
	if st.currentProcState != procStateNotStarted {
		return fmt.Errorf("invalid state before start: %s", st.currentProcState.String())
	}
	st.currentProcState = procStateStarting
	for _, c := range cfg.PreambleCommands() {
		if err := runPreambleCommand(c); err != nil {
			l.Errorf(logger.ERROR, "Error running preamble command: %s", err)
		}
	}
	if cmd, err := runServerCommand(cfg.ServerCommand()); err != nil {
		return err
	} else {
		st.cmd = cmd
		st.lastRestartAt = time.Now()
		st.currentProcState = procStateStarted
		return nil
	}
}

func runPreambleCommand(c string) error {
	commandParts := strings.Split(c, " ")
	name := commandParts[0]
	args := commandParts[1:]
	for i, a := range args {
		args[i] = os.ExpandEnv(a)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	proc := exec.CommandContext(ctx, name, args...)
	proc.Stdout = os.Stdout
	proc.Stderr = os.Stderr
	if err := proc.Start(); err != nil {
		return fmt.Errorf("starting preamble command: %w", err)
	}
	if err := proc.Wait(); err != nil {
		return fmt.Errorf("waiting for preamble command to complete: %w", err)
	}
	return nil
}

func runServerCommand(c string) (*exec.Cmd, error) {
	commandParts := strings.Split(c, " ")
	name := commandParts[0]
	args := commandParts[1:]
	for i, a := range args {
		args[i] = os.ExpandEnv(a)
	}
	proc := exec.Command(name, args...)
	proc.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}
	proc.Stdout = os.Stdout
	proc.Stderr = os.Stderr
	if err := proc.Start(); err != nil {
		return nil, fmt.Errorf("starting preamble command: %w", err)
	}
	return proc, nil
}

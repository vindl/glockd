package main

import (
	"flag"
	"fmt"
	"os"
	"regexp"
	"runtime"
	"syscall"
)

// Structure for requesting a lock with
type lock_request struct {
	lock   string
	action int
	reply  chan lock_reply
	client string
}

// Structure for a response generated during a lock request
type lock_reply struct {
	lock     string
	response string
}

type Configuration struct {
	Port     int
	Pid      string
	Verbose  bool
	Ws       int
	Registry bool
	Dump     bool
	Unix     string
}

var cfg *Configuration

func init() {
	cfg = new(Configuration)
	flag.IntVar(&cfg.Port, "port", 9999, "Listen on the following TCP ws. 0 Disables.")
	flag.IntVar(&cfg.Ws, "ws", 9998, "Listen on the following TCP Port. 0 Disables.")
	flag.StringVar(&cfg.Pid, "pidfile", "", "pidfile to use (required)")
	flag.StringVar(&cfg.Unix, "unix", "", "Filesystem path to the unix socket to listen on.  '' Disables.")
	flag.BoolVar(&cfg.Registry, "registry", true, "allow use of the registry.")
	flag.BoolVar(&cfg.Dump, "dump", true, "Allow use of the dump, d, and sd commands.")
	flag.BoolVar(&cfg.Verbose, "verbose", false, "be verbose about what's going on.")
	flag.Parse()
}

var rx_validate_remote_addr *regexp.Regexp

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	if cfg.Verbose == true {
		fmt.Printf("cfg_port:     %+v\n", cfg.Port)
		fmt.Printf("cfg_ws:       %+v\n", cfg.Ws)
		fmt.Printf("cfg_pidfile:  %+v\n", cfg.Pid)
		fmt.Printf("cfg_unix:     %+v\n", cfg.Unix)
		fmt.Printf("cfg_registry: %+v\n", cfg.Registry)
		fmt.Printf("cfg_dump:     %+v\n", cfg.Dump)
		fmt.Printf("cfg_verbose:  %+v\n", cfg.Verbose)
	}

	if cfg.Pid == "" {
		println("Please specify a pidfile")
		os.Exit(2)
	}

	rx_validate_remote_addr = regexp.MustCompile(":\\d+$")

	pidfile, err1 := os.OpenFile(cfg.Pid, os.O_CREATE|os.O_RDWR, 0666)
	err2 := syscall.Flock(int(pidfile.Fd()), syscall.LOCK_NB|syscall.LOCK_EX)
	if err1 != nil {
		fmt.Printf("Error opening pidfile: %s: %v\n", cfg.Pid, err1)
		os.Exit(3)
	}
	if err2 != nil {
		fmt.Printf("Error locking  pidfile: %s: %v\n", cfg.Pid, err2)
		os.Exit(4)
	}
	syscall.Ftruncate(int(pidfile.Fd()), 0)
	syscall.Write(int(pidfile.Fd()), []byte(fmt.Sprintf("%d", os.Getpid())))

	// Spawn a goroutine for stats
	go mind_stats()
	// Spawn a goroutine for locks
	go mind_locks()
	// Spawn a goroutine for shared locks
	go mind_shared_locks()
	// Spawn a goroutine for the websockets interface
	go mind_websockets()
	// Spawn a goroutine for accepting and handling incoming unix socket connections
	go mind_unix()
	// Spawn a goroutine for accepting and handling incoming tcp connections
	go mind_tcp()

	if cfg.Registry == true {
		// Spawn a goroutine for minding the user registry
		go mind_registry()
	}
	// Block indefinitely
	wait := make(chan bool)
	<-wait
}

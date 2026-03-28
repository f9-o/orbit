package main

import (
	"fmt"
	"os"
	"time"

	v1 "github.com/f9-o/orbit/api/v1"
	"github.com/f9-o/orbit/internal/core/state"
)

// ANSI Colors for the visual aesthetic
const (
	Reset   = "\033[0m"
	Red     = "\033[31m"
	Green   = "\033[32m"
	Yellow  = "\033[33m"
	Blue    = "\033[34m"
	Magenta = "\033[35m"
	Cyan    = "\033[36m"
	Bold    = "\033[1m"
)

var asciiBanner = `
             .             *
   *                   .
            _..._     *
          .'     '.      .
     .   /         \
    _ _.-|         |-._ _
     '-._\         /_.+'
         '.       .'
     *     '-...-'      *
           .     . 
                 *
`

func main() {
	// 1. Clear Screen and Print Banner
	fmt.Print("\033[H\033[2J") 
	fmt.Printf("%s%s", Bold, Cyan)
	fmt.Println(asciiBanner)
	fmt.Printf("%s\n", Reset)

	// Print Title Exactly as Requested
	fmt.Printf("  %s%s[ ORBIT LOCAL DEVSECOPS PLATFORM v0.2 ]%s\n\n", Bold, Magenta, Reset)

	time.Sleep(1 * time.Second)

	// 2. Simulate startup sequence with EXACT texts provided by the user
	step("Initializing Defense-in-Depth Security Engine...")
	step("Checking environment for [ORBIT_SECRET_KEY]...")
	
	// Force it to use the file-based generation for the demo
	os.Unsetenv("ORBIT_SECRET_KEY")
	time.Sleep(800 * time.Millisecond)

	warn("No environment key found. Procedurally generating Master Key...")
	time.Sleep(1 * time.Second)
	
	// 3. Trigger the actual AES engine we built
	dbPath := "orbit-demo.db"
	os.Remove(dbPath) // start fresh

	db, err := state.Open(dbPath)
	if err != nil {
		fmt.Printf("\n%s[FATAL] Failed to securely mount DB: %v%s\n", Red, err, Reset)
		return
	}
	defer db.Close()
	defer os.Remove(dbPath)
	
	success("Vault generated! Master Key securely stored in ~/.orbit/.master.key")
	step("Mounting BoltDB State Store...")
	time.Sleep(500 * time.Millisecond)

	// 4. Simulate a dummy Node write
	step("Injecting secure node data into data streams...")
	node := v1.NodeInfo{}
	node.Spec.Name = "worker-alpha"
	node.Status = "Provisioning"
	node.LastSeen = time.Now().UTC()
	db.PutNode(node)

	time.Sleep(600 * time.Millisecond)
	success("Payload encrypted with AES-256-GCM. 0 plaintext bytes on disk.")
	fmt.Println()
}

func step(msg string) {
	fmt.Printf("                   %s[~]%s %s\n", Cyan, Reset, msg)
	time.Sleep(700 * time.Millisecond)
}

func success(msg string) {
	fmt.Printf("                   %s[V]%s %s%s%s\n", Green, Reset, Bold, msg, Reset)
	time.Sleep(800 * time.Millisecond)
}

func warn(msg string) {
	fmt.Printf("                   %s[!]%s %s%s%s\n", Yellow, Reset, Bold, msg, Reset)
	time.Sleep(600 * time.Millisecond)
}

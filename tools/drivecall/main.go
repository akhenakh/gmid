// Command drivecall injects input into a running gmid instance and optionally
// captures a screenshot. It is used for interactive visual QA.
package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"go.hasen.dev/shirei/drive"
)

func main() {
	var (
		port  = flag.Int("port", 0, "drive port")
		shot  = flag.String("shot", "", "screenshot path")
		key   = flag.String("key", "", "key to press (space, right, f11, …)")
		mods  = flag.String("mods", "", "comma-separated modifiers for -key (ctrl, shift, alt)")
		click = flag.String("click", "", "click at x,y")
		move  = flag.String("move", "", "move the pointer to x,y without clicking")
		hover = flag.Bool("hovered", false, "print the accessibility node under the pointer")
		wait  = flag.Duration("wait", 300*time.Millisecond, "settle delay")
	)
	flag.Parse()
	if *port <= 0 {
		fmt.Fprintln(os.Stderr, "drivecall: -port is required")
		os.Exit(2)
	}
	if err := drive.Ping(*port); err != nil {
		fmt.Fprintln(os.Stderr, "drivecall: ping:", err)
		os.Exit(1)
	}
	if *move != "" {
		parts := strings.Split(*move, ",")
		if len(parts) != 2 {
			fmt.Fprintln(os.Stderr, "drivecall: -move wants x,y")
			os.Exit(2)
		}
		x, _ := strconv.ParseFloat(strings.TrimSpace(parts[0]), 32)
		y, _ := strconv.ParseFloat(strings.TrimSpace(parts[1]), 32)
		if err := drive.Move(*port, float32(x), float32(y)); err != nil {
			fmt.Fprintln(os.Stderr, "drivecall: move:", err)
			os.Exit(1)
		}
	}
	if *click != "" {
		parts := strings.Split(*click, ",")
		if len(parts) != 2 {
			fmt.Fprintln(os.Stderr, "drivecall: -click wants x,y")
			os.Exit(2)
		}
		x, _ := strconv.ParseFloat(strings.TrimSpace(parts[0]), 32)
		y, _ := strconv.ParseFloat(strings.TrimSpace(parts[1]), 32)
		if err := drive.Move(*port, float32(x), float32(y)); err != nil {
			fmt.Fprintln(os.Stderr, "drivecall: move:", err)
			os.Exit(1)
		}
		if err := drive.Down(*port); err != nil {
			fmt.Fprintln(os.Stderr, "drivecall: down:", err)
			os.Exit(1)
		}
		if err := drive.Up(*port); err != nil {
			fmt.Fprintln(os.Stderr, "drivecall: up:", err)
			os.Exit(1)
		}
	}
	if *key != "" {
		var list []string
		if *mods != "" {
			list = strings.Split(*mods, ",")
		}
		if err := drive.Key(*port, *key, list...); err != nil {
			fmt.Fprintln(os.Stderr, "drivecall: key:", err)
			os.Exit(1)
		}
	}
	time.Sleep(*wait)
	if *hover {
		resp, err := drive.Cmd(*port, "hovered")
		if err != nil {
			fmt.Fprintln(os.Stderr, "drivecall: hovered:", err)
			os.Exit(1)
		}
		fmt.Println(resp)
	}
	if *shot != "" {
		if err := drive.Screenshot(*port, *shot); err != nil {
			fmt.Fprintln(os.Stderr, "drivecall: screenshot:", err)
			os.Exit(1)
		}
		fmt.Println(*shot)
	}
}

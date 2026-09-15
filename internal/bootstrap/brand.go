package bootstrap

import (
	"fmt"
	"time"
)

const (
	brandBlue  = "\033[38;2;33;102;255m"
	brandCyan  = "\033[1;38;2;98;216;255m"
	brandReset = "\033[0m"
)

func printBrandLogo() {
	fmt.Printf(brandCyan + ` ______ ` + brandReset + "\n")
	fmt.Printf(brandCyan + `/\  __ \ ` + brandReset + "\n")
	fmt.Printf(brandCyan + `\ \ \/\_\ ` + brandReset + "\n")
	fmt.Printf(brandCyan + ` \ \___` + brandReset + brandBlue + `\_\ ` + brandReset + "\n")
	fmt.Printf(brandCyan + `  \/___` + brandReset + brandBlue + `/_/ ` + brandReset + "\n")
	fmt.Println("")
}

func (a *App) setupBrand() error {
	printBrandLogo()
	fmt.Println("")
	fmt.Println(brandBlue + "  QPub Server:" + brandReset + " " + brandCyan + "Open-Source Data Plane" + brandReset)
	fmt.Println("  • Channels (Pub/Sub)")
	fmt.Println("  • Queues")
	fmt.Println("  • Control / REST / WebSocket")
	currentYear := time.Now().UTC().Year()
	fmt.Printf(brandBlue + "  [+] https://qpub.io" + brandReset + "\n")
	fmt.Printf("  © 2019-%d Q.\n", currentYear)
	fmt.Println()
	return nil
}

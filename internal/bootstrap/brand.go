package bootstrap

import (
	"fmt"
	"time"
)

func (a *App) setupBrand() error {
	fmt.Printf("\033[1;38;2;98;216;255m" + ` ______ ` + "\033[0m\n")
	fmt.Printf("\033[1;38;2;98;216;255m" + `/\  __ \ ` + "\033[0m\n")
	fmt.Printf("\033[1;38;2;98;216;255m" + `\ \ \/\_\ ` + "\033[0m\n")
	fmt.Printf("\033[1;38;2;98;216;255m" + ` \ \___` + "\033[0m\033[38;2;33;102;255m" + `\_\ ` + "\033[0m\n")
	fmt.Printf("\033[1;38;2;98;216;255m" + `  \/___` + "\033[0m\033[38;2;33;102;255m" + `/_/ ` + "\033[0m\n")
	fmt.Println("")
	fmt.Println("\033[38;2;33;102;255m  QPub Server:\033[0m \033[1;38;2;98;216;255mOpen-Source Data Plane\033[0m")
	fmt.Println("  • Messaging (Pub/Sub)")
	fmt.Println("  • Product Queues")
	fmt.Println("  • Control / REST / WebSocket")
	currentYear := time.Now().UTC().Year()
	fmt.Printf("\033[38;2;33;102;255m  [+] https://qpub.io\033[0m\n")
	fmt.Printf("  © 2019-%d Q.\n", currentYear)
	fmt.Println()
	return nil
}

// Command kidsd is TECHO5 Kids: an Echo Spot running TECHO5 turned into a learning clock for a child's
// room. It is built from TECHO5's own hardware layer, and installed in place of TECHO5's daemon:
//
//	GOOS=linux GOARCH=arm GOARM=7 CGO_ENABLED=0 go build -tags spot -o kidsd ./cmd/kidsd
//
// See .github/README.md, the fork's front page.
package main

import "github.com/HuskerMinion/techo5/echod/internal/kids/cli"

func main() { cli.Execute() }

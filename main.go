package main

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type FlagsComponents struct {
	Link       string
	OutputFile string
	PathFile   string
	RateLimite int
	Exclude    string
	Reject     string
	Mirror     bool
	Background bool
	Convert    bool
}

func main() {
	args := os.Args[1:]
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage go run . link to download \n go run -O=filename link")
		return
	}
	components := FlagsComponents{}
	// fmt.Println(args, len(args))
	i := 0
	for i < len(args) {
		if strings.HasPrefix(args[i], "http") || strings.HasPrefix(args[i], "ftp") {
			components.Link = args[i]
		} else if strings.HasPrefix(args[i], "-O") && i <= len(args)-2 {
				checker, err := CatchOutputFile(args[i:i+2], &components)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				return
			}
			if checker {
				i += 2
				continue
			}
		} else if strings.HasPrefix(args[i], "-P") && i <= len(args)-2 {
			checker, err := CatchPath(args[i:i+2], &components)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				return
			}
			if checker {
				i += 2
				continue
			}
		}else if strings.HasPrefix(args[i], "--") && i <= len(args)-2 {
			checker, err := CatchRate(args[i:i+2], &components)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				return
			}
			if checker {
				i += 2
				continue
			}
		}
		i += 1
	}
	if components.Link == "" {
		fmt.Fprintln(os.Stderr, "you don't provide the program with link to download from it")
		return
	}
	fmt.Println(components)
}

func CatchOutputFile(args []string, comp *FlagsComponents) (bool, error) {
	if strings.Contains(args[0], "=") {
		sli := strings.Split(args[0], "=")
		if len(sli) != 2 || sli[1] == "" {
			return false, errors.New("missing output file while presence of the -O flag")
		} else {
			comp.OutputFile = sli[1]
			return false, nil
		}
	} else {
		if args[1] == "" || strings.HasPrefix(args[1], "http") {
			return false, errors.New("missing output file while presence of the -O flag")
		}
		comp.OutputFile = args[1]
		return true, nil
	}
}

func CatchPath(args []string, comp *FlagsComponents) (bool, error) {
	if strings.Contains(args[0], "=") {
		sli := strings.Split(args[0], "=")
		if len(sli) != 2 || sli[1] == "" {
			return false, errors.New("missing path while presence of the -P flag")
		} else {
			comp.PathFile = sli[1]
			return false, nil
		}
	} else {
		if args[1] == "" || strings.HasPrefix(args[1], "http") {
			return false, errors.New("missing path while presence of the -P flag")
		}
		comp.PathFile = args[1]
		return true, nil
	}
}

func CatchRate(args []string, comp *FlagsComponents) (bool, error) {
	// fmt.Println(args)
	if strings.Contains(args[0], "=") {
		sli := strings.Split(args[0], "=")
		// fmt.Println(sli)
		if len(sli) != 2 || sli[1] == "" {
			return false, errors.New("missing the rate while presence of the --rate-limit flag")
		} else {
			rate := sli[1]
			
			holder, err := strconv.Atoi(rate[:len(rate)-1])
			// fmt.Println(holder)
			if err!=nil {
				return false, errors.New("the rate isn't a valid number")
			}
			comp.RateLimite = holder
			return false, nil
		}
	} else {
		if args[1] == "" {
			return false, errors.New("missing the rate while presence of the --rate-limit flag")
		}
		rate := args[1]
		holder, err := strconv.Atoi(rate[:len(rate)-1])
		// fmt.Println(holder)
		if err!=nil {
			return false, errors.New("the rate isn't a valid number")
		}
		comp.RateLimite = holder
		return true, nil
	}
}
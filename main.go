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
	Exclude    []string
	Reject     []string
	isMirror   bool
	// Mirror     string
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
	var flags = []string{"-O", "-B", "-P", "--rate-limit", "--mirror", "-R", "--reject", "-X", "--exclude"}
	// fmt.Println(args, len(args))
	i := 0
	for i < len(args) {
		if strings.HasPrefix(args[i], "http") || strings.HasPrefix(args[i], "ftp") {
			components.Link = args[i]
		} else if strings.HasPrefix(args[i], "-O") && i <= len(args)-2 {
			checker, err := CatchOutputFile(args[i:i+2], &components, flags)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				return
			}
			if checker {
				i += 2
				continue
			}
		} else if strings.HasPrefix(args[i], "-P") && i <= len(args)-2 {
			checker, err := CatchPath(args[i:i+2], &components, flags)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				return
			}
			if checker {
				i += 2
				continue
			}
		} else if strings.HasPrefix(args[i], "--rate-limit") && i <= len(args)-2 {
			checker, err := CatchRate(args[i:i+2], &components, flags)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				return
			}
			if checker {
				i += 2
				continue
			}
		} else if strings.HasPrefix(args[i], "--mirror") {
			if !CheckValidFlag(args[i], flags) {
				fmt.Fprintln(os.Stderr, "invalid flag --mirror")
				return
			}
			components.isMirror = true
		} else if strings.HasPrefix(args[i], "-R") || strings.HasPrefix(args[i], "--reject") && i <= len(args)-2 {
			checker, err := CatchTheRejectedSuffix(args[i:i+2], &components, flags)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				return
			}
			if checker {
				i += 2
				continue
			}
		} else if strings.HasPrefix(args[i], "-X") || strings.HasPrefix(args[i], "--exclude") && i <= len(args)-2 {
			checker, err := CatchTheRExcludedFolders(args[i:i+2], &components, flags)
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
	if len(args) >= 2 && args[0] == "-B" {
	HandleBackgroundDownload(args[1])
	return
}
for i := 0; i < len(args); i++ {
	if strings.HasPrefix(args[i], "-i") {
		var filePath string
		if strings.Contains(args[i], "=") {
			parts := strings.SplitN(args[i], "=", 2)
			if len(parts) != 2 || parts[1] == "" {
				fmt.Fprintln(os.Stderr, "Invalid -i flag format")
				return
			}
			filePath = parts[1]
		} else if i+1 < len(args) {
			filePath = args[i+1]
		} else {
			fmt.Fprintln(os.Stderr, "Missing file after -i flag")
			return
		}
		HandleMultipleDownloads(filePath)
		return
	}
}
	if components.Link == "" {
		fmt.Fprintln(os.Stderr, "you don't provide the program with link to download from it")
		return
	}
	fmt.Println(components)
}

func CheckValidFlag(f string, flags []string) bool {
	for _,flag := range flags {
		if flag == f {
			return  true
		}
	}
	return false
}

func CatchOutputFile(args []string, comp *FlagsComponents, flags []string) (bool, error) {
	if strings.Contains(args[0], "=") {
		sli := strings.Split(args[0], "=")
		if !CheckValidFlag(sli[0], flags) {
			return false, errors.New("invalid flag -O")
		}
		if len(sli) != 2 || sli[1] == "" {
			return false, errors.New("missing output file while presence of the -O flag")
		} 	
		comp.OutputFile = sli[1]
		return false, nil
		
	} else {
		if !CheckValidFlag(args[0], flags) {
			return false, errors.New("invalid flag -O")
		}
		if args[1] == "" || strings.HasPrefix(args[1], "http") {
			return false, errors.New("missing output file while presence of the -O flag")
		}
		comp.OutputFile = args[1]
		return true, nil
	}
}

func CatchPath(args []string, comp *FlagsComponents, flags []string) (bool, error) {
	if strings.Contains(args[0], "=") {
		sli := strings.Split(args[0], "=")
		if !CheckValidFlag(sli[0], flags) {
			return false, errors.New("invalid flag -P")
		}
		if len(sli) != 2 || sli[1] == "" {
			return false, errors.New("missing path while presence of the -P flag")
		} 
		comp.PathFile = sli[1]
		return false, nil
		
	} else {
		if !CheckValidFlag(args[0], flags) {
			return false, errors.New("invalid flag -P")
		}
		if args[1] == "" || strings.HasPrefix(args[1], "http") {
			return false, errors.New("missing path while presence of the -P flag")
		}
		comp.PathFile = args[1]
		return true, nil
	}
}

func CatchRate(args []string, comp *FlagsComponents, flags []string) (bool, error) {
	// fmt.Println(args)
	if strings.Contains(args[0], "=") {
		sli := strings.Split(args[0], "=")
		if !CheckValidFlag(sli[0], flags) {
			return false, errors.New("invalid flag --rate-limit")
		}
		// fmt.Println(sli)
		if len(sli) != 2 || sli[1] == "" {
			return false, errors.New("missing the rate while presence of the --rate-limit flag")
		} 
		rate := sli[1]

		holder, err := strconv.Atoi(rate[:len(rate)-1])
		// fmt.Println(holder)
		if err != nil {
			return false, errors.New("the rate isn't a valid number")
		}
		comp.RateLimite = holder
		return false, nil
		
	} else {
		if !CheckValidFlag(args[0], flags) {
			return false, errors.New("invalid flag --rate-limit")
		}
		if args[1] == "" {
			return false, errors.New("missing the rate while presence of the --rate-limit flag")
		}
		rate := args[1]
		holder, err := strconv.Atoi(rate[:len(rate)-1])
		// fmt.Println(holder)
		if err != nil {
			return false, errors.New("the rate isn't a valid number")
		}
		comp.RateLimite = holder
		return true, nil
	}
}

func CatchTheRejectedSuffix(args []string, comp *FlagsComponents, flags []string) (bool, error) {
	if !comp.isMirror {
		return false, errors.New("flag --mirror is missing")
	}
	var rejectEx string
	var checker bool
	if strings.Contains(args[0], "=") {
		sli := strings.Split(args[0], "=")
		if !CheckValidFlag(sli[0], flags) {
			return false, errors.New("invalid flag -R || --reject")
		}
		if len(sli) != 2 || sli[1] == "" {
			return false, errors.New("missing rejected extentions while presence of the -R || --reject flag")
		}
		rejectEx = sli[1]
		checker = false
		
	} else {
		if !CheckValidFlag(args[0], flags) {
			return false, errors.New("invalid flag -R || --reject")
		}
		if args[1] == "" || strings.HasPrefix(args[1], "http") {
			return false, errors.New("missing rejected extentions while presence of the -R || --reject flag")
		}
		rejectEx = args[1]
		checker = true
	}
	comp.Reject = strings.Split(rejectEx, ",")
	return checker, nil
}

func CatchTheRExcludedFolders(args []string, comp *FlagsComponents, flags []string) (bool, error) {
	if !comp.isMirror {
		return false, errors.New("flag --mirror is missing")
	}
	var ExcludFolfers string
	var checker bool
	if strings.Contains(args[0], "=") {
		sli := strings.Split(args[0], "=")
		if !CheckValidFlag(sli[0], flags) {
			return false, errors.New("invalid flag -X || --exclud")
		}
		if len(sli) != 2 || sli[1] == "" {
			return false, errors.New("missing rejected extentions while presence of the -R || --reject flag")
		} else {
			ExcludFolfers = sli[1]
			checker = false
		}
	} else {
		if !CheckValidFlag(args[0], flags) {
			return false, errors.New("invalid flag -X || --exclud")
		}
		if args[1] == "" || strings.HasPrefix(args[1], "http") {
			return false, errors.New("missing rejected extentions while presence of the -R || --reject flag")
		}
		ExcludFolfers = args[1]
		checker = true
	}
	comp.Exclude = strings.Split(ExcludFolfers, ",")
	return checker, nil
}

package main

import (
	"errors"
	"fmt"
	"log"
	"slices"
	"strconv"
	"strings"
)

func CheckValidFlag(f string, flags []string) bool {
	return slices.Contains(flags, f)
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

func CatchInput(args []string, comp *FlagsComponents, flags []string) (bool, error) {
	if strings.Contains(args[0], "=") {
		sli := strings.Split(args[0], "=")
		if !CheckValidFlag(sli[0], flags) {
			return false, errors.New("invalid flag -i")
		}
		if len(sli) != 2 || sli[1] == "" {
			return false, errors.New("missing input file while presence of the -i flag")
		}
		comp.InputFile = sli[1]
		return false, nil

	} else {
		if !CheckValidFlag(args[0], flags) {
			return false, errors.New("invalid flag -i")
		}
		if args[1] == "" || strings.HasPrefix(args[1], "http") {
			return false, errors.New("missing input file while presence of the -i flag")
		}
		comp.InputFile = args[1]
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
		fmt.Println("haaaaaaaani")
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
		comp.RateLimite = sli[1]

		return false, nil

	} else {
		if !CheckValidFlag(args[0], flags) {
			return false, errors.New("invalid flag --rate-limit")
		}
		if args[1] == "" {
			return false, errors.New("missing the rate while presence of the --rate-limit flag")
		}
		comp.RateLimite = args[1]
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

func (c *FlagsComponents) Validate() error {
	// Check for conflicting flags
	if c.InputFile != "" && len(c.Links) != 0 {
		return fmt.Errorf("cannot use both -i (input file) and direct URL")
	}

	if c.OutputFile != "" && c.InputFile != "" {
		return fmt.Errorf("cannot use -O (output file) with -i (batch download)")
	}

	if c.isMirror && c.OutputFile != "" {
		return fmt.Errorf("cannot use -O (output file) with --mirror")
	}

	// Mirror-specific validations
	if (len(c.Reject) > 0 || len(c.Exclude) > 0 || c.Convert) && !c.isMirror {
		return fmt.Errorf("-R, -X, and --convert-links can only be used with --mirror")
	}

	// Require either URL or input file
	if len(c.Links) == 0 && c.InputFile == "" {
		return fmt.Errorf("must provide either URL or -i input file")
	}

	return nil
}

func logOrPrint(logger *log.Logger, background bool, message string) {
	if background && logger != nil {
		logger.Println(message)
	} else {
		fmt.Println(message)
	}
}

func parseRateLimit(rateLimitStr string) (int64, error) {
	if rateLimitStr == "" {
		return 0, nil
	}

	// Remove whitespace and convert to lowercase
	rateLimitStr = strings.ToLower(strings.TrimSpace(rateLimitStr))

	// Default multiplier (bytes)
	var multiplier int64 = 1

	// Check for suffix and remove it
	if strings.HasSuffix(rateLimitStr, "k") {
		multiplier = 1024
		rateLimitStr = rateLimitStr[:len(rateLimitStr)-1]
	} else if strings.HasSuffix(rateLimitStr, "m") {
		multiplier = 1024 * 1024
		rateLimitStr = rateLimitStr[:len(rateLimitStr)-1]
	}

	// Parse the numeric part
	value, err := strconv.ParseInt(rateLimitStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid rate limit format: %s", rateLimitStr)
	}

	return value * multiplier, nil
}


func formatSpeed(speedMBps float64) string {
	// Cap extremely high speeds to avoid scientific notation
	if speedMBps > 999.99 {
		speedMBps = 999.99
	}
	if speedMBps < 0.001 {
		speedMBps = 0.001
	}

	// Fixed logic: show KB/s when speed is LESS than 1 MB/s
	if speedMBps < 1 {
		return fmt.Sprintf("%.0f KB/s", speedMBps*1024)
	}
	return fmt.Sprintf("%.2f MB/s", speedMBps)
}

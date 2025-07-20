package tools

import (
	"log"
	"strconv"
)

func StrToInt(stringID string) (int64, bool) {
	intID, err := strconv.ParseInt(stringID, 10, 64)
	if err != nil {
		log.Printf("StrToInt: cannot convert %q to int64: %v", stringID, err)
		return 0, false
	}
	return intID, true
}

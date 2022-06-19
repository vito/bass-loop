package present

import (
	"fmt"
	"time"
)

func Duration(dt time.Duration) string {
	prec := 1
	sec := dt.Seconds()
	if sec < 10 {
		prec = 2
	} else if sec < 100 {
		prec = 1
	}

	return fmt.Sprintf("%.[2]*[1]fs", sec, prec)
}

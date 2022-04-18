package logging

import (
	"fmt"
	"time"
)

type Writer struct {
	Loc *time.Location
}

func (writer Writer) Write(b []byte) (n int, err error) {
	return fmt.Printf("%s %s", time.Now().In(writer.Loc).Format("2006-01-02 15:04:05"), b)
}

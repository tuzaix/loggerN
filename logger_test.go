package loggerN_test

import (
	"fmt"
	. "loggerN"
	"testing"
)

func TestLogger(t *testing.T) {
	logger := NewDefaultLogger()
	logger.Info(fmt.Sprintf("xxxxxxxxxxx"))
}

package bongoz

import (
	. "gopkg.in/check.v1"

	"log"
	// "net/url"
	"testing"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type TestSuite struct{}

var _ = Suite(&TestSuite{})

type NullWriter int

func (NullWriter) Write([]byte) (int, error) { return 0, nil }

func (s *TestSuite) SetUpTest(c *C) {

	if !testing.Verbose() {
		log.SetOutput(new(NullWriter))
	}

}

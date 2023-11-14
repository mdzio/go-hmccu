package xmlrpc

import (
	"time"

	"github.com/mdzio/go-lib/conc"
)

type RetryingCaller struct {
	// Function that is called multiple times if it returns an error.
	Caller Caller

	// Number of retries. 0 disables retries.
	RetryCount int

	// Delay between retries.
	RetryDelay time.Duration

	// The repeated calls can be cancelled with this context.
	Context conc.Context
}

func (c *RetryingCaller) Call(method string, params Values) (*Value, error) {
	// retry counter
	rcnt := 0
	for {
		// try a call
		value, err := c.Caller.Call(method, params)
		// on success, return value
		if err == nil {
			return value, nil
		}
		// give up when the retries have been used up
		rcnt++
		if rcnt > c.RetryCount {
			return nil, err
		}
		clnLog.Debugf("Call of method %s failed, retry in %s: %v", method, c.RetryDelay, err)
		// wait before the next call
		errc := c.Context.Sleep(c.RetryDelay)
		if errc != nil {
			// return last error
			return nil, err
		}
	}
}

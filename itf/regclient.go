package itf

import (
	"time"
)

const (
	// delay before registration
	startupDelay = 1 * time.Second
	// if no callback arrives within this time period, a ping is triggered
	callbackTimeout = 5 * time.Minute
	// if no pong arrives within this time period, a reregistration is triggered
	pingTimeout = 5 * time.Second
)

// RegisteredClient provides access to a CCU interface process. The registration state is
// monitored and reestablished on time out.
type RegisteredClient struct {
	*DeviceLayerClient
	RegistrationURL string
	RegistrationID  string
	ReGaHssID       string

	stopRequest chan struct{}
	stopped     chan struct{}
	callback    chan struct{}
	timer       *time.Timer
}

// Setup initializes the RegisteredClient.
func (i *RegisteredClient) Setup() {
	// setup
	i.stopRequest = make(chan struct{})
	i.stopped = make(chan struct{})
	// use buffered channel to hold one callback notification
	i.callback = make(chan struct{}, 1)
}

// Start registers at the CCU interface process and starts monitoring.
func (i *RegisteredClient) Start() {
	go func() {
		dclnLog.Debug("Starting interface ", i.ReGaHssID)

		// defer clean up
		defer func() {
			// free timer
			if !i.timer.Stop() {
				<-i.timer.C
			}
			dclnLog.Trace("Interface stopped: ", i.ReGaHssID)
			i.stopped <- struct{}{}
		}()

		// startup delay
		i.timer = time.NewTimer(startupDelay)
		for q := false; !q; {
			select {
			case <-i.stopRequest:
				return
			case <-i.callback:
				// ignore callbacks
			case <-i.timer.C:
				q = true
			}
		}

		// register
		i.register()
		// unregister on shut down
		defer i.unregister()
		i.timer.Reset(callbackTimeout)

		// re-registration loop
		for {
			// wait for time out
			for q := false; !q; {
				select {
				case <-i.stopRequest:
					return
				case <-i.callback:
					i.timer.Reset(callbackTimeout)
				case <-i.timer.C:
					q = true
				}
			}

			// ping
			ok, err := i.Ping(i.RegistrationID + "-Ping")
			if err != nil {
				dclnLog.Warning(err)
			} else if !ok {
				dclnLog.Warning("Ping returned a failure")
			}
			i.timer.Reset(pingTimeout)

			// wait for time out or callback
			select {
			case <-i.stopRequest:
				return
			case <-i.callback:
				// ping received
			case <-i.timer.C:
				// register again, if ping timed out
				dclnLog.Errorf("CCU interface %s timed out", i.ReGaHssID)
				i.register()
			}
			i.timer.Reset(callbackTimeout)
		}
	}()
}

// Stop stops the registration and monitoring.
func (i *RegisteredClient) Stop() {
	i.stopRequest <- struct{}{}
	<-i.stopped
}

// CallbackReceived must be called, when a callback from the CCU is received.
// The call is always non-blocking. Startup must be called first.
func (i *RegisteredClient) CallbackReceived() {
	// try to send
	select {
	case i.callback <- struct{}{}:
	default:
		// a full channel is ok
	}
}

func (i *RegisteredClient) register() {
	// register for callbacks (events, ...)
	if err := i.Init(i.RegistrationURL, i.RegistrationID); err != nil {
		dclnLog.Warning(err)
	}
}

func (i *RegisteredClient) unregister() {
	// stop callbacks
	if err := i.Deinit(i.RegistrationURL); err != nil {
		dclnLog.Warning(err)
	}
}

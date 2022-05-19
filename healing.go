package main

import (
	"fmt"
	"log"
	"os"
	"time"
)

type startGoroutineFn func(done <-chan interface{}, pulseInterval time.Duration) (heartBeat <-chan interface{})

func newStewed(timeout time.Duration, startGoroutine startGoroutineFn) startGoroutineFn {
	return func(done <-chan interface{}, pulseInterval time.Duration) <-chan interface{} {
		heartbeat := make(chan interface{})
		go func() {
			defer close(heartbeat)
			var wardDone chan interface{}
			var wardHeartBeat <-chan interface{}
			startWard := func() {
				wardDone = make(chan interface{})
				wardHeartBeat = startGoroutine(or(wardDone, done), timeout/2)
			}
			startWard()
			pulse := time.Tick(pulseInterval)
		monitorLoop:
			for {
				timeOutSignal := time.After(timeout)
				for {
					select {
					case <-pulse:
						fmt.Println("beating")
						select {
						case heartbeat <- struct{}{}:
						default:
						}
					case <-wardHeartBeat:
						continue monitorLoop
					case <-timeOutSignal:
						log.Println("stewed: ward unhealthy;restarting")
						close(wardDone)
						startWard()
						continue monitorLoop
					case <-done:
						return
					}
				}
			}
		}()
		return heartbeat
	}
}

func or(wardDone chan interface{}, done <-chan interface{}) <-chan interface{} {
	select {
	case <-wardDone:
		return wardDone
	case <-done:
		return done
	}
}

func main() {
	log.SetOutput(os.Stdout)
	log.SetFlags(log.Ltime | log.LUTC)
	doWork := func(done <-chan interface{}, _ time.Duration) <-chan interface{} {
		log.Println("ward: Hello, I'm irresponsible!")
		go func() {
			<-done
			log.Println("ward: I am halting.")
		}()
		return nil
	}
	doWorkWithSteward := newStewed(20*time.Second, doWork)
	done := make(chan interface{})
	time.AfterFunc(9*time.Second, func() {
		log.Println("main: halting steward and ward.")
		close(done)
	})
	for range doWorkWithSteward(done, 10*time.Second) {
	}
	log.Println("Done")
}

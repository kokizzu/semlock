
# SemLock

a simple semaphore lock, difference between built-in semaphore, the limit can be updated dynamically.
example use case to dynamically limit the number of concurrent tasks, 
eg. when database overload we want to decrease number of query hitting database, 
  or when CPU/RAM/Bandwidth used by something else, we may want to decrease the number of worker

## Usage

```go
package main

import "github.com/kokizzu/semlock"

func _() {
	// maximum 10 concurrent tasks, 100ms delay before try acquire lock again
    s := semlock.NewMinSemaphoreLock(10, 100 * time.Millisecond)
    
	for _ = range 10 { // maximum 10 worker
        go func() {
            for bla := range someChannel {
                // block until acquire lock
                s.BlockUntilAllowed()
                
                // do expensive query or process
				expensiveQueryOrCalculation(bla)
                
                // release lock
                s.ReleaseActive()
            }
        }
	}
	
	for {
		select {
		    case <- cpuOverloaded, <- databaseOverloadedSignal:
				s.DecAllowed()
			case <- cpuLessHalfIdle, <- databaseLessHalfIdleSignal:
				s.IncAllowed()
			case <- ctx.Done():
				close(someChannel)
				return
        }
    }
}
```

# SemLock

a simple semaphore lock, difference between built-in semaphore, the limit can be updated dynamically.
example use case to dynamically limit the number of concurrent tasks, 
eg. when database overload we want to decrease number of query hitting database, 
  or when CPU/RAM/Bandwidth used by something else, we may want to decrease the number of worker

## How it works

this struct will ensure `min` <= `allowed` <= `max`, and `active` <= `allowed`, if `active` > `allowed`, `BlockUntilAllowed()` will block indefinitely until `active` < `allowed`, number of `active` can be decreased by calling `ReleaseActive()`, number of `allowed` cab be increased or decreased by calling `IncAllowed()` or `DecAllowed()`

## Usage

```go
package main

import "github.com/kokizzu/semlock"

func _() {
    // MinSemaphoreLock will start with allowed=1
    // MaxSemaphoreLock will start with allowed=max
    // maximum 10 concurrent tasks, 100ms delay before try acquire lock again
    s := semlock.NewMinSemaphoreLock(10, 100 * time.Millisecond)
    
    for _ = range 10 { // maximum 10 worker
        go func() {
            for bla := range someChannel {
                // block until acquire lock (active+1)
                s.BlockUntilAllowed() // will block if active >= allowed
                
                // do expensive query or process
                expensiveQueryOrCalculation(bla)
                
                // release lock (active-1)
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

## TODO

- replace WaitDelay with channel so doesn't have to do polling, but if we do that, max can be no longer dynamic.

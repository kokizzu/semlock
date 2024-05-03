
# SemLock

a simple semaphore lock, difference between built-in `sync/semaphore`: the limit can be updated dynamically, difference with worker pool libraries: this package can be used to increase or decrease number of goroutine running at the same time in after initialization.

the use case for this library to dynamically limit the number of concurrent tasks adaptively

eg. when database overload we want to decrease number of query hitting database for example based on latency or when CPU/RAM/Bandwidth overloaded, we may want to decrease the number of worker and increase it back when no longer overloaded.


## How it works

- `allowed` is number of available locks, can be manipulated by calling `IncAllowed()` or `DecAllowed()`
- `active` is number of locks that given to the worker, can be manipulated by calling `BlockUntilAllowed()` or `ReleaseActive()`
- `min` (default 1) and `max` is the minimum and maximum threshold for `allowed`

the `SemaphoreLock` struct will ensure `min` <= `allowed` <= `max`, and `active` <= `allowed`

if `active` >= `allowed` (rate limit exceeded), `BlockUntilAllowed()` (acquire lock) will block indefinitely until locks available (`active` < `allowed`)

lock can be released (decreasing number of `active`) by calling `ReleaseActive()`

number of available locks (number of `allowed`) can be increased or decreased by calling `IncAllowed()` or `DecAllowed()`

```
Example: MinSemaphoreLock, L=lock acquired/active, A=available lock/allowed

   min=1 max=3 available=1
   [A] [ ] [ ]

   thread1: BlockUntilAllowed() // will pass
   [L] [ ] [ ]

   thread2: BlockUntilAllowed() // will block
   [L] [ ] [ ]

   thread3: IncAllowed()
   [L] [A] [ ]
   thread2 continued
   [L] [L] [ ]

   thread3: IncAllowed()
   [L] [L] [A]

   thread1: ReleaseActive()
   [L] [A] [A]

   thread2: releaseActive()
   [A] [A] [A]
```

## Usage Example

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
                s.DecAllowed() // reduce number of goroutine allowed to progress
            case <- cpuLessHalfIdle, <- databaseLessHalfIdleSignal:
                s.IncAllowed() // increase number of allowed goroutine progressing
            case <- ctx.Done():
                close(someChannel)
                return
        }
    }
}
```

## TODO

- replace `WaitDelay` with channel so doesn't have to do polling, but if we do that, `max` limit can be no longer dynamic.

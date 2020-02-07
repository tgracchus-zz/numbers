# README

## Language and performance
Give the requirements, it looks like the app is going to be low level (tcp, not app protocol) network intensive and with high concurrency and contention, given the need keep track totals counts and a write to a single file. That, along the need for high performance code, make me think than golang is a good fit.

So, the following structure is used:
```
ConnectionListener -> calls
    N concurrent NumbersController (handles parsing) -> sends to n channel
        1 NumberStore (fan in from the past n channels into 1 and keeps track of statistics) -> send to 1 channel
            1 NumberWrite (writes numbers to disk)
```

The idea idea is to protect shared resources, memory statistics and the numbers.log, using channels and make only one   
goroutine handle them. So: not sharing them, which afaik it´s a common strategy in golang due to channels and goroutines.

## How to
Executable definition
```bash
./cmd/server/numbers -help
Usage of ./cmd/server/numbers:
      --concurrent-connections int   number of concurrent connections (default 5)
      --port string                  tcp port where to start the server (default "4000")
      --profile                      profile the server
pflag: help requested
```

### Run it    
Two options here:

1.- Building and executing yourself 
```bash
./script/build.sh
```
and then execute it 
```bash
./cmd/server/numbers 
```

2.- Using provided scripts
To run it with default args (5 server connections and port 4000) and no profile
```bash
./script/run.sh
```
To run it with different number of server connections and different port
```bash
./script/run.sh <connections> <port>
```

### Build it    
```bash
./script/build.sh
```
Then it can be run

### CI it    
```bash
./script/ci.sh
```

### Distribute it    
```bash
./script/distribution.sh
```
This will create a linux and darwin distribution under distribution folder, in case you  
want to distribute it

### Test it    
```bash
./script/test.sh
```

### Run with Profile enabled 
```bash
./script/profile.sh
```


## Profiling
I have profiled the final result program with go tool prof. You can see the result in profile001.svg.
As you can see there the code for our app is taking about the 18% of cpu time. The rest is
netpoll, runtime scheduling and I guess channel waiting in the form of pthread_cond_wait and pthread_con_signal.


1.- First, make sure tcp connections are ok, for that we can query netstat and make sure connections   
are opened and closed
```bash
netstat -anp tcp | grep 127.0.0.1.4000
```
2.- Go profiling tools:
In th repo there's a cpu profile of the server when running the load test inside test folder.
It's generated when executing the TestServerBaseline which is 5 clients with 400.000 request each.
To sum up the 2M request required in the description
To start the profiling tool execute
```
go tool pprof numbers_cpu.prof
(pprof) top20
Showing nodes accounting for 15.94s, 94.88% of 16.80s total
Dropped 104 nodes (cum <= 0.08s)
Showing top 20 nodes out of 64
      flat  flat%   sum%        cum   cum%
     4.15s 24.70% 24.70%      4.15s 24.70%  runtime.pthread_cond_wait
     3.26s 19.40% 44.11%      3.26s 19.40%  runtime.kevent
     2.05s 12.20% 56.31%      2.05s 12.20%  syscall.syscall
     1.70s 10.12% 66.43%      1.70s 10.12%  runtime.pthread_cond_signal
     1.46s  8.69% 75.12%      1.46s  8.69%  runtime.usleep
     0.94s  5.60% 80.71%      4.20s 25.00%  runtime.netpoll
     0.48s  2.86% 83.57%      0.48s  2.86%  runtime.nanotime
     0.44s  2.62% 86.19%      0.66s  3.93%  tgracchus/numbers.fanIn.func1.1
     0.39s  2.32% 88.51%      0.39s  2.32%  runtime.(*waitq).dequeueSudoG
     0.22s  1.31% 89.82%      0.22s  1.31%  runtime.pthread_cond_timedwait_relative_np
     0.18s  1.07% 90.89%      0.18s  1.07%  indexbytebody
     0.15s  0.89% 91.79%      0.37s  2.20%  runtime.notetsleep
     0.11s  0.65% 92.44%      0.12s  0.71%  runtime.walltime
     0.09s  0.54% 92.98%      0.09s  0.54%  runtime.madvise
     0.09s  0.54% 93.51%      1.22s  7.26%  runtime.runqgrab
     0.07s  0.42% 93.93%      0.17s  1.01%  runtime.lock
     0.06s  0.36% 94.29%      0.10s   0.6%  runtime.evacuate_fast64
     0.04s  0.24% 94.52%     10.12s 60.24%  runtime.findrunnable
     0.04s  0.24% 94.76%      0.62s  3.69%  runtime.selectgo
     0.02s  0.12% 94.88%     10.91s 64.94%  runtime.schedule
```


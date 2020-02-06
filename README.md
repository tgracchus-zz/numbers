# README

## Language and performance
Give the requirements, it looks like the app is going to be low level (tcp, not app protocol) 
network intensive and with high concurrency and contention, given the need keep track totals counts    
and a write to a single file. That, along the need for high performance code, make me think than golang    
is a good fit.

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
sudo netstat -anp tcp | grep 127.0.0.1.4000
```
2.- Go profiling tools:
In th repo there's a cpu profile of the server when running the load test inside test folder.
It's generated when executing the TestServerBaseline which is 5 clients with 400.000 request each.
To sum up the 2M request required in the description
To start the profiling tool execute
```
go tool pprof numbers_cpu.prof
Type: cpu
Time: Feb 5, 2020 at 10:59pm (CET)
Duration: 50.03s, Total samples = 17.87s (35.72%)
Entering interactive mode (type "help" for commands, "o" for options)
(pprof) top20
Showing nodes accounting for 17.26s, 96.59% of 17.87s total
Dropped 77 nodes (cum <= 0.09s)
Showing top 20 nodes out of 51
      flat  flat%   sum%        cum   cum%
     4.15s 23.22% 23.22%      4.15s 23.22%  runtime.pthread_cond_wait
     3.64s 20.37% 43.59%      3.64s 20.37%  runtime.kevent
     2.60s 14.55% 58.14%      2.61s 14.61%  syscall.syscall
     1.98s 11.08% 69.22%      1.98s 11.08%  runtime.pthread_cond_signal
     1.28s  7.16% 76.39%      1.28s  7.16%  runtime.usleep
     0.85s  4.76% 81.14%      4.49s 25.13%  runtime.netpoll
     0.83s  4.64% 85.79%      0.85s  4.76%  runtime.nanotime
     0.52s  2.91% 88.70%      0.52s  2.91%  runtime.(*waitq).dequeueSudoG
     0.45s  2.52% 91.21%      0.54s  3.02%  tgracchus/numbers.fanIn.func1.1
     0.40s  2.24% 93.45%      0.40s  2.24%  indexbytebody
     0.18s  1.01% 94.46%      0.18s  1.01%  runtime.pthread_cond_timedwait_relative_np
     0.11s  0.62% 95.08%      0.29s  1.62%  runtime.notetsleep
     0.07s  0.39% 95.47%      0.71s  3.97%  runtime.selectgo
     0.06s  0.34% 95.80%      0.31s  1.73%  tgracchus/numbers.NewNumberStore.func1
     0.03s  0.17% 95.97%     10.39s 58.14%  runtime.findrunnable
     0.03s  0.17% 96.14%      1.03s  5.76%  runtime.runqgrab
     0.03s  0.17% 96.31%     11.13s 62.28%  runtime.schedule
     0.02s  0.11% 96.42%      1.11s  6.21%  runtime.ready
     0.02s  0.11% 96.53%      1.05s  5.88%  runtime.runqsteal
     0.01s 0.056% 96.59%      3.03s 16.96%  bufio.(*Reader).ReadBytes
(pprof)
```


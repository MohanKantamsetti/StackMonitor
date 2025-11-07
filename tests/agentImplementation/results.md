Test 1: Agent Language & Protocol Comparison


1. CPU Time

Metric:
process_cpu_seconds_total


2. Memory Usage

Metric:
Resident Memory Size (RSS): process_resident_memory_bytes 
 

3. Throughput

Metric: 
Total Duration : http_request_duration_seconds_sum 
Total HTTP Requests : http_request_total

Calculate the throughput as the number of requests per second:

Throughput = Total HTTP Requests / Total Duration

4. Latency

Metrics :
Total Duration : http_request_duration_seconds_sum 
Total HTTP Requests : http_request_total

Average Latency = http_request_duration_seconds_sum / http_request_total

attempt 1

Server
CPU
Memory
Throughput
Latency
Go Rest
0.15 
28.0 MB
12,500 requests/sec
79.3 ms  
Go Grpc
0.05 
27.8 MB
20,000 requests/sec     
11.6 ms           
Python Rest
0.82    
72.59 MB
1,219 requests/sec    
 314.6 ms
Python Grpc
0.58
58.85 MB
2,777 requests/sec
96.8 ms 

attempt 2

Server
CPU
Memory
Throughput
Latency
Go Rest
0.13 
25.5 MB
14,000 requests/sec 
70.4 ms     
Go Grpc
0.05
27.0 MB     
20,000 requests/sec
8.9 ms 
Python Rest
0.92
72.22 MB
1,086 requests/sec
359.1 ms
Python Grpc
0.63
58.83 MB   
1,587 requests/sec 
96.5 ms 



Test Output Logs:


--- Testing Go REST ---


Average latency for 1000 requests: 108.747233ms
Metrics from http://localhost:8083/metrics:
# HELP go_gc_duration_seconds A summary of the pause duration of garbage collection cycles.
# TYPE go_gc_duration_seconds summary
go_gc_duration_seconds{quantile="0"} 0.000251772
go_gc_duration_seconds{quantile="0.25"} 0.000459847
go_gc_duration_seconds{quantile="0.5"} 0.0005424
go_gc_duration_seconds{quantile="0.75"} 0.000614728
go_gc_duration_seconds{quantile="1"} 0.002610902
go_gc_duration_seconds_sum 0.004479649
go_gc_duration_seconds_count 5
# HELP go_goroutines Number of goroutines that currently exist.
# TYPE go_goroutines gauge
go_goroutines 11
# HELP go_info Information about the Go environment.
# TYPE go_info gauge
go_info{version="go1.25.3"} 1
# HELP go_memstats_alloc_bytes Number of bytes allocated and still in use.
# TYPE go_memstats_alloc_bytes gauge
go_memstats_alloc_bytes 5.693552e+06
# HELP go_memstats_alloc_bytes_total Total number of bytes allocated, even if freed.
# TYPE go_memstats_alloc_bytes_total counter
go_memstats_alloc_bytes_total 9.171448e+06
# HELP go_memstats_buck_hash_sys_bytes Number of bytes used by the profiling bucket hash table.
# TYPE go_memstats_buck_hash_sys_bytes gauge
go_memstats_buck_hash_sys_bytes 1.444351e+06
# HELP go_memstats_frees_total Total number of frees.
# TYPE go_memstats_frees_total counter
go_memstats_frees_total 46473
# HELP go_memstats_gc_sys_bytes Number of bytes used for garbage collection system metadata.
# TYPE go_memstats_gc_sys_bytes gauge
go_memstats_gc_sys_bytes 2.778896e+06
# HELP go_memstats_heap_alloc_bytes Number of heap bytes allocated and still in use.
# TYPE go_memstats_heap_alloc_bytes gauge
go_memstats_heap_alloc_bytes 5.693552e+06
# HELP go_memstats_heap_idle_bytes Number of heap bytes waiting to be used.
# TYPE go_memstats_heap_idle_bytes gauge
go_memstats_heap_idle_bytes 8.445952e+06
# HELP go_memstats_heap_inuse_bytes Number of heap bytes that are in use.
# TYPE go_memstats_heap_inuse_bytes gauge
go_memstats_heap_inuse_bytes 8.953856e+06
# HELP go_memstats_heap_objects Number of allocated objects.
# TYPE go_memstats_heap_objects gauge
go_memstats_heap_objects 19973
# HELP go_memstats_heap_released_bytes Number of heap bytes released to OS.
# TYPE go_memstats_heap_released_bytes gauge
go_memstats_heap_released_bytes 5.718016e+06
# HELP go_memstats_heap_sys_bytes Number of heap bytes obtained from system.
# TYPE go_memstats_heap_sys_bytes gauge
go_memstats_heap_sys_bytes 1.7399808e+07
# HELP go_memstats_last_gc_time_seconds Number of seconds since 1970 of last garbage collection.
# TYPE go_memstats_last_gc_time_seconds gauge
go_memstats_last_gc_time_seconds 1.761451937325844e+09
# HELP go_memstats_lookups_total Total number of pointer lookups.
# TYPE go_memstats_lookups_total counter
go_memstats_lookups_total 0
# HELP go_memstats_mallocs_total Total number of mallocs.
# TYPE go_memstats_mallocs_total counter
go_memstats_mallocs_total 66446
# HELP go_memstats_mcache_inuse_bytes Number of bytes in use by mcache structures.
# TYPE go_memstats_mcache_inuse_bytes gauge
go_memstats_mcache_inuse_bytes 14496
# HELP go_memstats_mcache_sys_bytes Number of bytes used for mcache structures obtained from system.
# TYPE go_memstats_mcache_sys_bytes gauge
go_memstats_mcache_sys_bytes 15704
# HELP go_memstats_mspan_inuse_bytes Number of bytes in use by mspan structures.
# TYPE go_memstats_mspan_inuse_bytes gauge
go_memstats_mspan_inuse_bytes 275520
# HELP go_memstats_mspan_sys_bytes Number of bytes used for mspan structures obtained from system.
# TYPE go_memstats_mspan_sys_bytes gauge
go_memstats_mspan_sys_bytes 293760
# HELP go_memstats_next_gc_bytes Number of heap bytes when next garbage collection will take place.
# TYPE go_memstats_next_gc_bytes gauge
go_memstats_next_gc_bytes 1.2454178e+07
# HELP go_memstats_other_sys_bytes Number of bytes used for other system allocations.
# TYPE go_memstats_other_sys_bytes gauge
go_memstats_other_sys_bytes 2.633505e+06
# HELP go_memstats_stack_inuse_bytes Number of bytes in use by the stack allocator.
# TYPE go_memstats_stack_inuse_bytes gauge
go_memstats_stack_inuse_bytes 3.571712e+06
# HELP go_memstats_stack_sys_bytes Number of bytes obtained from system for stack allocator.
# TYPE go_memstats_stack_sys_bytes gauge
go_memstats_stack_sys_bytes 3.571712e+06
# HELP go_memstats_sys_bytes Number of bytes obtained from system.
# TYPE go_memstats_sys_bytes gauge
go_memstats_sys_bytes 2.8137736e+07
# HELP go_threads Number of OS threads created.
# TYPE go_threads gauge
go_threads 18
# HELP http_request_duration_seconds Duration of HTTP requests.
# TYPE http_request_duration_seconds histogram
http_request_duration_seconds_bucket{path="/ping",le="0.005"} 1000
http_request_duration_seconds_bucket{path="/ping",le="0.01"} 1005
http_request_duration_seconds_bucket{path="/ping",le="0.025"} 1005
http_request_duration_seconds_bucket{path="/ping",le="0.05"} 1005
http_request_duration_seconds_bucket{path="/ping",le="0.1"} 1005
http_request_duration_seconds_bucket{path="/ping",le="0.25"} 1005
http_request_duration_seconds_bucket{path="/ping",le="0.5"} 1005
http_request_duration_seconds_bucket{path="/ping",le="1"} 1005
http_request_duration_seconds_bucket{path="/ping",le="2.5"} 1005
http_request_duration_seconds_bucket{path="/ping",le="5"} 1005
http_request_duration_seconds_bucket{path="/ping",le="10"} 1005
http_request_duration_seconds_bucket{path="/ping",le="+Inf"} 1005
http_request_duration_seconds_sum{path="/ping"} 0.08963540000000013
http_request_duration_seconds_count{path="/ping"} 1005
# HELP http_requests_total Total number of HTTP requests
# TYPE http_requests_total counter
http_requests_total{path="/ping"} 1005
# HELP process_cpu_seconds_total Total user and system CPU time spent in seconds.
# TYPE process_cpu_seconds_total counter
process_cpu_seconds_total 0.24
# HELP process_max_fds Maximum number of open file descriptors.
# TYPE process_max_fds gauge
process_max_fds 1.048576e+06
# HELP process_open_fds Number of open file descriptors.
# TYPE process_open_fds gauge
process_open_fds 13
# HELP process_resident_memory_bytes Resident memory size in bytes.
# TYPE process_resident_memory_bytes gauge
process_resident_memory_bytes 2.7787264e+07
# HELP process_start_time_seconds Start time of the process since unix epoch in seconds.
# TYPE process_start_time_seconds gauge
process_start_time_seconds 1.76145192905e+09
# HELP process_virtual_memory_bytes Virtual memory size in bytes.
# TYPE process_virtual_memory_bytes gauge
process_virtual_memory_bytes 2.485137408e+09
# HELP process_virtual_memory_max_bytes Maximum amount of virtual memory available in bytes.
# TYPE process_virtual_memory_max_bytes gauge
process_virtual_memory_max_bytes 1.8446744073709552e+19
# HELP promhttp_metric_handler_requests_in_flight Current number of scrapes being served.
# TYPE promhttp_metric_handler_requests_in_flight gauge
promhttp_metric_handler_requests_in_flight 1
# HELP promhttp_metric_handler_requests_total Total number of scrapes by HTTP status code.
# TYPE promhttp_metric_handler_requests_total counter
promhttp_metric_handler_requests_total{code="200"} 0
promhttp_metric_handler_requests_total{code="500"} 0
promhttp_metric_handler_requests_total{code="503"} 0



--- Sleeping for 30 seconds between tests ---


 
--- Testing Go gRPC ---


Average latency for 1000 requests: 8.893063ms
Metrics from http://localhost:9092/metrics:
# HELP go_gc_duration_seconds A summary of the wall-time pause (stop-the-world) duration in garbage collection cycles.
# TYPE go_gc_duration_seconds summary
go_gc_duration_seconds{quantile="0"} 0.000163966
go_gc_duration_seconds{quantile="0.25"} 0.000163966
go_gc_duration_seconds{quantile="0.5"} 0.00020844
go_gc_duration_seconds{quantile="0.75"} 0.00020844
go_gc_duration_seconds{quantile="1"} 0.00020844
go_gc_duration_seconds_sum 0.000372406
go_gc_duration_seconds_count 2
# HELP go_gc_gogc_percent Heap size target percentage configured by the user, otherwise 100. This value is set by the GOGC environment variable, and the runtime/debug.SetGCPercent function. Sourced from /gc/gogc:percent.
# TYPE go_gc_gogc_percent gauge
go_gc_gogc_percent 100
# HELP go_gc_gomemlimit_bytes Go runtime memory limit configured by the user, otherwise math.MaxInt64. This value is set by the GOMEMLIMIT environment variable, and the runtime/debug.SetMemoryLimit function. Sourced from /gc/gomemlimit:bytes.
# TYPE go_gc_gomemlimit_bytes gauge
go_gc_gomemlimit_bytes 9.223372036854776e+18
# HELP go_goroutines Number of goroutines that currently exist.
# TYPE go_goroutines gauge
go_goroutines 11
# HELP go_info Information about the Go environment.
# TYPE go_info gauge
go_info{version="go1.25.3"} 1
# HELP go_memstats_alloc_bytes Number of bytes allocated in heap and currently in use. Equals to /memory/classes/heap/objects:bytes.
# TYPE go_memstats_alloc_bytes gauge
go_memstats_alloc_bytes 1.227344e+06
# HELP go_memstats_alloc_bytes_total Total number of bytes allocated in heap until now, even if released already. Equals to /gc/heap/allocs:bytes.
# TYPE go_memstats_alloc_bytes_total counter
go_memstats_alloc_bytes_total 4.40096e+06
# HELP go_memstats_buck_hash_sys_bytes Number of bytes used by the profiling bucket hash table. Equals to /memory/classes/profiling/buckets:bytes.
# TYPE go_memstats_buck_hash_sys_bytes gauge
go_memstats_buck_hash_sys_bytes 1.444429e+06
# HELP go_memstats_frees_total Total number of heap objects frees. Equals to /gc/heap/frees:objects + /gc/heap/tiny/allocs:objects.
# TYPE go_memstats_frees_total counter
go_memstats_frees_total 51817
# HELP go_memstats_gc_sys_bytes Number of bytes used for garbage collection system metadata. Equals to /memory/classes/metadata/other:bytes.
# TYPE go_memstats_gc_sys_bytes gauge
go_memstats_gc_sys_bytes 2.665232e+06
# HELP go_memstats_heap_alloc_bytes Number of heap bytes allocated and currently in use, same as go_memstats_alloc_bytes. Equals to /memory/classes/heap/objects:bytes.
# TYPE go_memstats_heap_alloc_bytes gauge
go_memstats_heap_alloc_bytes 1.227344e+06
# HELP go_memstats_heap_idle_bytes Number of heap bytes waiting to be used. Equals to /memory/classes/heap/released:bytes + /memory/classes/heap/free:bytes.
# TYPE go_memstats_heap_idle_bytes gauge
go_memstats_heap_idle_bytes 2.998272e+06
# HELP go_memstats_heap_inuse_bytes Number of heap bytes that are in use. Equals to /memory/classes/heap/objects:bytes + /memory/classes/heap/unused:bytes
# TYPE go_memstats_heap_inuse_bytes gauge
go_memstats_heap_inuse_bytes 4.243456e+06
# HELP go_memstats_heap_objects Number of currently allocated objects. Equals to /gc/heap/objects:objects.
# TYPE go_memstats_heap_objects gauge
go_memstats_heap_objects 8495
# HELP go_memstats_heap_released_bytes Number of heap bytes released to OS. Equals to /memory/classes/heap/released:bytes.
# TYPE go_memstats_heap_released_bytes gauge
go_memstats_heap_released_bytes 1.695744e+06
# HELP go_memstats_heap_sys_bytes Number of heap bytes obtained from system. Equals to /memory/classes/heap/objects:bytes + /memory/classes/heap/unused:bytes + /memory/classes/heap/released:bytes + /memory/classes/heap/free:bytes.
# TYPE go_memstats_heap_sys_bytes gauge
go_memstats_heap_sys_bytes 7.241728e+06
# HELP go_memstats_last_gc_time_seconds Number of seconds since 1970 of last garbage collection.
# TYPE go_memstats_last_gc_time_seconds gauge
go_memstats_last_gc_time_seconds 1.7614519673733985e+09
# HELP go_memstats_mallocs_total Total number of heap objects allocated, both live and gc-ed. Semantically a counter version for go_memstats_heap_objects gauge. Equals to /gc/heap/allocs:objects + /gc/heap/tiny/allocs:objects.
# TYPE go_memstats_mallocs_total counter
go_memstats_mallocs_total 60312
# HELP go_memstats_mcache_inuse_bytes Number of bytes in use by mcache structures. Equals to /memory/classes/metadata/mcache/inuse:bytes.
# TYPE go_memstats_mcache_inuse_bytes gauge
go_memstats_mcache_inuse_bytes 14496
# HELP go_memstats_mcache_sys_bytes Number of bytes used for mcache structures obtained from system. Equals to /memory/classes/metadata/mcache/inuse:bytes + /memory/classes/metadata/mcache/free:bytes.
# TYPE go_memstats_mcache_sys_bytes gauge
go_memstats_mcache_sys_bytes 15704
# HELP go_memstats_mspan_inuse_bytes Number of bytes in use by mspan structures. Equals to /memory/classes/metadata/mspan/inuse:bytes.
# TYPE go_memstats_mspan_inuse_bytes gauge
go_memstats_mspan_inuse_bytes 155840
# HELP go_memstats_mspan_sys_bytes Number of bytes used for mspan structures obtained from system. Equals to /memory/classes/metadata/mspan/inuse:bytes + /memory/classes/metadata/mspan/free:bytes.
# TYPE go_memstats_mspan_sys_bytes gauge
go_memstats_mspan_sys_bytes 163200
# HELP go_memstats_next_gc_bytes Number of heap bytes when next garbage collection will take place. Equals to /gc/heap/goal:bytes.
# TYPE go_memstats_next_gc_bytes gauge
go_memstats_next_gc_bytes 4.194304e+06
# HELP go_memstats_other_sys_bytes Number of bytes used for other system allocations. Equals to /memory/classes/other:bytes.
# TYPE go_memstats_other_sys_bytes gauge
go_memstats_other_sys_bytes 2.091219e+06
# HELP go_memstats_stack_inuse_bytes Number of bytes obtained from system for stack allocator in non-CGO environments. Equals to /memory/classes/heap/stacks:bytes.
# TYPE go_memstats_stack_inuse_bytes gauge
go_memstats_stack_inuse_bytes 1.14688e+06
# HELP go_memstats_stack_sys_bytes Number of bytes obtained from system for stack allocator. Equals to /memory/classes/heap/stacks:bytes + /memory/classes/os-stacks:bytes.
# TYPE go_memstats_stack_sys_bytes gauge
go_memstats_stack_sys_bytes 1.14688e+06
# HELP go_memstats_sys_bytes Number of bytes obtained from system. Equals to /memory/classes/total:byte.
# TYPE go_memstats_sys_bytes gauge
go_memstats_sys_bytes 1.4768392e+07
# HELP go_sched_gomaxprocs_threads The current runtime.GOMAXPROCS setting, or the number of operating system threads that can execute user-level Go code simultaneously. Sourced from /sched/gomaxprocs:threads.
# TYPE go_sched_gomaxprocs_threads gauge
go_sched_gomaxprocs_threads 12
# HELP go_threads Number of OS threads created.
# TYPE go_threads gauge
go_threads 15
# HELP grpc_request_duration_seconds Duration of gRPC requests.
# TYPE grpc_request_duration_seconds histogram
grpc_request_duration_seconds_bucket{le="0.005"} 1000
grpc_request_duration_seconds_bucket{le="0.01"} 1000
grpc_request_duration_seconds_bucket{le="0.025"} 1000
grpc_request_duration_seconds_bucket{le="0.05"} 1000
grpc_request_duration_seconds_bucket{le="0.1"} 1000
grpc_request_duration_seconds_bucket{le="0.25"} 1000
grpc_request_duration_seconds_bucket{le="0.5"} 1000
grpc_request_duration_seconds_bucket{le="1"} 1000
grpc_request_duration_seconds_bucket{le="2.5"} 1000
grpc_request_duration_seconds_bucket{le="5"} 1000
grpc_request_duration_seconds_bucket{le="10"} 1000
grpc_request_duration_seconds_bucket{le="+Inf"} 1000
grpc_request_duration_seconds_sum 0.0006749670000000003
grpc_request_duration_seconds_count 1000
# HELP grpc_requests_total Total number of gRPC requests
# TYPE grpc_requests_total counter
grpc_requests_total 1000
# HELP process_cpu_seconds_total Total user and system CPU time spent in seconds.
# TYPE process_cpu_seconds_total counter
process_cpu_seconds_total 0.05
# HELP process_max_fds Maximum number of open file descriptors.
# TYPE process_max_fds gauge
process_max_fds 1.048576e+06
# HELP process_network_receive_bytes_total Number of bytes received by the process over the network.
# TYPE process_network_receive_bytes_total counter
process_network_receive_bytes_total 33500
# HELP process_network_transmit_bytes_total Number of bytes sent by the process over the network.
# TYPE process_network_transmit_bytes_total counter
process_network_transmit_bytes_total 47858
# HELP process_open_fds Number of open file descriptors.
# TYPE process_open_fds gauge
process_open_fds 10
# HELP process_resident_memory_bytes Resident memory size in bytes.
# TYPE process_resident_memory_bytes gauge
process_resident_memory_bytes 1.9136512e+07
# HELP process_start_time_seconds Start time of the process since unix epoch in seconds.
# TYPE process_start_time_seconds gauge
process_start_time_seconds 1.76145192906e+09
# HELP process_virtual_memory_bytes Virtual memory size in bytes.
# TYPE process_virtual_memory_bytes gauge
process_virtual_memory_bytes 2.254217216e+09
# HELP process_virtual_memory_max_bytes Maximum amount of virtual memory available in bytes.
# TYPE process_virtual_memory_max_bytes gauge
process_virtual_memory_max_bytes 1.8446744073709552e+19
# HELP promhttp_metric_handler_requests_in_flight Current number of scrapes being served.
# TYPE promhttp_metric_handler_requests_in_flight gauge
promhttp_metric_handler_requests_in_flight 1
# HELP promhttp_metric_handler_requests_total Total number of scrapes by HTTP status code.
# TYPE promhttp_metric_handler_requests_total counter
promhttp_metric_handler_requests_total{code="200"} 0
promhttp_metric_handler_requests_total{code="500"} 0
promhttp_metric_handler_requests_total{code="503"} 0



--- Sleeping for 30 seconds between tests ---


--- Testing Python REST ---


Average latency for 1000 requests: 359.056604ms
Metrics from http://localhost:8082/metrics:
# HELP python_gc_objects_collected_total Objects collected during gc
# TYPE python_gc_objects_collected_total counter
python_gc_objects_collected_total{generation="0"} 373.0
python_gc_objects_collected_total{generation="1"} 395.0
python_gc_objects_collected_total{generation="2"} 177.0
# HELP python_gc_objects_uncollectable_total Uncollectable objects found during GC
# TYPE python_gc_objects_uncollectable_total counter
python_gc_objects_uncollectable_total{generation="0"} 0.0
python_gc_objects_uncollectable_total{generation="1"} 0.0
python_gc_objects_uncollectable_total{generation="2"} 0.0
# HELP python_gc_collections_total Number of times this generation was collected
# TYPE python_gc_collections_total counter
python_gc_collections_total{generation="0"} 318.0
python_gc_collections_total{generation="1"} 28.0
python_gc_collections_total{generation="2"} 2.0
# HELP python_info Python platform information
# TYPE python_info gauge
python_info{implementation="CPython",major="3",minor="11",patchlevel="14",version="3.11.14"} 1.0
# HELP process_virtual_memory_bytes Virtual memory size in bytes.
# TYPE process_virtual_memory_bytes gauge
process_virtual_memory_bytes 3.103379456e+09
# HELP process_resident_memory_bytes Resident memory size in bytes.
# TYPE process_resident_memory_bytes gauge
process_resident_memory_bytes 7.2237056e+07
# HELP process_start_time_seconds Start time of the process since unix epoch in seconds.
# TYPE process_start_time_seconds gauge
process_start_time_seconds 1.76145192913e+09
# HELP process_cpu_seconds_total Total user and system CPU time spent in seconds.
# TYPE process_cpu_seconds_total counter
process_cpu_seconds_total 0.92
# HELP process_open_fds Number of open file descriptors.
# TYPE process_open_fds gauge
process_open_fds 14.0
# HELP process_max_fds Maximum number of open file descriptors.
# TYPE process_max_fds gauge
process_max_fds 1.048576e+06
# HELP http_requests_total Total number of requests by method, status and handler.
# TYPE http_requests_total counter
http_requests_total{handler="/ping",method="GET",status="2xx"} 1005.0
# HELP http_requests_created Total number of requests by method, status and handler.
# TYPE http_requests_created gauge
http_requests_created{handler="/ping",method="GET",status="2xx"} 1.7614519974417403e+09
# HELP http_request_size_bytes Content length of incoming requests by handler. Only value of header is respected. Otherwise ignored. No percentile calculated. 
# TYPE http_request_size_bytes summary
http_request_size_bytes_count{handler="/ping"} 1005.0
http_request_size_bytes_sum{handler="/ping"} 0.0
# HELP http_request_size_bytes_created Content length of incoming requests by handler. Only value of header is respected. Otherwise ignored. No percentile calculated. 
# TYPE http_request_size_bytes_created gauge
http_request_size_bytes_created{handler="/ping"} 1.7614519974417555e+09
# HELP http_response_size_bytes Content length of outgoing responses by handler. Only value of header is respected. Otherwise ignored. No percentile calculated. 
# TYPE http_response_size_bytes summary
http_response_size_bytes_count{handler="/ping"} 1005.0
http_response_size_bytes_sum{handler="/ping"} 18090.0
# HELP http_response_size_bytes_created Content length of outgoing responses by handler. Only value of header is respected. Otherwise ignored. No percentile calculated. 
# TYPE http_response_size_bytes_created gauge
http_response_size_bytes_created{handler="/ping"} 1.7614519974417822e+09
# HELP http_request_duration_highr_seconds Latency with many buckets but no API specific labels. Made for more accurate percentile calculations. 
# TYPE http_request_duration_highr_seconds histogram
http_request_duration_highr_seconds_bucket{le="0.01"} 15.0
http_request_duration_highr_seconds_bucket{le="0.025"} 15.0
http_request_duration_highr_seconds_bucket{le="0.05"} 16.0
http_request_duration_highr_seconds_bucket{le="0.075"} 16.0
http_request_duration_highr_seconds_bucket{le="0.1"} 153.0
http_request_duration_highr_seconds_bucket{le="0.25"} 1005.0
http_request_duration_highr_seconds_bucket{le="0.5"} 1005.0
http_request_duration_highr_seconds_bucket{le="0.75"} 1005.0
http_request_duration_highr_seconds_bucket{le="1.0"} 1005.0
http_request_duration_highr_seconds_bucket{le="1.5"} 1005.0
http_request_duration_highr_seconds_bucket{le="2.0"} 1005.0
http_request_duration_highr_seconds_bucket{le="2.5"} 1005.0
http_request_duration_highr_seconds_bucket{le="3.0"} 1005.0
http_request_duration_highr_seconds_bucket{le="3.5"} 1005.0
http_request_duration_highr_seconds_bucket{le="4.0"} 1005.0
http_request_duration_highr_seconds_bucket{le="4.5"} 1005.0
http_request_duration_highr_seconds_bucket{le="5.0"} 1005.0
http_request_duration_highr_seconds_bucket{le="7.5"} 1005.0
http_request_duration_highr_seconds_bucket{le="10.0"} 1005.0
http_request_duration_highr_seconds_bucket{le="30.0"} 1005.0
http_request_duration_highr_seconds_bucket{le="60.0"} 1005.0
http_request_duration_highr_seconds_bucket{le="+Inf"} 1005.0
http_request_duration_highr_seconds_count 1005.0
http_request_duration_highr_seconds_sum 143.12402803100667
# HELP http_request_duration_highr_seconds_created Latency with many buckets but no API specific labels. Made for more accurate percentile calculations. 
# TYPE http_request_duration_highr_seconds_created gauge
http_request_duration_highr_seconds_created 1.7614519305338378e+09
# HELP http_request_duration_seconds Latency with only few buckets by handler. Made to be only used if aggregation by handler is important. 
# TYPE http_request_duration_seconds histogram
http_request_duration_seconds_bucket{handler="/ping",le="0.1",method="GET"} 153.0
http_request_duration_seconds_bucket{handler="/ping",le="0.5",method="GET"} 1005.0
http_request_duration_seconds_bucket{handler="/ping",le="1.0",method="GET"} 1005.0
http_request_duration_seconds_bucket{handler="/ping",le="+Inf",method="GET"} 1005.0
http_request_duration_seconds_count{handler="/ping",method="GET"} 1005.0
http_request_duration_seconds_sum{handler="/ping",method="GET"} 143.12402803100667
# HELP http_request_duration_seconds_created Latency with only few buckets by handler. Made to be only used if aggregation by handler is important. 
# TYPE http_request_duration_seconds_created gauge
http_request_duration_seconds_created{handler="/ping",method="GET"} 1.7614519974418087e+09



--- Sleeping for 30 seconds between tests ---


--- Testing Python gRPC ---


Average latency for 1000 requests: 96.463323ms
Metrics from http://localhost:9091/:
# HELP python_gc_objects_collected_total Objects collected during gc
# TYPE python_gc_objects_collected_total counter
python_gc_objects_collected_total{generation="0"} 342.0
python_gc_objects_collected_total{generation="1"} 15.0
python_gc_objects_collected_total{generation="2"} 0.0
# HELP python_gc_objects_uncollectable_total Uncollectable objects found during GC
# TYPE python_gc_objects_uncollectable_total counter
python_gc_objects_uncollectable_total{generation="0"} 0.0
python_gc_objects_uncollectable_total{generation="1"} 0.0
python_gc_objects_uncollectable_total{generation="2"} 0.0
# HELP python_gc_collections_total Number of times this generation was collected
# TYPE python_gc_collections_total counter
python_gc_collections_total{generation="0"} 60.0
python_gc_collections_total{generation="1"} 5.0
python_gc_collections_total{generation="2"} 0.0
# HELP python_info Python platform information
# TYPE python_info gauge
python_info{implementation="CPython",major="3",minor="12",patchlevel="12",version="3.12.12"} 1.0
# HELP process_virtual_memory_bytes Virtual memory size in bytes.
# TYPE process_virtual_memory_bytes gauge
process_virtual_memory_bytes 2.098294784e+09
# HELP process_resident_memory_bytes Resident memory size in bytes.
# TYPE process_resident_memory_bytes gauge
process_resident_memory_bytes 5.8290176e+07
# HELP process_start_time_seconds Start time of the process since unix epoch in seconds.
# TYPE process_start_time_seconds gauge
process_start_time_seconds 1.7614519291e+09
# HELP process_cpu_seconds_total Total user and system CPU time spent in seconds.
# TYPE process_cpu_seconds_total counter
process_cpu_seconds_total 0.63
# HELP process_open_fds Number of open file descriptors.
# TYPE process_open_fds gauge
process_open_fds 10.0
# HELP process_max_fds Maximum number of open file descriptors.
# TYPE process_max_fds gauge
process_max_fds 1.048576e+06
# HELP grpc_requests_total Total number of gRPC requests
# TYPE grpc_requests_total counter
grpc_requests_total 1000.0
# HELP grpc_requests_created Total number of gRPC requests
# TYPE grpc_requests_created gauge
grpc_requests_created 1.7614519302998118e+09
# HELP grpc_request_duration_seconds Duration of gRPC requests
# TYPE grpc_request_duration_seconds histogram
grpc_request_duration_seconds_bucket{le="0.005"} 1000.0
grpc_request_duration_seconds_bucket{le="0.01"} 1000.0
grpc_request_duration_seconds_bucket{le="0.025"} 1000.0
grpc_request_duration_seconds_bucket{le="0.05"} 1000.0
grpc_request_duration_seconds_bucket{le="0.075"} 1000.0
grpc_request_duration_seconds_bucket{le="0.1"} 1000.0
grpc_request_duration_seconds_bucket{le="0.25"} 1000.0
grpc_request_duration_seconds_bucket{le="0.5"} 1000.0
grpc_request_duration_seconds_bucket{le="0.75"} 1000.0
grpc_request_duration_seconds_bucket{le="1.0"} 1000.0
grpc_request_duration_seconds_bucket{le="2.5"} 1000.0
grpc_request_duration_seconds_bucket{le="5.0"} 1000.0
grpc_request_duration_seconds_bucket{le="7.5"} 1000.0
grpc_request_duration_seconds_bucket{le="10.0"} 1000.0
grpc_request_duration_seconds_bucket{le="+Inf"} 1000.0
grpc_request_duration_seconds_count 1000.0
grpc_request_duration_seconds_sum 0.0038068294525146484
# HELP grpc_request_duration_seconds_created Duration of gRPC requests
# TYPE grpc_request_duration_seconds_created gauge
grpc_request_duration_seconds_created 1.7614519302998524e+09







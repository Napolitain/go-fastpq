❯ go test -run '^$' -bench 'BenchmarkQueue' -benchmem
goos: darwin
goarch: arm64
pkg: github.com/Napolitain/go-fastpq
cpu: Apple M4
BenchmarkQueuePushPop/priorities_10/items_per_bucket_100/fastpq-10        197941             6110 ns/op           20792 B/op         83 allocs/op
BenchmarkQueuePushPop/priorities_10/items_per_bucket_100/container_heap-10                20264             59001 ns/op           72608 B/op       2002 allocs/op
BenchmarkQueuePushPop/priorities_10/items_per_bucket_10000/fastpq-10                       2010            601464 ns/op         3576648 B/op        193 allocs/op
BenchmarkQueuePushPop/priorities_10/items_per_bucket_10000/container_heap-10                 85          13974729 ns/op         7200303 B/op     200002 allocs/op
BenchmarkQueuePushPop/priorities_10/items_per_bucket_1000000/fastpq-10                       19          55547919 ns/op        416781595 B/op       385 allocs/op
BenchmarkQueuePushPop/priorities_10/items_per_bucket_1000000/container_heap-10                1        2128741625 ns/op        720001248 B/op  20000004 allocs/op
BenchmarkQueuePushPop/priorities_1000/items_per_bucket_100/fastpq-10                       1941            618956 ns/op         2072971 B/op       8003 allocs/op
BenchmarkQueuePushPop/priorities_1000/items_per_bucket_100/container_heap-10                 81          14660891 ns/op         7200311 B/op     200002 allocs/op
BenchmarkQueuePushPop/priorities_1000/items_per_bucket_10000/fastpq-10                       18          64717264 ns/op        357657059 B/op     19003 allocs/op
BenchmarkQueuePushPop/priorities_1000/items_per_bucket_10000/container_heap-10                1        2595122666 ns/op        720001248 B/op  20000004 allocs/op
BenchmarkQueuePushPop/priorities_100000/items_per_bucket_100/fastpq-10                       12          96787941 ns/op        207217176 B/op    800003 allocs/op
BenchmarkQueuePushPop/priorities_100000/items_per_bucket_100/container_heap-10                1        2243074959 ns/op        720001248 B/op  20000004 allocs/op
BenchmarkQueueSteadyFlow/priorities_10/items_per_bucket_100/fastpq-10                  258531934                4.642 ns/op           0 B/op          0 allocs/op
BenchmarkQueueSteadyFlow/priorities_10/items_per_bucket_100/container_heap-10          14406583                82.65 ns/op           48 B/op          2 allocs/op
BenchmarkQueueSteadyFlow/priorities_10/items_per_bucket_10000/fastpq-10                257629177                4.656 ns/op           0 B/op          0 allocs/op
BenchmarkQueueSteadyFlow/priorities_10/items_per_bucket_10000/container_heap-10         8607201               131.9 ns/op            48 B/op          2 allocs/op
BenchmarkQueueSteadyFlow/priorities_10/items_per_bucket_1000000/fastpq-10              217237099                5.241 ns/op           7 B/op          0 allocs/op
BenchmarkQueueSteadyFlow/priorities_10/items_per_bucket_1000000/container_heap-10       6162297               213.8 ns/op            48 B/op          2 allocs/op
BenchmarkQueueSteadyFlow/priorities_1000/items_per_bucket_100/fastpq-10                158383824                7.604 ns/op           0 B/op          0 allocs/op
BenchmarkQueueSteadyFlow/priorities_1000/items_per_bucket_100/container_heap-10         8725818               129.2 ns/op            48 B/op          2 allocs/op
BenchmarkQueueSteadyFlow/priorities_1000/items_per_bucket_10000/fastpq-10              163382545                7.492 ns/op           6 B/op          0 allocs/op
BenchmarkQueueSteadyFlow/priorities_1000/items_per_bucket_10000/container_heap-10       4365426               286.8 ns/op            48 B/op          2 allocs/op
BenchmarkQueueSteadyFlow/priorities_100000/items_per_bucket_100/fastpq-10              100000000              137.4 ns/op             5 B/op          0 allocs/op
BenchmarkQueueSteadyFlow/priorities_100000/items_per_bucket_100/container_heap-10       5963227               229.1 ns/op            48 B/op          2 allocs/op
PASS
ok      github.com/Napolitain/go-fastpq 54.039s

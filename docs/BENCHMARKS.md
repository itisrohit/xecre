# Local execution measurements

Measured on November 17, 2026. These results describe one local run, not a throughput or latency guarantee.

## Environment and workload

- Docker Desktop Engine 28.3.3, aarch64.
- Docker reported 8 CPUs and approximately 3.83 GiB memory.
- Python 3.11 Alpine and Node 20 Alpine execution images.
- Per-sandbox CPU quota: half a core; memory limit: 256 MiB.
- Pool target: two ready containers per language; global tracked sandbox capacity: six.
- Four admitted requests maximum; one pool builder per language.
- Another application remained running, so the host was not dedicated to the benchmark.

Each row below contains 20 requests, for 80 measured requests total. Workloads were `print(42)` and `console.log(42)`. The client verified HTTP 200, expected stdout, and exit code zero. Timing spans the complete HTTP request and decoded response. Runs used fixed client concurrency and warmed the pool before each group; pool replenishment continued during the group. Groups ran in table order. Values are rounded to milliseconds.

## Results

| Language | Concurrency | Successful / total | Mean | Median | p95 | Mean pool wait | Mean execution + Docker | Successful requests/s |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| Python | 1 | 20 / 20 | 446 ms | 390 ms | 812 ms | 293 ms | 147 ms | 2.23 |
| Python | 2 | 20 / 20 | 980 ms | 1,063 ms | 1,404 ms | 807 ms | 167 ms | 2.00 |
| JavaScript | 1 | 20 / 20 | 774 ms | 600 ms | 1,655 ms | 310 ms | 454 ms | 1.29 |
| JavaScript | 2 | 20 / 20 | 1,243 ms | 1,311 ms | 1,948 ms | 886 ms | 344 ms | 1.57 |

Percentiles use the nearest-rank method; the median averages the middle pair. Throughput is successful requests divided by the group's elapsed wall time. Latencies exclude any client executor wait before starting a request.

A separate Python request immediately after HTTP availability took 200 ms, including 82 ms pool wait. This is one startup sample, not a cold-start distribution and not a measurement from process launch to readiness.

## Interpretation and limitations

At concurrency two, waiting for replacement containers dominated Python latency and was the largest component of JavaScript latency. Increasing client concurrency did not consistently improve throughput. The current half-core quotas, Docker Desktop VM, interpreter startup, and container creation/removal all influence the results. These samples do not isolate their individual contributions.

Twenty observations per scenario provide a small baseline. Repeat scenarios across multiple runs before drawing conclusions about p95. No p99, maximum sustainable throughput, or production capacity claim is supported by this run. The earlier approximately 84 ms figure came from dividing batch wall time by concurrent request count and is not a valid mean request latency.

Separate checks verified nonzero exit-code propagation and invalid-language validation. Earlier targeted runtime checks verified memory/CPU/process/network configuration, output caps, execution deadlines, stale ownership-label recovery, and shutdown cleanup. After this measurement run, the API was stopped and zero app-labeled sandboxes remained.

## Reproduce a small Python baseline

Start the app using the README instructions. Allow both pools to fill, then run:

```bash
go run scripts/benchmark.go -requests 20 -concurrency 1
go run scripts/benchmark.go -requests 20 -concurrency 2
docker compose stop
```

The bundled benchmark currently runs Python only and checks HTTP status rather than output correctness. The four-group measurement above used a separate HTTP client with output validation for both languages; the commands are a comparable Python baseline, not an exact reproduction of the full suite. Record the environment and error count alongside every result.

# http_accumulator
Acts as a proxy that accumulates many small http requests into bigger requests

## testing it

`docker-compose up` will raise the test stack

send requests to our demo 'doubler' server at `curl localhost:9991 --data twelve=12`


now, use the accumulator to consolidate requests into larger batches

    for i in `seq 10`; do curl localhost:9999 --data twelve=12 > /tmp/testing_$i & done
    cat /tmp/testing_*

the accumulator sorts everything out for you, creating less pressure on the server

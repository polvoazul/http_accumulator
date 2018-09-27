# http_accumulator
Acts as a proxy that accumulates many small http requests into bigger requests

## testing it

`docker-compose up` will raise the test stack

Send requests to our demo 'doubler' server at `curl localhost:9991 --data 12 --data 10`

Notice that it needs to support reading and responding in multiform content-type. We will later add support for other forms
of batching, notably JSON arrays when content is JSON.

Now, use the accumulator to consolidate singular requests into larger batches

    for i in `seq 10`; do curl localhost:9999 --data $i > /tmp/testing_$i & done
    cat /tmp/testing_*

The accumulator will wait and hold connections until there are N accumulated requests
or T time has passed since the first request and finally send everything
to the underlying server. It will then de-multiplex responses and send the correct one to each client

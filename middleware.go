package main

import (
    "time"
    "strconv"
    "io/ioutil"
    "mime"
    "log"
    "net/http"
    "net/http/httptest"
    "mime/multipart"
    "bytes"

)

func init() {
    log.SetFlags(log.Lshortfile)
}


type request_w_handle struct {r *http.Request; handle chan []byte}
func enable_accumulation(next http.HandlerFunc) http.HandlerFunc {
    var main_chan = make(chan request_w_handle)
    go accumulator(main_chan, next)
    return func(w http.ResponseWriter, r *http.Request) {
        var handle = make(chan []byte)
        main_chan <- request_w_handle{r, handle}
        log.Printf("Tracing request for %s", r.RequestURI)
        var result = <- handle
        w.Write(result)
    }
}

const BATCH int = 3
const TIMEOUT = 3
func accumulator(main_chan chan request_w_handle, next http.HandlerFunc){
    for {
        var response = httptest.NewRecorder()
        request_body := new(bytes.Buffer)
        writer := multipart.NewWriter(request_body)
        var handles [BATCH]chan []byte
        timeout := time.After(TIMEOUT * time.Second)
        var i int
        buffering: for i = 0; i < BATCH; i++ {
            select {
                case request_w_handle := <- main_chan:
                    log.Println("RECEIVED" + request_w_handle.r.RequestURI)
                    writer.WriteField(strconv.Itoa(i), request_w_handle.r.RequestURI)
                    handles[i] = request_w_handle.handle
                    break
                case <- timeout: log.Println("timeout"); break buffering
            }
        }
        batch_size := i
        if (batch_size == 0) {continue}
        var request = httptest.NewRequest("POST", "/", request_body)
        request.Header.Set("Content-Type", "multipart/mixed; boundary=" + writer.Boundary())
        next.ServeHTTP(response, request)
        _, params, err := mime.ParseMediaType(response.Header().Get("Content-Type"))
        if (err != nil) {panic("FUCK")};
        var reader = multipart.NewReader(response.Body, params["boundary"])
        for i:= 0; i< batch_size; i++ {
            var part, err = reader.NextPart()
            if(err == nil) {
                var part_content, _ = ioutil.ReadAll(part)
                handles[i] <- part_content
            }
        }
    }
}

func home(w http.ResponseWriter, r *http.Request) {
    writer := multipart.NewWriter(w)
    var reader, _ = r.MultipartReader()
    for part, err := reader.NextPart(); err == nil; part, err = reader.NextPart() {
        var part_content, _ = ioutil.ReadAll(part)
        writer.WriteField(part.FileName(), "hello " + string(part_content))
    }
    w.Header().Set("Content-Type", "multipart/mixed; boundary=" + writer.Boundary())
    // fmt.Fprintf(w, "welcome " + r.RequestURI + string(body))
}

func main() {
    http.Handle("/", enable_accumulation(home))
    log.Fatal(http.ListenAndServe(":8080", nil))
}

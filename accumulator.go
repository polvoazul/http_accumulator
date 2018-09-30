package main

import (
    "time"
    //"encoding/json"
    "io"
    "os"
    "strconv"
    "io/ioutil"
    "mime"
    "net/url"
    "log"
    "net/http"
    "net/http/httptest"
    "net/http/httputil"
    "mime/multipart"
    "bytes"
)

func decorator_main_() {
    http.Handle("/", enable_accumulation(internal_handler_function))
    log.Fatal(http.ListenAndServe(":8080", nil))
}

func main() {
    url, _ := url.Parse("http://" + os.Getenv("SERVICE_URL"))
    http.Handle("/", enable_accumulation(httputil.NewSingleHostReverseProxy(url).ServeHTTP))
    log.Fatal(http.ListenAndServe(":9992", nil))
}

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

type batch_writer interface {
    WriteField(string, string) error
    Close() error
    Boundary() string
}

type multipart_writer batch_writer

func NewMultipartWriter(buffer io.Writer) multipart_writer {
    return multipart.NewWriter(buffer)
}

type json_writer struct {
    buffer io.Writer
    first bool;
}

func NewJsonWriter(buffer io.Writer) *json_writer {
    r := new(json_writer)
    r.first = true
    r.buffer = buffer
    return r
}

func (json json_writer) Boundary() string { return "" }
func (json json_writer) Close() error { return nil}
func (json json_writer) WriteField(request string, boundary string) error {
    if(!json.first) {
        json.buffer.Write([]byte(","))
    }
    json.first = false
    json.buffer.Write([]byte(request))
    return nil
}




const BATCH int = 3
const TIMEOUT = 3
func accumulator(main_chan chan request_w_handle, next http.HandlerFunc){
    for {
        request_body := new(bytes.Buffer)
        //var writer batch_writer = multipart.NewWriter(request_body)
        var writer batch_writer = NewJsonWriter(request_body)
        var handles [BATCH]chan []byte
        timeout := time.After(TIMEOUT * time.Second)
        var i int
        buffering: for i = 0; i < BATCH; i++ {
            select {
                case request_w_handle := <- main_chan:
                    log.Println("RECEIVED" + request_w_handle.r.RequestURI)
                    field, _ := ioutil.ReadAll(request_w_handle.r.Body)
                    writer.WriteField(strconv.Itoa(i), string(field))
                    handles[i] = request_w_handle.handle
                    break
                case <- timeout: log.Println("timeout"); break buffering
            }
        }
        batch_size := i
        if (batch_size == 0) {continue}
        writer.Close()
        response := make_request(request_body, writer.Boundary(), next)
        _, params, err := mime.ParseMediaType(response.Header().Get("Content-Type"))
        if (err != nil) {panic("DAMN!")};
        var reader = multipart.NewReader(response.Body, params["boundary"])
        for i:= 0; i< batch_size; i++ {
            var part, err = reader.NextPart()
            if(err == nil) {
                var part_content, _ = ioutil.ReadAll(part)
                handles[i] <- part_content
            } else {
                handles[i] <- []byte(err.Error())
            }
        }
    }
}

func make_request(request_body io.Reader, boundary string, next http.HandlerFunc) *httptest.ResponseRecorder {
    var response = httptest.NewRecorder()
    var request = httptest.NewRequest("POST", "/", request_body)
    request.Header.Set("Content-Type", "multipart/form-data; boundary=" + boundary)
    log.Println("Sending request")
    next.ServeHTTP(response, request)
    return response
}

func internal_handler_function(w http.ResponseWriter, r *http.Request) {
    writer := multipart.NewWriter(w)
    var reader, _ = r.MultipartReader()
    for part, err := reader.NextPart(); err == nil; part, err = reader.NextPart() {
        var part_content, _ = ioutil.ReadAll(part)
        writer.WriteField(part.FileName(), "hello " + string(part_content))
    }
    w.Header().Set("Content-Type", "multipart/mixed; boundary=" + writer.Boundary())
    // fmt.Fprintf(w, "welcome " + r.RequestURI + string(body))
}


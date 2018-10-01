package main

import (
    "time"
    "encoding/json"
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
    go accumulator(main_chan, next, get_writer_type())
    return func(w http.ResponseWriter, r *http.Request) {
        var handle = make(chan []byte)
        main_chan <- request_w_handle{r, handle}
        log.Printf("Tracing request for %s", r.RequestURI)
        var result = <- handle
        log.Printf("Returning request for %s", result)
        w.Write(result)
    }
}

func get_writer_type() func(io.Writer) batch_writer{
    content_type, defined := os.LookupEnv("CONTENT_TYPE")
    if(defined == false){
        content_type = "application/json"
    }
    switch content_type {
        case "application/json":
            return NewJsonWriter
        case "multipart/form-data":
            return NewMultipartWriter
        default:
            log.Panic("CANNOT HANDLE CONTENT_TYPE: " + content_type)
    }
    return nil
}


type batch_reader interface {
    Next() ([]byte, error)
}

type batch_json_reader struct {
    jsons []json.RawMessage
    counter int
}
type batch_multipart_reader struct {
    *multipart.Reader
}
func NewBatchReader(body io.Reader, header http.Header) batch_reader {
    content_type := header.Get("Content-Type")
    switch content_type {
        case "application/json":
            decoder := json.NewDecoder(body)
            var out []json.RawMessage
            decoder.Decode(&out)
            return &batch_json_reader{out, 0}
        case "multipart/form-data":
            _, content_type_map, _ := mime.ParseMediaType(content_type)
            return batch_multipart_reader{multipart.NewReader(body, content_type_map["boundary"])}
        default:
            b, _ := ioutil.ReadAll(body)
            log.Println("BAD RESPONSE: " + string(b))
            log.Panic("CANNOT HANDLE CONTENT_TYPE IN RESPONSE: " + content_type)
            return nil
    }
}
func (self batch_multipart_reader) Next() ([]byte, error){
    part, err := self.NextPart()
    if(err!=nil) {return nil, err}
    part_content, err2 := ioutil.ReadAll(part)
    return part_content, err2
}
func (self *batch_json_reader) Next() ([]byte, error){
    ret := []byte(self.jsons[self.counter])
    self.counter++
    return ret, nil
}


type batch_writer interface {
    WriteField(string, string) error
    Close() error
    ContentType() string
}

type multipart_writer struct {
    *multipart.Writer
}

func NewMultipartWriter(buffer io.Writer) batch_writer {
    return multipart_writer{multipart.NewWriter(buffer)}
}
func (self multipart_writer) ContentType() string { return "multipart/form-data; boundary=" + self.Boundary()}

type json_writer struct {
    buffer io.Writer
    first bool;
}

func NewJsonWriter(buffer io.Writer) batch_writer {
    r := json_writer{first: true, buffer: buffer}
    return &r
}

func (self json_writer) ContentType() string {return "application/json"}
func (self json_writer) Close() error {
    _, err := self.buffer.Write([]byte("]"))
    return err
}
func (self *json_writer) WriteField(index string, request string) error {
    if(self.first == true) {
        self.buffer.Write([]byte("[\n"))
        self.first = false
    } else {
        self.buffer.Write([]byte(","))
    }
    log.Println(request)
    self.buffer.Write([]byte(request))
    return nil
}

const BATCH int = 3
const TIMEOUT = 3
func accumulator(main_chan chan request_w_handle, next http.HandlerFunc, batch_writer func(io.Writer) batch_writer){
    for {
        request_body := new(bytes.Buffer)
        writer := batch_writer(request_body)
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
        response := make_request(request_body, writer.ContentType(), next)
        var reader = NewBatchReader(response.Body, response.Header())
        for i:= 0; i< batch_size; i++ {
            var part, err = reader.Next()
            if(err == nil) {
                handles[i] <- []byte(part)
            } else {
                handles[i] <- []byte(err.Error())
            }
        }
    }
}

func make_request(request_body io.Reader, content_type string, next http.HandlerFunc) *httptest.ResponseRecorder {
    var response = httptest.NewRecorder()
    var request = httptest.NewRequest("POST", "/", request_body)
    request.Header.Set("Content-Type", content_type)
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


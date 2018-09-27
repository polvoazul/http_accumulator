import flask
import time
from requests_toolbelt import MultipartEncoder

app = flask.Flask('')

@app.route('/', methods=['POST'])
def echo():
    time.sleep(2) # this is a very hard computation
    doubled = {k: str(int(v) * 2) for k, v in flask.request.form.items()}
    m = MultipartEncoder(fields=doubled)
    return flask.Response(content_type=m.content_type, response=m.to_string())

if __name__ == '__main__':
    app.run(host='0.0.0.0', port=9991, debug=True)

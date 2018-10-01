import flask
import time, json
from requests_toolbelt import MultipartEncoder

app = flask.Flask('')

@app.route('/', methods=['POST'])
def doubler():
    print("%%%REQUEST: " +str(flask.request.get_data()) + str(flask.request.__dict__))
    time.sleep(1) # this is a very hard computation
    if flask.request.content_type == 'application/json':
        ret = json_doubler()
    elif 'multipart/form-data' in flask.request.content_type:
        ret = multipart_doubler()
    print("%%%RETURNING: " + str(ret.__dict__))
    return ret

def json_doubler():
    doubled = [int(v) * 2 for v in flask.request.json]
    return flask.Response(content_type='application/json', response=json.dumps(doubled))

def multipart_doubler():
    doubled = {k: str(int(v) * 2) if v else str(int(k) * 2) for k, v in flask.request.form.items()}
    m = MultipartEncoder(fields=doubled)
    return flask.Response(content_type=m.content_type, response=m.to_string())

if __name__ == '__main__':
    app.run(host='0.0.0.0', port=9991, debug=True)

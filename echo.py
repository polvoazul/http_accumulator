import flask
app = flask.Flask('')

@app.route('/', methods=['POST'])
def echo():
    ret = flask.Response(content_type=flask.request.content_type)
    ret.set_data(flask.request.stream.read())
    return ret

app.run(port=1234, debug=True)

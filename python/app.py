UPSTREAM = 'http://localhost:8000'
import requests_async as requests

class App:
    async def __call__(self, scope, receive, send):
        assert scope['type'] == 'http'
        body = (await receive())['body'] # TODO: investigate 'more body' flag
        req = requests.request(method=scope['method'], url=UPSTREAM + '/' + scope['path'], data=body)
        response = await req

        await send({
            'type': 'http.response.start',
            'status': response.status_code,
            # 'headers': [
            #     [b'content-type', b'text/plain'],
            # ],
            # 'headers': [(bytes(k, encoding='ascii'), bytes(v, encoding='ascii')) for k,v in response.headers.items()],
        })
        await send({
            'type': 'http.response.body',
            'body': await response.raw.read(),
        })

app = App()

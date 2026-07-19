function handler(event) {
    var request = event.request;
    if (request.headers.host) {
        request.headers['x-forwarded-host'] = { value: request.headers.host.value };
    }
    return request;
}

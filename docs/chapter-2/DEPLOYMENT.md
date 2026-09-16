# Synthesis rate limit

Do not publicly enable `POST /api/v1/answer` until the deployment proxy enforces this limit before traffic reaches the Go API:

```nginx
limit_req_zone $binary_remote_addr zone=searchlens_answer:10m rate=5r/m;

location = /api/v1/answer {
    limit_req zone=searchlens_answer burst=2 nodelay;
    proxy_pass http://searchlens_api;
}
```

Use the deployment platform's equivalent when NGINX is not the proxy. Apply the limit to the verified client IP; do not trust an unvalidated forwarding header.

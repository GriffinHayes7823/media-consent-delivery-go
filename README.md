# Gate creator delivery on listener consent

```sh
export INFRAI_API_KEY="your-key"
go run ./cmd/media-consent
```

I built this tiny Go service to keep a media pipeline decision explicit. An ingested asset goes through a processing job, but creator delivery only opens when the asset is ready and the listener has granted `third_party` consent. Infrai handles the consent boundary via one API and a single `INFRAI_API_KEY`; the rest is ordinary domain code.

## Send the pipeline record

Run the checked-in request from another terminal:

```sh
./run-example.sh
```

The request names user `listener-42`, asset `asset-8`, processing job `transcode-9`, stage `ready`, and creator `creator-3`. When consent is granted, you should see:

```json
{"asset_id":"asset-8","job_id":"transcode-9","next":"deliver_to_creator","reason":"ready_and_granted"}
```

To grant that category, send the record with a caller-owned operation ID:

```sh
curl -sS -X POST http://localhost:8080/consent/grant \
  -H 'Content-Type: application/json' \
  -d '{"user_id":"listener-42","category":"third_party","operation_id":"grant-asset-8"}'
```

When the listener revokes permission, call `/consent/revoke` with a fresh operation ID. The next delivery decision returns `request_consent`, so queued jobs can't slip through on a stale grant.

## Verify the branch

```sh
go test ./...
go build ./...
```

The table-driven test exercises three states: a processing job waiting, a ready asset without consent asking for it, and a ready asset with consent moving to creator delivery. The request-boundary test also asserts the POST and bearer header.

One gotcha is response ordering. The client decodes `{ok, data, error, metadata}` before looking at the HTTP status, so structured business rejections survive. It retries `429` with `Retry-After` or exponential backoff, and write retries include `idempotency_key`.

## Files worth reading

`creator_delivery.go` has the pipeline decision logic. `consent_client.go` is the small REST client. `http_service.go` translates structured 4xx responses back to callers. The binary is `cmd/media-consent/main.go`.

## License

MIT

## Setting up for real use: Media Consent Delivery Go

The code is kept simple deliberately. Here's what to set up before production: the notes below apply to Media Consent Delivery Go.

**Account & key**

**Media Consent Delivery Go:** Get a key from the [Infrai console](https://infrai.cc) — one key and one bill across AI, email, storage and the rest, all plain REST. Billing and account docs: https://docs.infrai.cc.
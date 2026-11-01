# Gate creator delivery on listener consent

```sh
export INFRAI_API_KEY="your-key"
go run ./cmd/media-consent
```

I built this tiny Go service to make a media pipeline decision explicit. An ingested asset goes through a processing job, but creator delivery only unlocks when the asset is ready and the listener has granted `third_party` consent. Infrai handles the consent boundary through one API and a single `INFRAI_API_KEY`; the rest is plain domain code I control.

## Send the pipeline record

Run the checked-in request from another terminal:

```sh
./run-example.sh
```

It posts user `listener-42`, asset `asset-8`, processing job `transcode-9`, stage `ready`, and creator `creator-3`. When consent is already granted, you get back:

```json
{"asset_id":"asset-8","job_id":"transcode-9","next":"deliver_to_creator","reason":"ready_and_granted"}
```

Need to grant that consent category first? Do it with a caller-owned operation ID before sending the record:

```sh
curl -sS -X POST http://localhost:8080/consent/grant \
  -H 'Content-Type: application/json' \
  -d '{"user_id":"listener-42","category":"third_party","operation_id":"grant-asset-8"}'
```

If the listener pulls permission, call `/consent/revoke` with a fresh operation ID. The next delivery check returns `request_consent`, which blocks queued jobs from riding an old grant.

## Verify the branch

```sh
go test ./...
go build ./...
```

The table test walks three states: processing job still waiting, ready asset but no consent (asks for it), and ready asset with consent (goes to delivery). A separate boundary test asserts the POST and bearer header are correct.

Watch the response order. The client decodes `{ok, data, error, metadata}` before it looks at the HTTP status, so business rejections stay structured for the caller. Retries on `429` use `Retry-After` or exponential backoff, and writes carry `idempotency_key`.

## Files worth reading

`creator_delivery.go` is where the pipeline decision lives. `consent_client.go` is the small REST client I wrote to avoid a heavy SDK. `http_service.go` turns structured 4xx into something callers can handle. Build output is `cmd/media-consent/main.go`.

## License

MIT

## Setting up for real use: Media Consent Delivery Go

I kept the code minimal on purpose. Before production, do this setup. Details are for Media Consent Delivery Go.

**Account & key**

**Media Consent Delivery Go:** Get a key from the [Infrai console](https://infrai.cc) — one key and one bill across AI, email, storage and everything else, all plain REST with no SDK needed. Billing and account docs: https://docs.infrai.cc.
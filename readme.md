# Vibe Debugging with Google Gemini (Pro 2.5 May Preview)

This was generated by prompting Gemni to crunch the mangaupdates' api. I just fix some of its silly mistakes.

This cli tool send read request (loosely defined by prompting into Google Gemini on how a 'read' request should be) to mangaupdates. The [admin said read-only actions are not limited](https://www.mangaupdates.com/topic/4sw0ahm/-post/797126), but still, don't spam request to their site.

Remember to read the [Mangaupdates' Api Use Policy](https://api.mangaupdates.com/#section/Acceptable-Use-Policy).

This is a toy api, I have NOT extensively tested it, used at your own risk.

Installing:

```
git clone https://github.com/TheDucker1/mangaupdatescli.git
cd mangaupdatescli
curl https://api.mangaupdates.com/openapi.yaml -o openapi.yaml
go generate
go build .
```
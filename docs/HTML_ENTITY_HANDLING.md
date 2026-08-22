# HTML character references

XPath-Go follows browser-style HTML parsing for named and numeric character
references. XPath expressions evaluate the decoded DOM value, while source
locations continue to address the original response bytes.

## Text and attribute values

Given this source:

```html
<p title="Tom &amp; Jerry">A &lt; B &copy;</p>
```

the logical DOM contains:

- text: `A < B ©`
- `title`: `Tom & Jerry`

Queries therefore use decoded characters:

```go
source := `<p title="Tom &amp; Jerry">A &lt; B &copy;</p>`

results, err := xpath.Query(
    `//p[contains(text(), '<') and @title='Tom & Jerry']`,
    source,
)
if err != nil {
    log.Fatal(err)
}

fmt.Println(results[0].TextContent) // A < B ©
```

Do not pre-decode the complete HTML response. Decoding before parsing can turn
escaped markup into actual markup and changes source offsets.

## Source fidelity

Decoded DOM values and source fidelity are separate contracts:

- `Result.TextContent` and parsed attribute values contain decoded Unicode.
- `Result.StartLocation` and `Result.EndLocation` are byte offsets into the
  original input.
- In full-node mode, `Result.Value` is sliced from the original input and
  therefore retains the spelling used in the response, including references.

```go
source := `<p>A &amp; B</p>`
results, _ := xpath.Query(`//p`, source)

fmt.Println(results[0].TextContent) // A & B
fmt.Println(results[0].Value)       // <p>A &amp; B</p>
fmt.Println(source[
    results[0].StartLocation:results[0].EndLocation,
]) // <p>A &amp; B</p>
```

This distinction also applies to byte inputs decoded with `Options.Charset`:
XPath sees Unicode, while locations index the original encoded response.

## Browser recovery

The tokenizer covers the named and numeric reference behavior exercised by the
checked-in compatibility corpus, including semicolonless legacy references,
attribute-context restrictions, invalid numeric values, and references ending
at EOF. Unknown references remain literal text.

The project targets practical browser compatibility rather than claiming every
HTML Living Standard case. Add a browser-oracle fixture when extending this
area; see [Testing](TESTING.md) and [Compatibility](COMPATIBILITY.md).

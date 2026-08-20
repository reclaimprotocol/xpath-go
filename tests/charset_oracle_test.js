#!/usr/bin/env node

// Charset oracle for the eventual Go byte-oriented API. This deliberately
// does not invoke the Go implementation: it establishes jsdom/WHATWG's
// expected Unicode XPath behavior and keeps decoded DOM offsets separate from
// offsets into the original response bytes.

"use strict";

const assert = require("assert");
const { JSDOM } = require("jsdom");
const whatwgEncoding = require("whatwg-encoding");

function decodeByWhatwg(rawBytes, label) {
    const encoding = whatwgEncoding.labelToName(label);
    assert.ok(encoding, `WHATWG does not recognize charset label ${label}`);
    return whatwgEncoding.decode(rawBytes, encoding);
}

function evaluate(rawBytes, charset, expression) {
    const dom = new JSDOM(rawBytes, {
        contentType: `text/html; charset=${charset}`,
        includeNodeLocations: true
    });
    const { document, XPathResult } = dom.window;
    const result = document.evaluate(
        expression,
        document,
        null,
        XPathResult.ORDERED_NODE_SNAPSHOT_TYPE,
        null
    );
    const nodes = [];
    for (let i = 0; i < result.snapshotLength; i++) {
        const node = result.snapshotItem(i);
        nodes.push({
            node,
            location: dom.nodeLocation(node),
            textContent: node.textContent,
            title: node.getAttribute ? node.getAttribute("title") : null
        });
    }
    return { dom, nodes };
}

function byteIndex(rawBytes, needle) {
    const offset = rawBytes.indexOf(Buffer.from(needle));
    assert.notEqual(offset, -1, `raw fixture is missing ${JSON.stringify(needle)}`);
    return offset;
}

function locationPair(location) {
    return { startOffset: location.startOffset, endOffset: location.endOffset };
}

function testISO88591TextAndAttribute() {
    // 0xF1 and 0xE9 are valid single-byte ISO-8859-1 characters. The same
    // bytes must become U+00F1/U+00E9 before XPath evaluates text and attrs.
    const prefix = Buffer.from("<p>caf");
    const prefixTail = Buffer.from([0xE9, ...Buffer.from("</p>")]);
    const target = Buffer.from([
        ...Buffer.from('<div title="Se'), 0xF1,
        ...Buffer.from('or">Jos'), 0xE9,
        ...Buffer.from("</div>")
    ]);
    const rawBytes = Buffer.concat([prefix, prefixTail, target]);
    const expression = `//div[contains(text(), 'José') and @title='Señor']`;
    const { dom, nodes } = evaluate(rawBytes, "iso-8859-1", expression);

    assert.equal(nodes.length, 1, "ISO-8859-1 XPath should select the div");
    assert.equal(nodes[0].textContent, "José");
    assert.equal(nodes[0].title, "Señor");

    const decodedSource = decodeByWhatwg(rawBytes, "iso-8859-1");
    const decodedStart = decodedSource.indexOf('<div title="Señor">');
    assert.notEqual(decodedStart, -1);
    const decodedEnd = decodedStart + '<div title="Señor">José</div>'.length;
    const rawStart = byteIndex(rawBytes, '<div title="Se');
    const rawEnd = rawBytes.length;
    const expectedRaw = {
        startOffset: prefix.length + prefixTail.length,
        endOffset: rawBytes.length
    };

    // parse5/jsdom locations are offsets in the decoded JavaScript string;
    // the Go contract must instead return offsets into raw response bytes.
    assert.deepStrictEqual(locationPair(nodes[0].location), {
        startOffset: decodedStart,
        endOffset: decodedEnd
    });
    assert.deepStrictEqual({ startOffset: rawStart, endOffset: rawEnd }, expectedRaw);
    assert.equal(rawBytes.subarray(rawStart, rawEnd).toString("latin1"),
        '<div title="Señor">José</div>');
}

function testWindows1252AttributeAndTextLocation() {
    // 0x93/0x94 are curly quotes in Windows-1252. This expression catches
    // implementations that leave the bytes as replacement characters.
    const rawBytes = Buffer.from([
        ...Buffer.from('<div title="'), 0x93,
        ...Buffer.from("Hola"), 0x94,
        ...Buffer.from('">ok</div>')
    ]);
    const expression = `//div[@title='“Hola”']/text()`;
    const { dom, nodes } = evaluate(rawBytes, "windows-1252", expression);

    assert.equal(nodes.length, 1, "Windows-1252 XPath should select the text node");
    assert.equal(nodes[0].textContent, "ok");
    const rawStart = byteIndex(rawBytes, "ok");
    const rawEnd = rawStart + Buffer.byteLength("ok");
    const expectedRaw = { startOffset: 20, endOffset: 22 };
    const decodedSource = decodeByWhatwg(rawBytes, "windows-1252");
    const decodedStart = decodedSource.indexOf("ok");
    const decodedEnd = decodedStart + 2;

    assert.deepStrictEqual(locationPair(nodes[0].location), {
        startOffset: decodedStart,
        endOffset: decodedEnd
    });
    assert.deepStrictEqual({ decodedDOM: locationPair(nodes[0].location), rawBytes: { startOffset: rawStart, endOffset: rawEnd } }, {
        decodedDOM: { startOffset: decodedStart, endOffset: decodedEnd },
        rawBytes: expectedRaw
    });
}

function testWhatwgISO88591LabelUsesWindows1252Mapping() {
    // WHATWG aliases the ISO-8859-1 label to windows-1252. Keep this explicit
    // so a future oracle update does not silently change the expected C1-byte
    // behavior while the ñ/é case above remains ordinary Latin-1.
    const rawBytes = Buffer.from([ ...Buffer.from("<p>"), 0x80, ...Buffer.from("</p>") ]);
    const { nodes } = evaluate(rawBytes, "ISO-8859-1", "//p[text()='€']");
    assert.equal(nodes.length, 1, "WHATWG ISO-8859-1 label should decode 0x80 as U+20AC");
}

testISO88591TextAndAttribute();
testWindows1252AttributeAndTextLocation();
testWhatwgISO88591LabelUsesWindows1252Mapping();
console.log("charset oracle: 3 jsdom/WHATWG cases passed");

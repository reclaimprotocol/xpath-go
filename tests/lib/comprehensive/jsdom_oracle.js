'use strict';

const { JSDOM } = require('jsdom');
const whatwgEncoding = require('whatwg-encoding');

const SVG_NAMESPACE = 'http://www.w3.org/2000/svg';
const MATH_NAMESPACE = 'http://www.w3.org/1998/Math/MathML';

class JavaScriptOracle {
    evaluate(xpath, html, contentsOnly = false, testCase = null) {
        try {
            const charset = testCase && testCase.charset ? testCase.charset : null;
            const dom = new JSDOM(html, {
                contentType: charset ? `text/html; charset=${charset}` : 'text/html',
                includeNodeLocations: true
            });
            const document = dom.window.document;

            this.currentDom = dom;
            this.originalHTML = decodeOriginalHtml(html, charset);
            this.contentsOnly = contentsOnly;

            const xpathResult = document.evaluate(
                xpath,
                document,
                null,
                dom.window.XPathResult.ANY_TYPE,
                null
            );
            const results = this.collectResults(xpathResult, dom.window.XPathResult);
            applyExpectedRawLocations(results, testCase);
            return success(results);
        } catch (error) {
            return failure(error);
        }
    }

    collectResults(xpathResult, XPathResult) {
        const results = [];
        switch (xpathResult.resultType) {
            case XPathResult.ORDERED_NODE_ITERATOR_TYPE:
            case XPathResult.UNORDERED_NODE_ITERATOR_TYPE: {
                let node;
                while ((node = xpathResult.iterateNext())) {
                    results.push(this.nodeToResult(node));
                }
                break;
            }
            case XPathResult.FIRST_ORDERED_NODE_TYPE:
                if (xpathResult.singleNodeValue) {
                    results.push(this.nodeToResult(xpathResult.singleNodeValue));
                }
                break;
            case XPathResult.STRING_TYPE:
                results.push(stringResult(xpathResult.stringValue));
                break;
        }
        return results;
    }

    nodeToResult(node) {
        const location = this.currentDom.nodeLocation(node);
        let { startLocation, endLocation, value } = sourceLocation(
            node,
            location,
            this.originalHTML,
            this.contentsOnly
        );

        if (this.originalHTML) {
            startLocation = utf8Offset(this.originalHTML, startLocation);
            endLocation = utf8Offset(this.originalHTML, endLocation);
        }

        const namespaceURI = node.namespaceURI || '';
        const foreignOwner = node.nodeType === 2 && node.ownerElement &&
            (node.ownerElement.namespaceURI === SVG_NAMESPACE ||
             node.ownerElement.namespaceURI === MATH_NAMESPACE);
        const preservesCase = namespaceURI === SVG_NAMESPACE ||
            namespaceURI === MATH_NAMESPACE || foreignOwner;
        const nodeName = node.nodeName
            ? (preservesCase ? (node.name || node.nodeName) : node.nodeName.toLowerCase())
            : node.name;

        return {
            value,
            nodeName,
            nodeType: node.nodeType,
            namespaceURI,
            localName: node.localName || '',
            prefix: node.prefix || '',
            attributes: nodeAttributes(node),
            textContent: node.textContent || '',
            startLocation,
            endLocation,
            path: nodePath(node)
        };
    }
}

function decodeOriginalHtml(html, charset) {
    if (!Buffer.isBuffer(html)) return html;

    const encoding = whatwgEncoding.labelToName(charset || 'utf-8');
    if (!encoding) {
        throw new Error(`WHATWG does not recognize charset label ${charset}`);
    }
    return whatwgEncoding.decode(html, encoding);
}

function sourceLocation(node, location, originalHTML, contentsOnly) {
    let startLocation = 0;
    let endLocation = 0;
    let value = node.nodeValue || node.textContent || '';
    if (!location) return { startLocation, endLocation, value };

    if (contentsOnly) {
        startLocation = location.startTag ? location.startTag.endOffset : location.startOffset;
        endLocation = location.endTag ? location.endTag.startOffset : location.endOffset;
        value = node.textContent || '';
    } else {
        startLocation = location.startOffset || 0;
        endLocation = location.endOffset || startLocation;
        if (node.nodeType === 1 && originalHTML &&
            startLocation < originalHTML.length && endLocation <= originalHTML.length &&
            endLocation > startLocation) {
            value = originalHTML.substring(startLocation, endLocation);
        }
    }
    return { startLocation, endLocation, value };
}

function utf8Offset(html, offset) {
    const clampedOffset = Math.min(offset, html.length);
    return Buffer.byteLength(html.slice(0, clampedOffset), 'utf8');
}

function nodeAttributes(node) {
    const attributes = {};
    if (!node.attributes) return attributes;

    for (const attribute of node.attributes) {
        attributes[attribute.name] = attribute.value;
    }
    return attributes;
}

function nodePath(node) {
    const parts = [];
    let current = node;
    while (current && current.nodeType !== 9) {
        let name = current.namespaceURI === SVG_NAMESPACE
            ? current.nodeName
            : current.nodeName.toLowerCase();
        const position = siblingPosition(current);
        if (position > 1 || hasFollowingSiblingWithSameName(current)) {
            name += `[${position}]`;
        }
        parts.unshift(name);
        current = current.parentNode;
    }
    return `/${parts.join('/')}`;
}

function siblingPosition(node) {
    let position = 1;
    for (let sibling = node.previousSibling; sibling; sibling = sibling.previousSibling) {
        if (sibling.nodeName === node.nodeName) position++;
    }
    return position;
}

function hasFollowingSiblingWithSameName(node) {
    for (let sibling = node.nextSibling; sibling; sibling = sibling.nextSibling) {
        if (sibling.nodeName === node.nodeName) return true;
    }
    return false;
}

function applyExpectedRawLocations(results, testCase) {
    if (!testCase || !testCase.expectedRawLocations) return;
    if (testCase.expectedRawLocations.length !== results.length) {
        throw new Error(`Expected ${testCase.expectedRawLocations.length} raw locations, got ${results.length} jsdom results`);
    }
    results.forEach((result, index) => {
        result.startLocation = testCase.expectedRawLocations[index].start;
        result.endLocation = testCase.expectedRawLocations[index].end;
    });
}

function stringResult(value) {
    return {
        value,
        nodeName: '#text',
        nodeType: 3,
        attributes: {},
        textContent: value,
        startLocation: 0,
        endLocation: 0,
        path: '/string-result'
    };
}

function success(results) {
    return { success: true, results, count: results.length, error: null };
}

function failure(error) {
    return { success: false, results: [], count: 0, error: error.message };
}

module.exports = { JavaScriptOracle };

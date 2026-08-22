'use strict';

const SVG_NAMESPACE = 'http://www.w3.org/2000/svg';
const MATH_NAMESPACE = 'http://www.w3.org/1998/Math/MathML';

function mismatch(reason, jsResult, goResult) {
    return {
        match: false,
        reason,
        jsResults: jsResult.results,
        goResults: goResult.results
    };
}

function compareResults(jsResult, goResult) {
    if (!jsResult.success && !goResult.success) {
        return {
            match: jsResult.error === goResult.error,
            reason: jsResult.error === goResult.error ? 'Both failed with same error' : 'Different errors',
            jsResults: [],
            goResults: []
        };
    }

    if (jsResult.success !== goResult.success) {
        return mismatch(`Success mismatch: JS=${jsResult.success}, Go=${goResult.success}`, jsResult, goResult);
    }
    if (jsResult.count !== goResult.count) {
        return mismatch(`Count mismatch: JS=${jsResult.count}, Go=${goResult.count}`, jsResult, goResult);
    }

    for (let index = 0; index < jsResult.results.length; index++) {
        const jsNode = jsResult.results[index];
        const goNode = goResult.results[index];
        const difference = compareNode(jsNode, goNode, index);
        if (difference) return mismatch(difference, jsResult, goResult);
    }

    return {
        match: true,
        reason: 'Perfect match',
        jsResults: jsResult.results,
        goResults: goResult.results
    };
}

function compareNode(jsNode, goNode, index) {
    if (jsNode.nodeName !== goNode.nodeName) {
        return `Node name mismatch at index ${index}: JS="${jsNode.nodeName}", Go="${goNode.nodeName}"`;
    }

    const jsNamespace = jsNode.namespaceURI || '';
    const goNamespace = goNode.namespaceURI || '';
    const namespaceMatters = jsNode.nodeType === 2 || goNode.nodeType === 2 ||
        jsNamespace === SVG_NAMESPACE || goNamespace === SVG_NAMESPACE ||
        jsNamespace === MATH_NAMESPACE || goNamespace === MATH_NAMESPACE;
    if (namespaceMatters && jsNamespace !== goNamespace) {
        return `Namespace mismatch at index ${index}: JS="${jsNamespace}", Go="${goNamespace}"`;
    }

    const jsLocal = jsNode.localName || '';
    const goLocal = goNode.localName || '';
    const jsPrefix = jsNode.prefix || '';
    const goPrefix = goNode.prefix || '';
    if ((jsNode.nodeType === 2 || goNode.nodeType === 2) &&
        (jsLocal !== goLocal || jsPrefix !== goPrefix)) {
        return `Attribute identity mismatch at index ${index}: JS local/prefix="${jsLocal}"/"${jsPrefix}", Go="${goLocal}"/"${goPrefix}"`;
    }

    if (jsNode.textContent !== goNode.textContent) {
        return `Text content mismatch at index ${index}: JS="${jsNode.textContent}", Go="${goNode.textContent}"`;
    }
    if (jsNode.nodeType !== goNode.nodeType) {
        return `Node type mismatch at index ${index}: JS=${jsNode.nodeType}, Go=${goNode.nodeType}`;
    }
    if (jsNode.startLocation !== goNode.startLocation) {
        return `Start location mismatch at index ${index}: JS=${jsNode.startLocation}, Go=${goNode.startLocation}`;
    }
    if (jsNode.endLocation !== goNode.endLocation) {
        return `End location mismatch at index ${index}: JS=${jsNode.endLocation}, Go=${goNode.endLocation}`;
    }
    return null;
}

module.exports = { compareResults };

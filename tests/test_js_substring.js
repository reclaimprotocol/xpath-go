// Test what JavaScript returns for the substring+string-length case
const html = `<html><body><p>Some Text</p><p>Another Text</p><p>Short</p></body></html>`;
const xpath = `//p[substring(text(), string-length(text()) - 3) = 'Text']`;

// Parse HTML
const parser = new DOMParser();
const doc = parser.parseFromString(html, 'text/html');

// Evaluate XPath
const result = doc.evaluate(xpath, doc, null, XPathResult.ORDERED_NODE_SNAPSHOT_TYPE, null);

console.log(`JavaScript results: ${result.snapshotLength}`);
for (let i = 0; i < result.snapshotLength; i++) {
    const node = result.snapshotItem(i);
    console.log(`  [${i}] text: "${node.textContent}"`);
    
    // Let's debug the substring calculation
    const text = node.textContent;
    const length = text.length;
    const startPos = length - 3;
    const substring = text.substring(startPos);
    console.log(`    string-length("${text}") = ${length}`);
    console.log(`    substring("${text}", ${startPos+1}) = "${substring}"`);
    console.log(`    "${substring}" = "Text" ? ${substring === 'Text'}`);
}
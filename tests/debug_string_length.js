#!/usr/bin/env node

const { JSDOM } = require('jsdom');

function testStringLength() {
    const html = `<html><body><p>ShortText</p><p>VeryLongTextHere</p><p>Mid</p></body></html>`;
    const xpath = `//p[substring(text(), string-length(text()) - 3) = 'Text']`;
    
    console.log("=== DEBUGGING STRING-LENGTH WITH SUBSTRING ===\n");
    console.log(`XPath: ${xpath}`);
    console.log("Expected: 1 result (ShortText ends with 'Text')");
    
    const dom = new JSDOM(html);
    const document = dom.window.document;
    
    try {
        const result = document.evaluate(
            xpath,
            document,
            null,
            dom.window.XPathResult.ORDERED_NODE_SNAPSHOT_TYPE,
            null
        );
        
        console.log(`JavaScript found: ${result.snapshotLength} results`);
        
        for (let i = 0; i < result.snapshotLength; i++) {
            const node = result.snapshotItem(i);
            console.log(`  Result ${i + 1}: <${node.nodeName.toLowerCase()}>${node.textContent}</${node.nodeName.toLowerCase()}>`);
        }
        
        // Let's break down each text and check the logic
        console.log("\n--- Breaking down each paragraph ---");
        const allPs = document.evaluate(
            "//p",
            document,
            null,
            dom.window.XPathResult.ORDERED_NODE_SNAPSHOT_TYPE,
            null
        );
        
        for (let i = 0; i < allPs.snapshotLength; i++) {
            const p = allPs.snapshotItem(i);
            const text = p.textContent;
            const textLength = text.length;
            const xpathStartPos = textLength - 3; // XPath position (1-based)
            // XPath substring without length param means "from start to end"
            // Since XPath is 1-based, we need to convert to 0-based for JS
            const jsStartPos = Math.max(0, xpathStartPos - 1); 
            const xpathSubstring = text.substring(jsStartPos); // From position to end
            
            console.log(`P${i+1}: "${text}"`);
            console.log(`  Length: ${textLength}`);
            console.log(`  XPath expression: string-length(text()) - 3 = ${textLength} - 3 = ${xpathStartPos}`);
            console.log(`  XPath substring(text(), ${xpathStartPos}): "${xpathSubstring}"`);
            console.log(`  Equals 'Text'? ${xpathSubstring === 'Text'}`);
            console.log();
        }
        
    } catch (error) {
        console.error(`JavaScript XPath error: ${error.message}`);
    }
}

testStringLength();
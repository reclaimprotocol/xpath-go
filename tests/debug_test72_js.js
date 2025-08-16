#!/usr/bin/env node

const { JSDOM } = require('jsdom');

function testSubstringWithStringLength() {
    const html = `<html><body><p>ShortText</p><p>VeryLongTextHere</p><p>Mid</p></body></html>`;
    const xpath = `//p[substring(text(), string-length(text()) - 3) = 'Text']`;
    
    console.log("=== JAVASCRIPT REFERENCE TEST ===\n");
    console.log(`XPath: ${xpath}`);
    
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
            console.log(`  Result ${i + 1}: '${node.textContent}'`);
        }
        
        // Test each paragraph individually
        console.log("\n--- Testing each paragraph individually ---");
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
            
            console.log(`\nP${i+1}: '${text}' (length: ${text.length})`);
            
            // Test the individual components
            const stringLengthResult = document.evaluate(
                "string-length(text())",
                p,
                null,
                dom.window.XPathResult.NUMBER_TYPE,
                null
            );
            
            const startPos = stringLengthResult.numberValue - 3;
            console.log(`  string-length(text()) - 3 = ${stringLengthResult.numberValue} - 3 = ${startPos}`);
            
            const substringResult = document.evaluate(
                `substring(text(), ${startPos})`,
                p,
                null,
                dom.window.XPathResult.STRING_TYPE,
                null
            );
            
            console.log(`  substring(text(), ${startPos}) = '${substringResult.stringValue}'`);
            console.log(`  '${substringResult.stringValue}' = 'Text'? ${substringResult.stringValue === 'Text'}`);
        }
        
    } catch (error) {
        console.error(`JavaScript XPath error: ${error.message}`);
    }
}

testSubstringWithStringLength();